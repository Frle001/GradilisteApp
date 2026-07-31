import apiClient from '@/lib/api-client'
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

/**
 * Returns true for errors that should NOT be retried (4xx except 408/429).
 * Network errors (no response) and 5xx are transient and will be retried.
 */
function isPermanentError(err: unknown): boolean {
  const status = (err as { response?: { status?: number } })?.response?.status
  if (status === undefined) return false // network error — transient
  if (status === 408 || status === 429) return false // timeout / rate-limit — transient
  return status >= 400 && status < 500
}

function errorMessage(err: unknown): string {
  const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
  if (msg) return msg
  const netMsg = (err as { message?: string })?.message
  return netMsg ?? 'Nepoznata greška'
}

// ── Individual entry sync ─────────────────────────────────────────────────────

async function syncDailyReport(entry: OutboxEntry): Promise<void> {
  const payload = entry.payload as DailyReportPayload
  try {
    const res = await apiClient.post('/daily-reports', {
      ...payload,
      client_submission_id: entry.id,
    })
    const reportId: string = res.data.id
    await markSynced(entry.id, reportId)
    // Let photo entries targeting this report know the server-assigned ID.
    await propagateResolvedId(entry.id, reportId)
  } catch (err) {
    if (isPermanentError(err)) {
      await markFailed(entry.id, errorMessage(err))
    } else {
      await resetForRetry(entry.id, errorMessage(err))
    }
  }
}

async function syncWorkerHours(entry: OutboxEntry): Promise<void> {
  const payload = entry.payload as WorkerHoursPayload
  try {
    await apiClient.post('/worker-hours', {
      ...payload,
      client_submission_id: entry.id,
    })
    await markSynced(entry.id)
  } catch (err) {
    if (isPermanentError(err)) {
      await markFailed(entry.id, errorMessage(err))
    } else {
      await resetForRetry(entry.id, errorMessage(err))
    }
  }
}

async function syncPhotoUpload(entry: OutboxEntry): Promise<void> {
  // The resolvedId is the server-assigned daily-report UUID.
  if (!entry.resolvedId) return // parent not yet synced — skip this cycle

  const file = entry.blobKey ? await getBlob(entry.blobKey) : undefined
  if (!file) {
    // Blob was lost (e.g. user cleared storage) — nothing we can do.
    await markFailed(entry.id, 'Blob nedostupan')
    return
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
  } catch (err) {
    if (isPermanentError(err)) {
      await markFailed(entry.id, errorMessage(err))
      if (entry.blobKey) await deleteBlob(entry.blobKey)
    } else {
      await resetForRetry(entry.id, errorMessage(err))
    }
  }
}

// ── Main sync loop ────────────────────────────────────────────────────────────

let _running = false

/**
 * Run one full pass over the pending outbox.
 *
 * Order matters: process non-photo entries first so photo entries have their
 * parent `resolvedId` populated before we attempt them.
 */
export async function runSync(): Promise<void> {
  if (_running) return
  _running = true

  try {
    await pruneSynced()

    const pending = await getPendingEntries()
    if (pending.length === 0) return

    // Phase 1: non-photo entries
    for (const entry of pending) {
      if (entry.type === 'photo-upload') continue
      switch (entry.type) {
        case 'daily-report':
          await syncDailyReport(entry)
          break
        case 'worker-hours':
          await syncWorkerHours(entry)
          break
      }
    }

    // Phase 2: photo entries (resolvedId may have been written in phase 1)
    for (const entry of pending) {
      if (entry.type !== 'photo-upload') continue
      await syncPhotoUpload(entry)
    }
  } finally {
    _running = false
  }
}

/**
 * Single-entry sync: attempt to submit one outbox entry immediately and
 * return the resolved server ID on success, or null on failure.
 */
export async function trySyncEntry(entryId: string): Promise<string | null> {
  const { getEntry } = await import('./outbox')
  const entry = await getEntry(entryId)
  if (!entry || entry.status !== 'pending') return null

  switch (entry.type) {
    case 'daily-report':
      await syncDailyReport(entry)
      break
    case 'worker-hours':
      await syncWorkerHours(entry)
      break
    case 'photo-upload':
      await syncPhotoUpload(entry)
      break
  }

  const updated = await getEntry(entryId)
  return updated?.resolvedId ?? null
}
