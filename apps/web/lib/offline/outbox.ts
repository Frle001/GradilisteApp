import { getDB } from './db'
import type { OutboxEntry, OutboxEntryType } from './types'

// ── Write ─────────────────────────────────────────────────────────────────────

export async function enqueue(
  entry: Omit<OutboxEntry, 'status' | 'attempts' | 'createdAt'>
): Promise<void> {
  const db = await getDB()
  await db.put('outbox', {
    ...entry,
    status: 'pending',
    attempts: 0,
    createdAt: Date.now(),
  })
}

export async function enqueueBlob(key: string, file: File): Promise<void> {
  const db = await getDB()
  await db.put('blobs', file, key)
}

export async function markSynced(id: string, resolvedId?: string): Promise<void> {
  const db = await getDB()
  const tx = db.transaction('outbox', 'readwrite')
  const entry = await tx.objectStore('outbox').get(id)
  if (entry) {
    await tx.objectStore('outbox').put({
      ...entry,
      status: 'synced',
      resolvedId: resolvedId ?? entry.resolvedId,
    })
  }
  await tx.done
}

/**
 * Keep as pending for retry (transient errors: network, 5xx, timeout).
 */
export async function resetForRetry(id: string, error: string): Promise<void> {
  const db = await getDB()
  const tx = db.transaction('outbox', 'readwrite')
  const entry = await tx.objectStore('outbox').get(id)
  if (entry) {
    await tx.objectStore('outbox').put({
      ...entry,
      status: 'pending',
      attempts: entry.attempts + 1,
      lastAttemptAt: Date.now(),
      lastError: error,
    })
  }
  await tx.done
}

/**
 * Mark as permanently failed (4xx except 408/429 — user action required).
 */
export async function markFailed(id: string, error: string): Promise<void> {
  const db = await getDB()
  const tx = db.transaction('outbox', 'readwrite')
  const entry = await tx.objectStore('outbox').get(id)
  if (entry) {
    await tx.objectStore('outbox').put({
      ...entry,
      status: 'failed',
      attempts: entry.attempts + 1,
      lastAttemptAt: Date.now(),
      lastError: error,
    })
  }
  await tx.done
}

/**
 * Propagate the server-assigned report ID to all pending photo entries
 * that reference parentId, so they can build the upload URL.
 */
export async function propagateResolvedId(parentId: string, resolvedId: string): Promise<void> {
  const db = await getDB()
  const tx = db.transaction('outbox', 'readwrite')
  const store = tx.objectStore('outbox')
  const photoEntries = await store.index('by-parent').getAll(parentId)
  for (const photo of photoEntries) {
    if (photo.status === 'pending') {
      await store.put({ ...photo, resolvedId })
    }
  }
  await tx.done
}

// ── Read ──────────────────────────────────────────────────────────────────────

export async function getEntry(id: string): Promise<OutboxEntry | undefined> {
  const db = await getDB()
  return db.get('outbox', id)
}

export async function getBlob(key: string): Promise<File | undefined> {
  const db = await getDB()
  return db.get('blobs', key)
}

export async function getPendingEntries(): Promise<OutboxEntry[]> {
  const db = await getDB()
  return db.getAllFromIndex('outbox', 'by-status', 'pending')
}

export async function getPendingCount(): Promise<number> {
  const db = await getDB()
  return db.countFromIndex('outbox', 'by-status', 'pending')
}

export async function getAllEntries(): Promise<OutboxEntry[]> {
  const db = await getDB()
  return db.getAll('outbox')
}

export async function getEntriesByType(type: OutboxEntryType): Promise<OutboxEntry[]> {
  const db = await getDB()
  return db.getAllFromIndex('outbox', 'by-type', type)
}

/**
 * Find the first pending entry of a given type matching a predicate.
 * Used to locate an existing pending entry so its ID can be reused on retry.
 */
export async function findPendingEntry(
  type: OutboxEntryType,
  predicate: (entry: OutboxEntry) => boolean,
): Promise<OutboxEntry | undefined> {
  const db = await getDB()
  const entries = await db.getAllFromIndex('outbox', 'by-type', type)
  return entries.find(e => e.status === 'pending' && predicate(e))
}

// ── Cleanup ───────────────────────────────────────────────────────────────────

export async function deleteEntry(id: string): Promise<void> {
  const db = await getDB()
  await db.delete('outbox', id)
}

export async function deleteBlob(key: string): Promise<void> {
  const db = await getDB()
  await db.delete('blobs', key)
}

export async function getFailedEntries(): Promise<OutboxEntry[]> {
  const db = await getDB()
  return db.getAllFromIndex('outbox', 'by-status', 'failed')
}

export async function getFailedCount(): Promise<number> {
  const db = await getDB()
  return db.countFromIndex('outbox', 'by-status', 'failed')
}

/**
 * Permanently remove an entry (and its blob if any).
 * Used when the user explicitly discards a failed item.
 */
export async function discardEntry(id: string): Promise<void> {
  const db = await getDB()
  const tx = db.transaction(['outbox', 'blobs'], 'readwrite')
  const entry = await tx.objectStore('outbox').get(id)
  if (entry?.blobKey) await tx.objectStore('blobs').delete(entry.blobKey)
  await tx.objectStore('outbox').delete(id)
  await tx.done
}

/**
 * Reset a failed entry back to pending so it will be retried on next sync.
 */
export async function requeueFailed(id: string): Promise<void> {
  const db = await getDB()
  const tx = db.transaction('outbox', 'readwrite')
  const entry = await tx.objectStore('outbox').get(id)
  if (entry && entry.status === 'failed') {
    await tx.objectStore('outbox').put({ ...entry, status: 'pending', lastError: undefined })
  }
  await tx.done
}

/** Count of pending entries per type. */
export async function getPendingCountsByType(): Promise<Record<OutboxEntryType, number>> {
  const db = await getDB()
  const entries = await db.getAllFromIndex('outbox', 'by-status', 'pending')
  const counts: Record<OutboxEntryType, number> = {
    'daily-report': 0,
    'worker-hours': 0,
    'photo-upload': 0,
  }
  for (const e of entries) {
    counts[e.type] = (counts[e.type] ?? 0) + 1
  }
  return counts
}

/** Remove all synced entries older than `maxAgeMs` to keep the store tidy. */
export async function pruneSynced(maxAgeMs = 7 * 24 * 60 * 60 * 1000): Promise<void> {
  const db = await getDB()
  const all = await db.getAll('outbox')
  const cutoff = Date.now() - maxAgeMs
  const tx = db.transaction(['outbox', 'blobs'], 'readwrite')
  for (const entry of all) {
    if (entry.status === 'synced' && entry.createdAt < cutoff) {
      await tx.objectStore('outbox').delete(entry.id)
      if (entry.blobKey) {
        await tx.objectStore('blobs').delete(entry.blobKey)
      }
    }
  }
  await tx.done
}
