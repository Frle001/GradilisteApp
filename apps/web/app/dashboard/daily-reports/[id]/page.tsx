'use client'

import { useEffect, useState } from 'react'
import { useParams, useRouter, useSearchParams } from 'next/navigation'
import Link from 'next/link'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import DashboardShell from '@/components/layout/DashboardShell'
import DailyReportStatusBadge from '@/components/daily-reports/DailyReportStatusBadge'
import ActivityTypeBadge from '@/components/daily-reports/ActivityTypeBadge'
import {
  type DailyReportDetail,
  canEditDailyReport,
  canApproveReject,
} from '@/lib/types/daily-reports'
import apiClient from '@/lib/api-client'
import ReportPhotoGallery from '@/components/daily-reports/ReportPhotoGallery'

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString('hr-HR', {
    weekday: 'long', day: '2-digit', month: 'long', year: 'numeric',
  })
}

function formatDateTime(dateStr: string) {
  return new Date(dateStr).toLocaleString('hr-HR', {
    day: '2-digit', month: '2-digit', year: 'numeric',
    hour: '2-digit', minute: '2-digit',
  })
}

export default function DailyReportDetailPage() {
  const { user, employee, isLoading, logout } = useAuth()
  const params = useParams<{ id: string }>()
  const router = useRouter()
  const searchParams = useSearchParams()
  const uploadErrors = Number(searchParams.get('upload_errors') ?? 0)

  const [report, setReport] = useState<DailyReportDetail | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [acting, setActing] = useState(false)

  useEffect(() => {
    if (!user || !params.id) return
    apiClient
      .get(`/daily-reports/${params.id}`)
      .then(res => setReport(res.data.daily_report))
      .catch(() => setLoadError('Izvještaj nije pronađen ili nemate pristup.'))
  }, [user, params.id])

  const callerEmpId = employee?.id ?? null
  const canEdit = !!report && !!user && canEditDailyReport(user.role, callerEmpId, report.poslovoda.id, report.status)
  const canManage = !!user && canApproveReject(user.role)

  async function handleApprove() {
    if (!report || !confirm('Odobriti ovaj izvještaj?\n\nOdobrenjem će se automatski ažurirati dostupno stanje materijala na projektu za sve aktivnosti montaže i demontaže.')) return
    setActing(true)
    setActionError(null)
    try {
      await apiClient.patch(`/daily-reports/${report.id}/approve`)
      const res = await apiClient.get(`/daily-reports/${report.id}`)
      setReport(res.data.daily_report)
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } }
      setActionError(err?.response?.data?.error ?? 'Greška pri odobravanju.')
    } finally { setActing(false) }
  }

  async function handleReject() {
    if (!report) return
    const reason = prompt('Razlog odbijanja (obavezno):')
    if (!reason?.trim()) return
    setActing(true)
    setActionError(null)
    try {
      await apiClient.patch(`/daily-reports/${report.id}/reject`, { reason })
      const res = await apiClient.get(`/daily-reports/${report.id}`)
      setReport(res.data.daily_report)
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } }
      setActionError(err?.response?.data?.error ?? 'Greška pri odbijanju.')
    } finally { setActing(false) }
  }

  if (isLoading) return <LoadingScreen />
  if (!user) { router.replace('/login'); return null }

  return (
    <DashboardShell
      user={user}
      employee={employee}
      title="Dnevni izvještaj"
      backHref="/dashboard/daily-reports"
      onLogout={logout}
      action={
        report && canEdit ? (
          <Link
            href={`/dashboard/daily-reports/${report.id}/edit`}
            className="px-4 py-2 bg-white text-slate-900 font-semibold rounded-lg text-sm hover:bg-slate-100 transition"
          >
            Uredi
          </Link>
        ) : undefined
      }
    >
      <div className="max-w-3xl space-y-6">
        {loadError && (
          <div className="bg-red-950 border border-red-800 rounded-lg px-4 py-3">
            <p className="text-red-300 text-sm">{loadError}</p>
          </div>
        )}

        {!report && !loadError && (
          <p className="text-slate-400 text-sm">Učitavanje…</p>
        )}

        {report && (
          <>
            {/* ── Partial photo-upload failure banner ─────────────────── */}
            {uploadErrors > 0 && (
              <div className="flex items-start gap-3 bg-amber-950/50 border border-amber-700/60 rounded-xl px-4 py-3">
                <svg className="w-4 h-4 text-amber-400 mt-0.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
                </svg>
                <p className="text-sm text-amber-300">
                  Izvještaj je uspješno kreiran, ali{' '}
                  <span className="font-semibold">{uploadErrors} {uploadErrors === 1 ? 'fotografija nije prenesena' : 'fotografija nije preneseno'}</span>.
                  Možete ih dodati ovdje u odjeljku <span className="font-medium text-amber-200">Fotografije</span>.
                </p>
              </div>
            )}

            {/* ── Header ─────────────────────────────────────────────── */}
            <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 space-y-3">
              <div className="flex flex-wrap items-start gap-3 justify-between">
                <div>
                  <h1 className="text-xl font-bold text-white">{report.project.name}</h1>
                  <p className="text-slate-400 text-sm mt-0.5">{formatDate(report.report_date)}</p>
                </div>
                <DailyReportStatusBadge status={report.status} />
              </div>

              <dl className="grid grid-cols-2 sm:grid-cols-3 gap-3 text-sm">
                <div>
                  <dt className="text-xs text-slate-500 mb-0.5">Poslovođa</dt>
                  <dd className="text-slate-200">{report.poslovoda.full_name}</dd>
                </div>
                {report.submitted_at && (
                  <div>
                    <dt className="text-xs text-slate-500 mb-0.5">Predano</dt>
                    <dd className="text-slate-200">{formatDateTime(report.submitted_at)}</dd>
                  </div>
                )}
                <div>
                  <dt className="text-xs text-slate-500 mb-0.5">Kreirano</dt>
                  <dd className="text-slate-200">{formatDateTime(report.created_at)}</dd>
                </div>
              </dl>

              {report.notes && (
                <div className="border-t border-slate-800 pt-3">
                  <p className="text-xs text-slate-500 mb-1">Napomena</p>
                  <p className="text-sm text-slate-300">{report.notes}</p>
                </div>
              )}
            </div>

            {/* ── Worker hours ────────────────────────────────────────── */}
            {report.worker_hours.length > 0 ? (
              <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 space-y-3">
                <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-wide">
                  Radni sati
                  <span className="ml-2 text-slate-600 font-normal normal-case text-xs">
                    {report.worker_hours.length} radnika &middot; {report.worker_hours.reduce((s, w) => s + w.hours_worked, 0).toFixed(1)}h ukupno
                  </span>
                </h2>
                <div className="space-y-1">
                  {report.worker_hours.map(wh => (
                    <div key={wh.id} className="flex items-center justify-between px-2 py-1.5 rounded hover:bg-slate-800/40">
                      <span className="text-sm text-slate-200">{wh.worker_name}</span>
                      <div className="flex items-center gap-4 text-sm">
                        {wh.notes && <span className="text-slate-500 text-xs truncate max-w-32">{wh.notes}</span>}
                        <span className="text-slate-300 font-medium tabular-nums">{wh.hours_worked}h</span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ) : (
              <div className="flex items-start gap-3 bg-blue-950/40 border border-blue-800/50 rounded-xl px-4 py-3">
                <svg className="w-4 h-4 text-blue-400 mt-0.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <p className="text-sm text-blue-300">
                  Radnici sada samostalno unose svoje radne sate kroz modul <span className="font-medium text-blue-200">Moji sati</span>.
                </p>
              </div>
            )}

            {/* ── Activities ──────────────────────────────────────────── */}
            <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 space-y-3">
              <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-wide">
                Aktivnosti
                <span className="ml-2 text-slate-600 font-normal normal-case text-xs">
                  {report.activities.length} stavki
                </span>
              </h2>
              {report.activities.length === 0 ? (
                <p className="text-sm text-slate-500">Nema aktivnosti.</p>
              ) : (
                <div className="space-y-1.5">
                  {report.activities.map((a, idx) => (
                    <div key={a.id} className="flex items-start gap-3 bg-slate-800/60 border border-slate-700 rounded-lg px-3 py-2.5">
                      <span className="text-xs text-slate-600 w-5 pt-0.5 shrink-0">{idx + 1}.</span>
                      <div className="flex-1 min-w-0 space-y-1">
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="text-sm text-slate-100 font-medium">
                            {a.custom_material_name ?? a.material_name}
                          </span>
                          <ActivityTypeBadge type={a.activity_type} isVtk={a.is_vtk} trackingType={a.tracking_type} />
                        </div>
                        <div className="flex flex-wrap gap-x-4 text-xs text-slate-400">
                          <span>Količina: <span className="text-slate-200">{a.quantity} {a.unit}</span></span>
                          {a.notes && <span>Napomena: <span className="text-slate-300">{a.notes}</span></span>}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* ── Photo attachments ───────────────────────────────────── */}
            <ReportPhotoGallery
              reportId={report.id}
              initialAttachments={report.attachments ?? []}
              canEdit={canEdit}
            />

            {/* ── Approve / Reject actions ────────────────────────────── */}
            {canManage && report.status === 'submitted' && (
              <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 space-y-3">
                <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-wide">Akcije</h2>
                {actionError && (
                  <p className="text-sm text-red-400">{actionError}</p>
                )}
                <div className="flex gap-3">
                  <button
                    type="button"
                    disabled={acting}
                    onClick={handleApprove}
                    className="px-4 py-2 bg-green-700 hover:bg-green-600 disabled:opacity-50 text-white text-sm font-medium rounded-lg transition"
                  >
                    Odobri
                  </button>
                  <button
                    type="button"
                    disabled={acting}
                    onClick={handleReject}
                    className="px-4 py-2 bg-red-700 hover:bg-red-600 disabled:opacity-50 text-white text-sm font-medium rounded-lg transition"
                  >
                    Odbij
                  </button>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </DashboardShell>
  )
}
