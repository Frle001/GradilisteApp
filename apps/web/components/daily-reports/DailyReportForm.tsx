'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import {
  type DailyReportFormData,
  type DailyReportDetail,
  type WorkerHoursEntry,
  type ActivityInputUI,
  type CreateDailyReportPayload,
  type FormDataMaterial,
} from '@/lib/types/daily-reports'
import WorkerHoursEditor from './WorkerHoursEditor'
import ActivityEntryForm from './ActivityEntryForm'
import ActivityListPreview from './ActivityListPreview'
import apiClient from '@/lib/api-client'
import { useFormDraft } from '@/hooks/useFormDraft'

interface Props {
  formData: DailyReportFormData
  /** When provided, the form is in edit mode and pre-fills from this report. */
  existing?: DailyReportDetail
  onSubmit: (payload: CreateDailyReportPayload) => Promise<{ id: string } | void>
  submitLabel?: string
}

function localTodayStr(): string {
  return new Date().toLocaleDateString('sv') // YYYY-MM-DD in browser local time
}

export default function DailyReportForm({ formData, existing, onSubmit, submitLabel = 'Spremi izvještaj' }: Props) {
  const router = useRouter()
  const isEditMode = !!existing

  // Draft preservation for create mode only (reportDate excluded — always today)
  const [draftFields, setDraftFields, clearDraft] = useFormDraft(
    'daily-report-new',
    { projectId: '', notes: '' }
  )

  const [projectId, setProjectId] = useState(
    isEditMode ? existing!.project.id : draftFields.projectId
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

  const [workerEntries, setWorkerEntries] = useState<WorkerHoursEntry[]>(() => {
    if (!existing) return []
    return existing.worker_hours.map(wh => ({
      worker_id: wh.worker_id,
      worker_name: wh.worker_name,
      hours: String(wh.hours_worked),
      notes: wh.notes ?? '',
    }))
  })

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
    }))
  })

  const [submitting, setSubmitting] = useState(false)
  const [serverError, setServerError] = useState<string | null>(null)

  const [projectMaterials, setProjectMaterials] = useState<FormDataMaterial[]>([])
  const [materialsLoading, setMaterialsLoading] = useState(false)

  // Fetch project materials whenever the selected project changes.
  // In edit mode the project select is disabled so this only runs once on mount.
  useEffect(() => {
    if (!projectId) {
      setProjectMaterials([])
      return
    }
    setMaterialsLoading(true)
    setProjectMaterials([])
    // Only in create mode: clear previously added activities — they referenced a different project's materials.
    if (!existing) {
      setActivities([])
    }
    apiClient
      .get(`/projects/${projectId}/materials`)
      .then(res => setProjectMaterials(res.data.materials ?? []))
      .catch(() => setProjectMaterials([]))
      .finally(() => setMaterialsLoading(false))
  }, [projectId]) // existing is intentionally omitted — it is stable for the component lifetime

  function addActivity(a: ActivityInputUI) {
    setActivities(prev => [...prev, a])
  }

  function removeActivity(tempId: string) {
    setActivities(prev => prev.filter(a => a._tempId !== tempId))
  }

  function validate(): string[] {
    const errs: string[] = []
    if (!projectId) errs.push('Odaberite projekt.')
    if (!reportDate) errs.push('Datum izvještaja je obavezan.')
    const hasHours = workerEntries.some(e => e.hours.trim() !== '')
    if (!hasHours) errs.push('Unesite sate za najmanje jednog radnika.')
    for (const e of workerEntries) {
      if (e.hours.trim() === '') continue
      const n = parseFloat(e.hours)
      if (isNaN(n) || n < 0 || n > 24) {
        errs.push(`Nevažeći sati za radnika ${e.worker_name}.`)
      }
    }
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
      worker_hours: workerEntries
        .filter(e => e.hours.trim() !== '')
        .map(e => ({
          worker_id: e.worker_id,
          hours_worked: parseFloat(e.hours),
          notes: e.notes.trim() || null,
        })),
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
    try {
      const result = await onSubmit(payload)
      clearDraft()
      if (result && 'id' in result) {
        router.push(`/dashboard/daily-reports/${result.id}`)
      } else {
        router.push('/dashboard/daily-reports')
      }
    } catch (err: unknown) {
      setServerError(err instanceof Error ? err.message : 'Greška pri spremanju.')
    } finally {
      setSubmitting(false)
    }
  }

  const workers = formData.workers

  return (
    <form onSubmit={handleSubmit} className="space-y-8">
      {/* ── Header fields ──────────────────────────────────────────────────── */}
      <section className="bg-slate-900 border border-slate-800 rounded-xl p-5 space-y-4">
        <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-wide">Osnovni podaci</h2>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="block text-xs text-slate-400 mb-1">Projekt *</label>
            <select
              value={projectId}
              onChange={e => setProjectId(e.target.value)}
              disabled={submitting || !!existing}
              className="w-full bg-slate-800 border border-slate-700 rounded px-3 py-2 text-sm text-slate-100 focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:opacity-50"
            >
              <option value="">— Odaberi projekt —</option>
              {formData.projects.map(p => (
                <option key={p.id} value={p.id}>{p.name}</option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-xs text-slate-400 mb-1">Datum *</label>
            <input
              type="date"
              value={reportDate}
              onChange={() => {}}
              disabled={submitting || !!existing || !isEditMode}
              className="w-full bg-slate-800 border border-slate-700 rounded px-3 py-2 text-sm text-slate-100 focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:opacity-50"
            />
            {!isEditMode && (
              <p className="text-slate-500 text-xs mt-1">Dnevni izvještaj se može kreirati samo za današnji datum.</p>
            )}
          </div>

          <div className="sm:col-span-2">
            <label className="block text-xs text-slate-400 mb-1">Napomena</label>
            <textarea
              value={notes}
              onChange={e => setNotes(e.target.value)}
              disabled={submitting}
              rows={2}
              placeholder="Opcionalne napomene o radnom danu…"
              className="w-full bg-slate-800 border border-slate-700 rounded px-3 py-2 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-blue-500 resize-none disabled:opacity-50"
            />
          </div>
        </div>
      </section>

      {/* ── Worker hours ──────────────────────────────────────────────────── */}
      <section className="bg-slate-900 border border-slate-800 rounded-xl p-5 space-y-3">
        <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-wide">
          Radni sati
          <span className="ml-2 text-slate-600 font-normal normal-case text-xs">
            ({workerEntries.filter(e => e.hours.trim() !== '').length} od {workers.length} radnika)
          </span>
        </h2>
        <WorkerHoursEditor
          workers={workers}
          entries={workerEntries}
          onChange={setWorkerEntries}
          disabled={submitting}
        />
      </section>

      {/* ── Activities ────────────────────────────────────────────────────── */}
      <section className="bg-slate-900 border border-slate-800 rounded-xl p-5 space-y-4">
        <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-wide">
          Aktivnosti
          <span className="ml-2 text-slate-600 font-normal normal-case text-xs">
            ({activities.length} stavki)
          </span>
        </h2>

        <ActivityListPreview
          activities={activities}
          onRemove={removeActivity}
          disabled={submitting}
        />

        <ActivityEntryForm
          key={projectId}
          materials={projectMaterials}
          materialsLoading={materialsLoading}
          onAdd={addActivity}
          disabled={submitting || !projectId}
        />
      </section>

      {/* ── Submit ────────────────────────────────────────────────────────── */}
      {serverError && (
        <div className="bg-red-900/30 border border-red-800 rounded-lg px-4 py-3 text-sm text-red-300">
          {serverError}
        </div>
      )}

      <div className="flex flex-col-reverse sm:flex-row sm:items-center sm:justify-end gap-3">
        <button
          type="button"
          onClick={() => router.back()}
          disabled={submitting}
          className="w-full sm:w-auto px-4 py-3 sm:py-2 text-sm text-slate-400 hover:text-slate-200 disabled:opacity-50 transition-colors text-center border border-slate-800 rounded-lg sm:border-0 sm:rounded-none"
        >
          Odustani
        </button>
        <button
          type="submit"
          disabled={submitting}
          className="w-full sm:w-auto px-5 py-3 sm:py-2 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white text-sm font-medium rounded-lg transition-colors"
        >
          {submitting ? 'Spremanje…' : submitLabel}
        </button>
      </div>
    </form>
  )
}
