// ── Outbox ────────────────────────────────────────────────────────────────────

export type OutboxStatus = 'pending' | 'synced' | 'failed'

export type OutboxEntryType = 'daily-report' | 'worker-hours' | 'photo-upload'

/**
 * One item in the submission outbox.
 *
 * `id` doubles as the `client_submission_id` sent to the server for daily
 * reports, enabling idempotent retries.
 *
 * Photo uploads carry a `parentId` referencing the parent daily-report entry
 * and a `blobKey` pointing to the File stored in the IDB `blobs` store.
 * Once the parent report syncs, its server-assigned UUID is written into
 * `resolvedId` so photo entries can build the correct upload URL.
 */
export interface OutboxEntry {
  id: string
  type: OutboxEntryType
  status: OutboxStatus
  payload: unknown
  attempts: number
  createdAt: number
  lastAttemptAt?: number
  lastError?: string
  /** Server-assigned ID returned after a successful sync (e.g. daily-report UUID). */
  resolvedId?: string
  /** For photo-upload: outbox ID of the parent daily-report entry. */
  parentId?: string
  /** For photo-upload: key into the IDB `blobs` store. */
  blobKey?: string
  /** For photo-upload: original filename for display. */
  photoName?: string
  /** User who queued this entry — used to skip entries that belong to a different user during sync. */
  ownerId?: string
  /** Company the entry belongs to — guards against cross-company sync. */
  companyId?: string
}

// ── Worker-hours payload ──────────────────────────────────────────────────────

export interface WorkerHoursPayload {
  project_id: string
  work_date: string
  hours_worked: number
  notes: string | null
  work_description?: string | null
}

// ── Daily-report payload ──────────────────────────────────────────────────────

export interface ActivityInputPayload {
  project_material_id: string | null
  custom_material_name: string | null
  quantity: number
  unit: string
  activity_type: string
  is_vtk: boolean
  notes: string | null
}

export interface DailyReportPayload {
  project_id: string
  report_date: string
  notes: string | null
  worker_hours: []
  activities: ActivityInputPayload[]
}
