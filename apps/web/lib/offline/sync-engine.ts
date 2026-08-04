import apiClient from '@/lib/api-client'
import { broadcastRefresh, type RefreshDomain } from '@/lib/refresh-events'
import {
  getPendingEntries,
  getBlob,
  markSynced,
  markFailed,
  resetForRetry,
  propagateResolvedId,
  deleteBlob,
  pruneSynced,
} from './outbox'
import type { OutboxEntry, DailyReportPayload, WorkerHoursPayload } from './types'

// ── Error classification ──────────────────────────────────────────────────────

function isPermanentError(err: unknown): boolean {
  const status = (err as { response?: { status?: number } })?.response?.status
  if (status === undefined) return false
  if (status === 408 || status === 429) return false
  return status >= 400 && status < 500
}

function errorMessage(err: unknown): string {
  const serverMsg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
  if (serverMsg) return serverMsg
  const raw = ((err as { message?: string })?.message ?? '').toLowerCase()
  if (!raw || raw.includes('network') || raw.includes('failed to fetch') || raw.includes('load failed')) {
    return 'Nema veze s internetom'
  }
  if (raw.includes('timeout') || raw.includes('econnrefused')) {
    return 'Zahtjev je prekoračio vremenski rok'
  }
  return 'Greška pri slanju podataka'
}

// ── Individual entry sync ─────────────────────────────────────────────────────

async function syncDailyReport(entry: OutboxEntry): Promise<RefreshDomain | null> {
  const payload = entry.payload as DailyReportPayload
  try {
    const res = await apiClient.post('/daily-reports', {
      ...payload,
      client_submission_id: entry.id,
    })
    const reportId: string = res.data.id
    await markSynced(entry.id, reportId)
    await propagateResolvedId(entry.id, reportId)
    return 'daily-reports'
  } catch (err) {
    if (isPermanentError(err)) {
      await markFailed(entry.id, errorMessage(err))
    } else {
      await resetForRetry(entry.id, errorMessage(err))
    }
    return null
  }
}

async function syncWorkerHours(entry: OutboxEntry): Promise<RefreshDomain | null> {
  const payload = entry.payload as WorkerHoursPayload
  try {
    await apiClient.post('/worker-hours', {
      ...payload,
      client_submission_id: entry.id,
    })
    await markSynced(entry.id)
    return 'worker-hours'
  } catch (err) {
    if (isPermanentError(err)) {
      await markFailed(entry.id, errorMessage(err))
    } else {
      await resetForRetry(entry.id, errorMessage(err))
    }
    return null
  }
}

async function syncPhotoUpload(entry: OutboxEntry): Promise<RefreshDomain | null> {
  if (!entry.resolvedId) return null

  const file = entry.blobKey ? await getBlob(entry.blobKey) : undefined
  if (!file) {
    await markFailed(entry.id, 'Blob nedostupan')
    return null
  }

  try {
    const form = new FormData()
    form.append('photo', file, entry.photoName ?? 'photo')
    await apiClient.post(
      `/daily-reports/${entry.resolvedId}/attachments`,
      form,
      { headers: { 'Content-Type': 'multipart/form-data' } }
    )
    await markSynced(entry.id)
    if (entry.blobKey) await deleteBlob(entry.blobKey)
    return 'daily-reports'
  } catch (err) {
    if (isPermanentError(err)) {
      await markFailed(entry.id, errorMessage(err))
      if (entry.blobKey) await deleteBlob(entry.blobKey)
    } else {
      await resetForRetry(entry.id, errorMessage(err))
    }
    return null
  }
}

// ── Core sync pass ────────────────────────────────────────────────────────────

interface SyncOptions {
  userId?: string
  companyId?: string
}

async function _doSync(opts: SyncOptions = {}): Promise<void> {
  await pruneSynced()

  const pending = await getPendingEntries()
  if (pending.length === 0) return

  // User isolation: skip entries queued by a different user/company.
  const filtered = opts.userId
    ? pending.filter(e => !e.ownerId || e.ownerId === opts.userId)
    : pending

  if (filtered.length === 0) return

  const refreshed = new Set<RefreshDomain>()

  // Phase 1: non-photo entries
  for (const entry of filtered) {
    if (entry.type === 'photo-upload') continue
    let domain: RefreshDomain | null = null
    switch (entry.type) {
      case 'daily-report':
        domain = await syncDailyReport(entry)
        break
      case 'worker-hours':
        domain = await syncWorkerHours(entry)
        break
    }
    if (domain) refreshed.add(domain)
  }

  // Phase 2: photo entries
  for (const entry of filtered) {
    if (entry.type !== 'photo-upload') continue
    const domain = await syncPhotoUpload(entry)
    if (domain) refreshed.add(domain)
  }

  if (refreshed.size > 0) {
    broadcastRefresh([...refreshed])
  }
}

// ── Multi-tab protection via Web Locks ────────────────────────────────────────

// Fallback flag for browsers without Web Locks API.
let _running = false

export async function runSync(opts: SyncOptions = {}): Promise<void> {
  if (typeof navigator !== 'undefined' && navigator.locks) {
    await navigator.locks.request(
      'gradiliste-sync',
      { ifAvailable: true },
      async (lock) => {
        if (!lock) return // another tab holds the lock
        await _doSync(opts)
      }
    )
    return
  }

  // Fallback: module-level flag (single-tab protection only)
  if (_running) return
  _running = true
  try {
    await _doSync(opts)
  } finally {
    _running = false
  }
}

// ── Single-entry sync ─────────────────────────────────────────────────────────

export async function trySyncEntry(entryId: string): Promise<string | null> {
  const { getEntry } = await import('./outbox')
  const entry = await getEntry(entryId)
  if (!entry || entry.status !== 'pending') return null

  let domain: RefreshDomain | null = null
  switch (entry.type) {
    case 'daily-report':
      domain = await syncDailyReport(entry)
      break
    case 'worker-hours':
      domain = await syncWorkerHours(entry)
      break
    case 'photo-upload':
      domain = await syncPhotoUpload(entry)
      break
  }

  if (domain) broadcastRefresh([domain])

  const updated = await getEntry(entryId)
  return updated?.resolvedId ?? null
}
