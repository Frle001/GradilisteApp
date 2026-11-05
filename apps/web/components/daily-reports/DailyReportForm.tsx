'use client'

import { useState, useEffect, useRef } from 'react'
import Image from 'next/image'
import { useRouter } from 'next/navigation'
import {
  type DailyReportFormData,
  type DailyReportDetail,
  type ActivityInputUI,
  type CreateDailyReportPayload,
  type FormDataMaterial,
} from '@/lib/types/daily-reports'
import ActivityEntryForm from './ActivityEntryForm'
import ActivityListPreview from './ActivityListPreview'
import apiClient from '@/lib/api-client'
import { useFormDraft } from '@/hooks/useFormDraft'

const MAX_PHOTO_SIZE = 10 * 1024 * 1024 // 10 MB — mirrors backend MaxReceiptFileSize
const MAX_PHOTOS = 100

const ALLOWED_PHOTO_TYPES: Record<string, true> = {
  'image/jpeg': true,
  'image/png': true,
  'image/webp': true,
}

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}

function validatePhotoFile(file: File): string | null {
  const mime = file.type.split(';')[0].trim().toLowerCase()
  if (mime === 'image/heic' || mime === 'image/heif') {
    return `${file.name}: HEIC/HEIF format nije podržan; koristite JPEG, PNG ili WebP`
  }
  if (!ALLOWED_PHOTO_TYPES[mime]) {
    return `${file.name}: Nepodržani format; dopušteni su JPEG, PNG i WebP`
  }
  if (file.size > MAX_PHOTO_SIZE) {
    return `${file.name}: Datoteka je prevelika (maks. 10 MB)`
  }
  return null
}

interface Props {
  formData: DailyReportFormData
  /** When provided, the form is in edit mode and pre-fills from this report. */
  existing?: DailyReportDetail
  /** Pre-select a project when entering the form from a project context. */
  initialProjectId?: string
  onSubmit: (payload: CreateDailyReportPayload) => Promise<{ id: string } | void>
  submitLabel?: string
  /** Controlled pending photos (create mode only). If omitted, form manages state internally. */
  pendingPhotos?: File[]
  /** Called whenever the pending photo list changes. */
  onPendingPhotosChange?: (files: File[]) => void
}

function localTodayStr(): string {
  return new Date().toLocaleDateString('sv') // YYYY-MM-DD in browser local time
}

const inputCls = 'w-full bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2 text-sm text-gi-text placeholder:text-gi-text-subtle focus:outline-none focus:border-gi-yellow transition-colors disabled:opacity-50'
const labelCls = 'block text-xs font-medium text-gi-text-muted uppercase tracking-wider mb-1.5'

export default function DailyReportForm({
  formData,
  existing,
  initialProjectId,
  onSubmit,
  submitLabel = 'Spremi izvještaj',
  pendingPhotos,
  onPendingPhotosChange,
}: Props) {
  const router = useRouter()
  const isEditMode = !!existing
  const existingRef = useRef(existing)

  // Draft preservation for create mode only (reportDate excluded — always today)
  const [draftFields, setDraftFields, clearDraft] = useFormDraft(
    'daily-report-new',
    { projectId: '', notes: '' }
  )

  const [projectId, setProjectId] = useState(
    isEditMode ? existing!.project.id : (initialProjectId || draftFields.projectId)
  )
  const [reportDate] = useState(
    isEditMode ? existing!.report_date : localTodayStr()
  )
  const [notes, setNotes] = useState(
    isEditMode ? (existing!.notes ?? '') : draftFields.notes
  )

  // Sync to draft whenever fields change (create mode only)
  useEffect(() => {
    if (!isEditMode) {
      setDraftFields({ projectId, notes })
    }
  }, [projectId, notes, isEditMode, setDraftFields])

  const [activities, setActivities] = useState<ActivityInputUI[]>(() => {
    if (!existing) return []
    return existing.activities.map(a => ({
      _tempId: crypto.randomUUID(),
      _displayName: a.custom_material_name ?? a.material_name,
      project_material_id: a.project_material_id,
      custom_material_name: a.custom_material_name,
      quantity: a.quantity,
      unit: a.unit,
      activity_type: a.activity_type,
      is_vtk: a.is_vtk,
      notes: a.notes,
      tracking_type: a.tracking_type,
    }))
  })

  const [editingActivity, setEditingActivity] = useState<ActivityInputUI | null>(null)

  const [submitting, setSubmitting] = useState(false)
  const [serverError, setServerError] = useState<string | null>(null)
  const [uploadProgress, setUploadProgress] = useState<string | null>(null)

  const [projectMaterials, setProjectMaterials] = useState<FormDataMaterial[]>([])
  const [materialsLoading, setMaterialsLoading] = useState(false)

  // ── Pending photos (create mode only) ──────────────────────────────────────
  const [internalPhotos, setInternalPhotos] = useState<File[]>([])
  const photos = pendingPhotos ?? internalPhotos
  const photoInputRef = useRef<HTMLInputElement>(null)
  const [photoErrors, setPhotoErrors] = useState<string[]>([])

  // Object URLs keyed by File reference. Revoked on removal, after upload, and on unmount.
  const photoUrlsRef = useRef<Map<File, string>>(new Map())

  function getPhotoUrl(file: File): string {
    if (!photoUrlsRef.current.has(file)) {
      photoUrlsRef.current.set(file, URL.createObjectURL(file))
    }
    return photoUrlsRef.current.get(file)!
  }

  function revokePhotoUrl(file: File) {
    const url = photoUrlsRef.current.get(file)
    if (url) {
      URL.revokeObjectURL(url)
      photoUrlsRef.current.delete(file)
    }
  }

  useEffect(() => {
    const urlsRef = photoUrlsRef.current
    return () => {
      urlsRef.forEach(url => URL.revokeObjectURL(url))
      urlsRef.clear()
    }
  }, [])

  function updatePhotos(files: File[]) {
    if (pendingPhotos === undefined) {
      setInternalPhotos(files)
    }
    onPendingPhotosChange?.(files)
  }

  function handlePhotoSelect(e: React.ChangeEvent<HTMLInputElement>) {
    const selected = Array.from(e.target.files ?? [])
    e.target.value = '' // allow re-selecting the same files later

    const errors: string[] = []
    const valid: File[] = []

    for (const file of selected) {
      const err = validatePhotoFile(file)
      if (err) errors.push(err)
      else valid.push(file)
    }

    const available = MAX_PHOTOS - photos.length
    const toAdd = valid.slice(0, available)
    if (valid.length > available) {
      errors.push(`Možete dodati najviše ${MAX_PHOTOS} fotografija.`)
    }

    setPhotoErrors(errors)
    if (toAdd.length > 0) {
      updatePhotos([...photos, ...toAdd])
    }
  }

  function removePhoto(file: File) {
    revokePhotoUrl(file)
    updatePhotos(photos.filter(f => f !== file))
  }

  // ── Project materials ───────────────────────────────────────────────────────
  useEffect(() => {
    if (!projectId) {
      setProjectMaterials([])
      return
    }
    setMaterialsLoading(true)
    setProjectMaterials([])
    if (!existingRef.current) {
      setActivities([])
    }
    apiClient
      .get(`/projects/${projectId}/materials`)
      .then(res => setProjectMaterials(res.data.materials ?? []))
      .catch(() => setProjectMaterials([]))
      .finally(() => setMaterialsLoading(false))
  }, [projectId])

  function addActivity(a: ActivityInputUI) {
    setActivities(prev => [...prev, a])
    setEditingActivity(null)
  }

  function removeActivity(tempId: string) {
    setActivities(prev => prev.filter(a => a._tempId !== tempId))
  }

  function handleEditActivity(activity: ActivityInputUI) {
    setActivities(prev => prev.filter(a => a._tempId !== activity._tempId))
    setEditingActivity(activity)
  }

  function validate(): string[] {
    const errs: string[] = []
    if (!projectId) errs.push('Odaberite projekt.')
    if (!reportDate) errs.push('Datum izvještaja je obavezan.')
    if (activities.length === 0) errs.push('Dodajte barem jednu aktivnost.')
    return errs
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setServerError(null)
    const errs = validate()
    if (errs.length) { setServerError(errs.join(' ')); return }

    const payload: CreateDailyReportPayload = {
      project_id: projectId,
      report_date: reportDate,
      notes: notes.trim() || null,
      worker_hours: [],
      activities: activities.map(a => ({
        project_material_id: a.project_material_id,
        custom_material_name: a.custom_material_name,
        quantity: a.quantity,
        unit: a.unit,
        activity_type: a.activity_type,
        is_vtk: a.is_vtk,
        notes: a.notes,
      })),
    }

    setSubmitting(true)
    setUploadProgress(null)
    try {
      const result = await onSubmit(payload)
      clearDraft()

      // Upload pending photos if this was a create and photos were selected.
      let failCount = 0
      if (!isEditMode && photos.length > 0 && result && 'id' in result) {
        for (let i = 0; i < photos.length; i++) {
          setUploadProgress(`${i + 1}/${photos.length}`)
          const form = new FormData()
          form.append('photo', photos[i])
          try {
            await apiClient.post(
              `/daily-reports/${result.id}/attachments`,
              form,
              { headers: { 'Content-Type': 'multipart/form-data' } },
            )
          } catch {
            failCount++
          }
        }
        // Release all object URLs now that upload is complete.
        photos.forEach(revokePhotoUrl)
        updatePhotos([])
      }

      if (result && 'id' in result) {
        const dest = failCount > 0
          ? `/dashboard/daily-reports/${result.id}?upload_errors=${failCount}`
          : `/dashboard/daily-reports/${result.id}`
        router.push(dest)
      } else {
        router.push('/dashboard/daily-reports')
      }
    } catch (err: unknown) {
      const backendMsg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
      setServerError(backendMsg ?? (err instanceof Error ? err.message : 'Greška pri spremanju.'))
      // Re-fetch materials so the dropdown reflects current server state (e.g. a zero-stock
      // material disappears without the user having to manually reload).
      if (projectId) {
        apiClient
          .get(`/projects/${projectId}/materials`)
          .then(res => setProjectMaterials(res.data.materials ?? []))
          .catch(() => {})
      }
    } finally {
      setSubmitting(false)
      setUploadProgress(null)
    }
  }

  const submitButtonLabel = uploadProgress
    ? `Prijenos fotografija (${uploadProgress})…`
    : submitting
      ? 'Spremanje…'
      : submitLabel

  const selectedProject = formData.projects.find(p => p.id === projectId)

  return (
    <form onSubmit={handleSubmit}>
      <div className="grid grid-cols-1 xl:grid-cols-[minmax(0,1fr)_300px] xl:gap-8 xl:items-start space-y-6 xl:space-y-0">

        {/* ── Main column ──────────────────────────────────────────────────── */}
        <div className="space-y-6">

          {/* ── Section 1: Osnovni podaci ─────────────────────────────────── */}
          <section className="bg-gi-card border border-gi-border rounded-xl p-5 space-y-4">
            <div>
              <h2 className="text-xs font-medium text-gi-text-subtle uppercase tracking-wider">Osnovni podaci</h2>
              <p className="text-xs text-gi-text-subtle/60 mt-0.5">Projekt, datum i opcionalne napomene.</p>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className={labelCls}>Projekt *</label>
                <select
                  value={projectId}
                  onChange={e => setProjectId(e.target.value)}
                  disabled={submitting || !!existing}
                  className={inputCls}
                >
                  <option value="">Odaberite projekt</option>
                  {formData.projects.map(p => (
                    <option key={p.id} value={p.id}>{p.name}</option>
                  ))}
                </select>
              </div>

              <div>
                <label className={labelCls}>Datum *</label>
                <input
                  type="date"
                  value={reportDate}
                  onChange={() => {}}
                  disabled={submitting || !!existing || !isEditMode}
                  className={inputCls}
                />
                {!isEditMode && (
                  <p className="text-gi-text-subtle text-xs mt-1">Dnevni izvještaj se može kreirati samo za današnji datum.</p>
                )}
              </div>

              <div className="sm:col-span-2">
                <label className={labelCls}>Napomena</label>
                <textarea
                  value={notes}
                  onChange={e => setNotes(e.target.value)}
                  disabled={submitting}
                  rows={2}
                  placeholder="Opcionalne napomene o radnom danu…"
                  className={inputCls + ' resize-none'}
                />
              </div>
            </div>
          </section>

          {/* ── Worker hours info banner ───────────────────────────────────── */}
          <div className="flex items-start gap-3 bg-blue-950/30 border border-blue-800/40 rounded-xl px-4 py-3">
            <svg className="w-4 h-4 text-blue-400 mt-0.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <p className="text-sm text-blue-300">
              Radnici sada samostalno unose svoje radne sate kroz modul <span className="font-medium text-blue-200">Moji sati</span>.
            </p>
          </div>

          {/* ── Section 2: Aktivnosti ──────────────────────────────────────── */}
          <section className="bg-gi-card border border-gi-border rounded-xl p-5 space-y-4">
            <div>
              <div className="flex items-center justify-between gap-2">
                <h2 className="text-xs font-medium text-gi-text-subtle uppercase tracking-wider">Aktivnosti</h2>
                {activities.length > 0 && (
                  <span className="text-xs text-gi-text-subtle bg-gi-card-elevated border border-gi-border px-2 py-0.5 rounded-full">
                    {activities.length}
                  </span>
                )}
              </div>
              <p className="text-xs text-gi-text-subtle/60 mt-0.5">Materijali ugrađeni ili skinuti s gradilišta.</p>
            </div>

            <ActivityListPreview
              activities={activities}
              onRemove={removeActivity}
              onEdit={!submitting ? handleEditActivity : undefined}
              disabled={submitting}
            />

            <ActivityEntryForm
              key={editingActivity?._tempId ?? projectId}
              initialActivity={editingActivity ?? undefined}
              materials={projectMaterials}
              materialsLoading={materialsLoading}
              onAdd={addActivity}
              disabled={submitting || !projectId}
              projectId={projectId || undefined}
              onMaterialAdded={m => setProjectMaterials(prev => [...prev, m])}
            />
          </section>

          {/* ── Section 3: Fotografije (create mode only) ─────────────────── */}
          {!isEditMode && (
            <section className="bg-gi-card border border-gi-border rounded-xl p-5 space-y-4">
              <div>
                <div className="flex items-center justify-between gap-2">
                  <h2 className="text-xs font-medium text-gi-text-subtle uppercase tracking-wider">Fotografije s gradilišta</h2>
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-gi-text-subtle">{photos.length} / {MAX_PHOTOS}</span>
                    {photos.length < MAX_PHOTOS && (
                      <button
                        type="button"
                        disabled={submitting}
                        onClick={() => photoInputRef.current?.click()}
                        className="flex items-center gap-1.5 px-3 py-1.5 bg-gi-card-elevated hover:bg-gi-border border border-gi-border text-gi-text-muted text-xs font-medium rounded-lg transition disabled:opacity-50"
                      >
                        <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                          <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
                        </svg>
                        Dodaj fotografije
                      </button>
                    )}
                  </div>
                </div>
                <p className="text-xs text-gi-text-subtle/60 mt-0.5">JPEG, PNG ili WebP do 10 MB po slici.</p>
              </div>

              <input
                ref={photoInputRef}
                type="file"
                accept="image/jpeg,image/png,image/webp"
                multiple
                className="hidden"
                onChange={handlePhotoSelect}
              />

              {photoErrors.length > 0 && (
                <div className="space-y-1">
                  {photoErrors.map((err, i) => (
                    <p key={i} className="text-xs text-amber-400">{err}</p>
                  ))}
                </div>
              )}

              {photos.length === 0 ? (
                <button
                  type="button"
                  disabled={submitting}
                  onClick={() => photoInputRef.current?.click()}
                  className="w-full flex flex-col items-center gap-2 py-6 border border-dashed border-gi-border rounded-lg text-gi-text-subtle hover:border-gi-border/80 hover:text-gi-text-muted transition disabled:pointer-events-none"
                >
                  <svg className="w-7 h-7" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M6.827 6.175A2.31 2.31 0 015.186 7.23c-.38.054-.757.112-1.134.175C2.999 7.58 2.25 8.507 2.25 9.574V18a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9.574c0-1.067-.75-1.994-1.802-2.169a47.865 47.865 0 00-1.134-.175 2.31 2.31 0 01-1.64-1.055l-.822-1.316a2.192 2.192 0 00-1.736-1.039 48.774 48.774 0 00-5.232 0 2.192 2.192 0 00-1.736 1.039l-.821 1.316z" />
                    <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 12.75a4.5 4.5 0 11-9 0 4.5 4.5 0 019 0zM18.75 10.5h.008v.008h-.008V10.5z" />
                  </svg>
                  <span className="text-xs">Kliknite za dodavanje fotografija s gradilišta</span>
                  <span className="text-xs text-gi-text-subtle">JPEG, PNG, WebP &middot; maks. 10 MB po slici</span>
                </button>
              ) : (
                <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
                  {photos.map((file, idx) => (
                    <div
                      key={idx}
                      className="relative aspect-square group rounded-lg overflow-hidden border border-gi-border bg-gi-card-elevated"
                    >
                      <Image
                        fill
                        src={getPhotoUrl(file)}
                        alt={file.name}
                        className="object-cover"
                        unoptimized
                      />
                      <button
                        type="button"
                        disabled={submitting}
                        onClick={() => removePhoto(file)}
                        className="absolute top-1 right-1 opacity-0 group-hover:opacity-100 transition-opacity bg-red-700 hover:bg-red-600 text-white rounded p-0.5 disabled:pointer-events-none"
                        title="Ukloni"
                      >
                        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                          <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                        </svg>
                      </button>
                      <div className="px-1.5 py-1 bg-gi-navy/80">
                        <p className="text-xs text-gi-text-muted truncate">{file.name}</p>
                        <p className="text-xs text-gi-text-subtle">{formatBytes(file.size)}</p>
                      </div>
                    </div>
                  ))}
                </div>
              )}

              {uploadProgress && (
                <div className="flex items-center gap-2 text-sm text-blue-300">
                  <svg className="w-4 h-4 animate-spin shrink-0" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
                  </svg>
                  Prijenos fotografija ({uploadProgress})…
                </div>
              )}
            </section>
          )}

          {/* ── Submit ────────────────────────────────────────────────────── */}
          {serverError && (
            <div className="bg-red-900/30 border border-red-800/60 rounded-lg px-4 py-3 text-sm text-red-300">
              {serverError}
            </div>
          )}

          <div className="flex flex-col-reverse sm:flex-row sm:items-center sm:justify-end gap-3">
            <button
              type="button"
              onClick={() => router.back()}
              disabled={submitting}
              className="w-full sm:w-auto px-4 py-3 sm:py-2.5 text-sm text-gi-text-muted hover:text-gi-text disabled:opacity-50 transition-colors text-center border border-gi-border rounded-lg hover:bg-gi-card-elevated"
            >
              Odustani
            </button>
            <button
              type="submit"
              disabled={submitting}
              className="w-full sm:w-auto px-5 py-3 sm:py-2.5 bg-gi-yellow hover:bg-gi-yellow-hover disabled:opacity-50 text-gi-navy text-sm font-semibold rounded-lg transition-colors"
            >
              {submitting && (
                <svg className="inline w-4 h-4 animate-spin mr-1.5 -mt-0.5" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
                </svg>
              )}
              {submitButtonLabel}
            </button>
          </div>
        </div>

        {/* ── Summary sidebar (wide desktop only) ─────────────────────────── */}
        <div className="hidden xl:block">
          <div className="xl:sticky xl:top-4 bg-gi-card border border-gi-border rounded-xl p-4 space-y-4">
            <h3 className="text-xs font-medium text-gi-text-subtle uppercase tracking-wider">Pregled</h3>

            <dl className="space-y-3">
              <div>
                <dt className="text-[10px] text-gi-text-subtle uppercase tracking-wider mb-0.5">Projekt</dt>
                <dd className="text-sm text-gi-text font-medium leading-snug">
                  {selectedProject ? selectedProject.name : (
                    <span className="text-gi-text-subtle italic">Nije odabran</span>
                  )}
                </dd>
              </div>

              <div>
                <dt className="text-[10px] text-gi-text-subtle uppercase tracking-wider mb-0.5">Datum</dt>
                <dd className="text-sm text-gi-text-muted">
                  {new Date(reportDate).toLocaleDateString('hr-HR', { day: '2-digit', month: '2-digit', year: 'numeric' })}
                </dd>
              </div>

              <div>
                <dt className="text-[10px] text-gi-text-subtle uppercase tracking-wider mb-0.5">Aktivnosti</dt>
                <dd className="flex items-center gap-2">
                  <span className={`text-sm font-semibold ${activities.length > 0 ? 'text-gi-yellow' : 'text-gi-text-subtle'}`}>
                    {activities.length}
                  </span>
                  {activities.length === 0 && (
                    <span className="text-xs text-gi-text-subtle">— dodajte barem jednu</span>
                  )}
                </dd>
              </div>

              {!isEditMode && (
                <div>
                  <dt className="text-[10px] text-gi-text-subtle uppercase tracking-wider mb-0.5">Fotografije</dt>
                  <dd className="text-sm text-gi-text-muted">{photos.length} / {MAX_PHOTOS}</dd>
                </div>
              )}

              {notes.trim() && (
                <div>
                  <dt className="text-[10px] text-gi-text-subtle uppercase tracking-wider mb-0.5">Napomena</dt>
                  <dd className="text-xs text-gi-text-muted line-clamp-3">{notes.trim()}</dd>
                </div>
              )}
            </dl>

            {!projectId && (
              <p className="text-xs text-amber-400/80 border-t border-gi-border pt-3">
                Odaberite projekt za pristup materijalima.
              </p>
            )}

            {projectId && activities.length === 0 && (
              <p className="text-xs text-amber-400/80 border-t border-gi-border pt-3">
                Dodajte barem jednu aktivnost prije slanja.
              </p>
            )}
          </div>
        </div>

      </div>
    </form>
  )
}
