'use client'

import { useEffect, useState } from 'react'
import { useParams, useRouter, useSearchParams } from 'next/navigation'
import Link from 'next/link'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
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

type ActionPanel = 'none' | 'confirming-approve' | 'rejecting'

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

  const [actionPanel, setActionPanel] = useState<ActionPanel>('none')
  const [rejectReason, setRejectReason] = useState('')

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
    if (!report) return
    setActing(true)
    setActionError(null)
    try {
      await apiClient.patch(`/daily-reports/${report.id}/approve`)
      const res = await apiClient.get(`/daily-reports/${report.id}`)
      setReport(res.data.daily_report)
      setActionPanel('none')
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } }
      setActionError(err?.response?.data?.error ?? 'Greška pri odobravanju.')
    } finally { setActing(false) }
  }

  async function handleReject() {
    if (!report || !rejectReason.trim()) {
      setActionError('Razlog odbijanja je obavezan.')
      return
    }
    setActing(true)
    setActionError(null)
    try {
      await apiClient.patch(`/daily-reports/${report.id}/reject`, { reason: rejectReason.trim() })
      const res = await apiClient.get(`/daily-reports/${report.id}`)
      setReport(res.data.daily_report)
      setActionPanel('none')
      setRejectReason('')
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } }
      setActionError(err?.response?.data?.error ?? 'Greška pri odbijanju.')
    } finally { setActing(false) }
  }

  if (isLoading) return <LoadingScreen />
  if (!user) { router.replace('/login'); return null }

  return (
    <AppShell
      user={user}
      employee={employee}
      title="Dnevni izvještaj"
      backHref="/dashboard/daily-reports"
      onLogout={logout}
      action={
        report && canEdit ? (
          <Link
            href={`/dashboard/daily-reports/${report.id}/edit`}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-gi-card-elevated border border-gi-border hover:border-gi-border/80 text-gi-text-muted hover:text-gi-text text-xs font-medium transition-colors"
          >
            <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931z" />
            </svg>
            Uredi
          </Link>
        ) : undefined
      }
    >
      <div className="space-y-4">
        {loadError && (
          <div className="bg-red-900/30 border border-red-800/60 rounded-lg px-4 py-3">
            <p className="text-red-300 text-sm">{loadError}</p>
          </div>
        )}

        {!report && !loadError && (
          <div className="bg-gi-card border border-gi-border rounded-xl p-10 text-center">
            <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje…</p>
          </div>
        )}

        {report && (
          <>
            {/* ── Partial photo-upload failure banner (full width) ─────────── */}
            {uploadErrors > 0 && (
              <div className="flex items-start gap-3 bg-amber-950/40 border border-amber-700/50 rounded-xl px-4 py-3">
                <svg className="w-4 h-4 text-amber-400 mt-0.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
                </svg>
                <p className="text-sm text-amber-300">
                  Izvještaj je uspješno kreiran, ali{' '}
                  <span className="font-semibold">{uploadErrors} {uploadErrors === 1 ? 'fotografija nije prenesena' : 'fotografija nije preneseno'}</span>.
                  Možete ih dodati u odjeljku <span className="font-medium text-amber-200">Fotografije</span> ispod.
                </p>
              </div>
            )}

            {/* ── Header card (full width) ────────────────────────────────── */}
            <div className="bg-gi-card border border-gi-border rounded-xl p-5 space-y-3">
              <div className="flex flex-wrap items-start gap-3 justify-between">
                <div>
                  <h1 className="text-xl font-bold text-gi-text">{report.project.name}</h1>
                  <p className="text-gi-text-muted text-sm mt-0.5 capitalize">{formatDate(report.report_date)}</p>
                </div>
                <DailyReportStatusBadge status={report.status} />
              </div>

              <dl className="grid grid-cols-2 sm:grid-cols-3 gap-x-6 gap-y-3 text-sm border-t border-gi-border pt-3">
                <div>
                  <dt className="text-[10px] text-gi-text-subtle uppercase tracking-wider mb-0.5">Poslovođa</dt>
                  <dd className="text-gi-text-muted">{report.poslovoda.full_name}</dd>
                </div>
                {report.submitted_at && (
                  <div>
                    <dt className="text-[10px] text-gi-text-subtle uppercase tracking-wider mb-0.5">Predano</dt>
                    <dd className="text-gi-text-muted">{formatDateTime(report.submitted_at)}</dd>
                  </div>
                )}
                <div>
                  <dt className="text-[10px] text-gi-text-subtle uppercase tracking-wider mb-0.5">Kreirano</dt>
                  <dd className="text-gi-text-muted">{formatDateTime(report.created_at)}</dd>
                </div>
              </dl>

              {report.notes && (
                <div className="border-t border-gi-border pt-3">
                  <p className="text-[10px] text-gi-text-subtle uppercase tracking-wider mb-1">Napomena</p>
                  <p className="text-sm text-gi-text-muted">{report.notes}</p>
                </div>
              )}
            </div>

            {/* ── Two-column body ─────────────────────────────────────────── */}
            <div className="grid grid-cols-1 lg:grid-cols-[1fr_320px] gap-4 lg:gap-6 lg:items-start">

              {/* ── Main column: activities + photos ──────────────────────── */}
              <div className="space-y-4">
                {/* Activities */}
                <div className="bg-gi-card border border-gi-border rounded-xl p-5 space-y-3">
                  <div className="flex items-center justify-between gap-2">
                    <h2 className="text-xs font-medium text-gi-text-subtle uppercase tracking-wider">Aktivnosti</h2>
                    {report.activities.length > 0 && (
                      <span className="text-xs text-gi-text-subtle bg-gi-card-elevated border border-gi-border px-2 py-0.5 rounded-full">
                        {report.activities.length}
                      </span>
                    )}
                  </div>
                  {report.activities.length === 0 ? (
                    <p className="text-sm text-gi-text-subtle">Nema aktivnosti.</p>
                  ) : (
                    <div className="space-y-1.5">
                      {report.activities.map((a, idx) => (
                        <div key={a.id} className="flex items-start gap-3 bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2.5">
                          <span className="text-xs text-gi-text-subtle w-5 pt-0.5 shrink-0 select-none">{idx + 1}.</span>
                          <div className="flex-1 min-w-0 space-y-1">
                            <div className="flex flex-wrap items-center gap-2">
                              <span className="text-sm text-gi-text font-medium">
                                {a.custom_material_name ?? a.material_name}
                              </span>
                              <ActivityTypeBadge type={a.activity_type} isVtk={a.is_vtk} trackingType={a.tracking_type} />
                            </div>
                            <div className="flex flex-wrap gap-x-4 text-xs text-gi-text-subtle">
                              <span>Količina: <span className="text-gi-text-muted">{a.quantity} {a.unit}</span></span>
                              {a.notes && <span>Napomena: <span className="text-gi-text-muted">{a.notes}</span></span>}
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>

                {/* Photo attachments */}
                <ReportPhotoGallery
                  reportId={report.id}
                  initialAttachments={report.attachments ?? []}
                  canEdit={canEdit}
                />
              </div>

              {/* ── Right sidebar: worker hours + approval actions ─────────── */}
              <div className="space-y-4 lg:sticky lg:top-4">

                {/* Worker hours */}
                {report.worker_hours.length > 0 ? (
                  <div className="bg-gi-card border border-gi-border rounded-xl p-5 space-y-3">
                    <div className="flex items-center justify-between gap-2">
                      <h2 className="text-xs font-medium text-gi-text-subtle uppercase tracking-wider">Radni sati</h2>
                      <span className="text-xs text-gi-text-subtle">
                        {report.worker_hours.length} radnika &middot; {report.worker_hours.reduce((s, w) => s + w.hours_worked, 0).toFixed(1)}h ukupno
                      </span>
                    </div>
                    <div className="space-y-1">
                      {report.worker_hours.map(wh => (
                        <div key={wh.id} className="flex items-center justify-between px-2 py-1.5 rounded-lg hover:bg-gi-card-elevated/60 transition-colors">
                          <span className="text-sm text-gi-text">{wh.worker_name}</span>
                          <div className="flex items-center gap-4 text-sm">
                            {wh.notes && <span className="text-gi-text-subtle text-xs truncate max-w-32">{wh.notes}</span>}
                            <span className="text-gi-text-muted font-medium tabular-nums">{wh.hours_worked}h</span>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                ) : (
                  <div className="flex items-start gap-3 bg-blue-950/30 border border-blue-800/40 rounded-xl px-4 py-3">
                    <svg className="w-4 h-4 text-blue-400 mt-0.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    <p className="text-sm text-blue-300">
                      Radnici sada samostalno unose svoje radne sate kroz modul <span className="font-medium text-blue-200">Moji sati</span>.
                    </p>
                  </div>
                )}

                {/* Approve / Reject actions */}
                {canManage && (report.status === 'submitted' || report.status === 'rejected') && (
                  <div className="bg-gi-card border border-gi-border rounded-xl p-5 space-y-4">
                    <h2 className="text-xs font-medium text-gi-text-subtle uppercase tracking-wider">Upravljanje izvještajem</h2>

                    {actionError && (
                      <div className="bg-red-900/30 border border-red-800/60 rounded-lg px-3 py-2.5">
                        <p className="text-sm text-red-300">{actionError}</p>
                      </div>
                    )}

                    {actionPanel === 'confirming-approve' ? (
                      <div className="bg-green-950/20 border border-green-800/40 rounded-lg px-4 py-3 space-y-3">
                        <p className="text-sm text-green-200 font-medium">Odobriti ovaj izvještaj?</p>
                        <p className="text-xs text-green-300/70">
                          Odobrenjem će se automatski ažurirati dostupno stanje materijala na projektu za sve aktivnosti montaže i demontaže.
                        </p>
                        <div className="flex gap-2">
                          <button
                            type="button"
                            disabled={acting}
                            onClick={handleApprove}
                            className="text-sm bg-green-700 hover:bg-green-600 disabled:opacity-40 text-white font-medium px-3 py-1.5 rounded-lg transition-colors"
                          >
                            {acting ? 'Odobravanje…' : 'Odobri'}
                          </button>
                          <button
                            type="button"
                            disabled={acting}
                            onClick={() => { setActionPanel('none'); setActionError(null) }}
                            className="text-sm border border-gi-border text-gi-text-muted hover:bg-gi-card-elevated px-3 py-1.5 rounded-lg transition-colors disabled:opacity-40"
                          >
                            Odustani
                          </button>
                        </div>
                      </div>
                    ) : actionPanel === 'rejecting' ? (
                      <div className="bg-red-950/20 border border-red-800/40 rounded-lg px-4 py-3 space-y-3">
                        <p className="text-sm text-red-200 font-medium">Odbiti izvještaj</p>
                        <div>
                          <label className="block text-xs font-medium text-gi-text-muted uppercase tracking-wider mb-1.5">
                            Razlog odbijanja *
                          </label>
                          <input
                            type="text"
                            value={rejectReason}
                            onChange={e => setRejectReason(e.target.value)}
                            autoFocus
                            disabled={acting}
                            placeholder="Unesite razlog…"
                            className="w-full bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2 text-sm text-gi-text placeholder:text-gi-text-subtle focus:outline-none focus:border-red-700 transition-colors disabled:opacity-50"
                          />
                        </div>
                        <div className="flex gap-2">
                          <button
                            type="button"
                            disabled={acting || !rejectReason.trim()}
                            onClick={handleReject}
                            className="text-sm bg-red-700 hover:bg-red-600 disabled:opacity-40 text-white font-medium px-3 py-1.5 rounded-lg transition-colors"
                          >
                            {acting ? 'Odbijanje…' : 'Odbij'}
                          </button>
                          <button
                            type="button"
                            disabled={acting}
                            onClick={() => { setActionPanel('none'); setRejectReason(''); setActionError(null) }}
                            className="text-sm border border-gi-border text-gi-text-muted hover:bg-gi-card-elevated px-3 py-1.5 rounded-lg transition-colors disabled:opacity-40"
                          >
                            Odustani
                          </button>
                        </div>
                      </div>
                    ) : (
                      <div className="flex flex-wrap gap-2">
                        <button
                          type="button"
                          disabled={acting}
                          onClick={() => { setActionPanel('confirming-approve'); setActionError(null) }}
                          className="px-4 py-2 bg-green-900/40 border border-green-700/60 hover:bg-green-800/50 disabled:opacity-50 text-green-300 text-sm font-medium rounded-lg transition-colors"
                        >
                          Odobri izvještaj
                        </button>
                        {report.status === 'submitted' && (
                          <button
                            type="button"
                            disabled={acting}
                            onClick={() => { setActionPanel('rejecting'); setActionError(null) }}
                            className="px-4 py-2 bg-red-900/40 border border-red-700/60 hover:bg-red-800/50 disabled:opacity-50 text-red-300 text-sm font-medium rounded-lg transition-colors"
                          >
                            Odbij
                          </button>
                        )}
                      </div>
                    )}
                  </div>
                )}
              </div>

            </div>
          </>
        )}
      </div>
    </AppShell>
  )
}
