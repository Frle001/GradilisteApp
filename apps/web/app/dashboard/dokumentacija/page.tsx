'use client'

import { useCallback, useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
import DocumentPreviewDialog from '@/components/ui/DocumentPreviewDialog'
import apiClient from '@/lib/api-client'
import type {
  DocEmployee,
  DocNotification,
  EmployeeDocSummary,
  MedicalExam,
  SafetyRecord,
  EmploymentContract,
  WorkPermit,
  PensionRecord,
  HealthRecord,
  AnnualLeave,
  MonthlyObligation,
} from '@/lib/types/documentation'

// ── Constants & helpers ───────────────────────────────────────────────────────

const HR_MONTHS = ['', 'Siječanj', 'Veljača', 'Ožujak', 'Travanj', 'Svibanj', 'Lipanj',
  'Srpanj', 'Kolovoz', 'Rujan', 'Listopad', 'Studeni', 'Prosinac']

function formatDate(s: string | null | undefined) {
  if (!s) return '—'
  const d = new Date(s)
  if (isNaN(d.getTime())) return s
  return d.toLocaleDateString('hr-HR')
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function recentMonthOptions(n = 24) {
  const now = new Date()
  const out: { value: string; label: string }[] = []
  for (let i = 0; i < n; i++) {
    const d = new Date(now.getFullYear(), now.getMonth() - i, 1)
    const y = d.getFullYear(); const m = d.getMonth() + 1
    out.push({ value: `${y}-${String(m).padStart(2, '0')}`, label: `${HR_MONTHS[m]} ${y}` })
  }
  return out
}

function parseYM(val: string) {
  const [y, m] = val.split('-').map(Number)
  return { year: y, month: m }
}

async function uploadDocFile(employeeId: string, file: File): Promise<string> {
  const fd = new FormData()
  fd.append('file', file)
  const { data } = await apiClient.post(`/documentation/employees/${employeeId}/files`, fd)
  return data.file_id as string
}

function errMsg(e: unknown): string {
  return (e as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Neočekivana greška'
}

function docPreviewUrl(fileId: string) { return `/documentation/files/${fileId}/download` }

// ── Shared styles ─────────────────────────────────────────────────────────────

const inputCls = [
  'w-full bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2',
  'text-sm text-gi-text placeholder:text-gi-text-subtle',
  'focus:outline-none focus:border-gi-yellow transition-colors',
].join(' ')

// ── DocFilePicker ─────────────────────────────────────────────────────────────

function DocFilePicker({ file, onChange }: { file: File | null; onChange: (f: File | null) => void }) {
  return (
    <div className={[
      'relative rounded-lg border transition-colors cursor-pointer',
      file ? 'border-gi-yellow/60 bg-gi-yellow/5' : 'border-dashed border-gi-border hover:border-gi-text-subtle bg-gi-card-elevated',
    ].join(' ')}>
      <input
        type="file"
        accept=".pdf,.doc,.docx,.xls,.xlsx,.jpg,.jpeg,.png"
        className="absolute inset-0 opacity-0 cursor-pointer w-full h-full"
        onChange={e => onChange(e.target.files?.[0] ?? null)}
      />
      <div className="px-3 py-2.5 flex items-center gap-2.5 pointer-events-none">
        {file ? (
          <>
            <svg className="w-4 h-4 text-gi-yellow shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" />
            </svg>
            <div className="min-w-0 flex-1">
              <p className="text-sm text-gi-text truncate">{file.name}</p>
              <p className="text-xs text-gi-text-subtle">{formatFileSize(file.size)}</p>
            </div>
            <svg className="w-3.5 h-3.5 text-gi-yellow shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M4.5 12.75l6 6 9-13.5" />
            </svg>
          </>
        ) : (
          <>
            <svg className="w-4 h-4 text-gi-text-subtle shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 16.5V9.75m0 0l3 3m-3-3l-3 3M6.75 19.5a4.5 4.5 0 01-1.41-8.775 5.25 5.25 0 0110.233-2.33 3 3 0 013.758 3.848A3.752 3.752 0 0118 19.5H6.75z" />
            </svg>
            <div>
              <p className="text-sm text-gi-text-muted">Odaberite dokument</p>
              <p className="text-xs text-gi-text-subtle">PDF, Word, Excel, JPG, PNG</p>
            </div>
          </>
        )}
      </div>
    </div>
  )
}

// ── Badges ────────────────────────────────────────────────────────────────────

function PriorityBadge({ priority }: { priority: string }) {
  if (priority === 'low')     return <span className="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-blue-950/40 text-blue-300 border border-blue-800/40">Nisko</span>
  if (priority === 'warning') return <span className="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-amber-900/40 text-amber-300 border border-amber-700/40">Upozorenje</span>
  if (priority === 'urgent')  return <span className="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-orange-900/40 text-orange-300 border border-orange-700/40">Hitno</span>
  if (priority === 'overdue') return <span className="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-red-950/50 text-red-400 border border-red-800/50">Prekoračeno</span>
  return null
}

function RecordStatusBadge({ status }: { status: string }) {
  if (status === 'pending')   return <span className="text-xs text-gi-text-muted">Na čekanju</span>
  if (status === 'uploaded')  return <span className="text-xs text-amber-300">Učitano</span>
  if (status === 'verified')  return <span className="text-xs text-emerald-400">Verificirano</span>
  if (status === 'completed') return <span className="text-xs text-emerald-400">Završeno</span>
  return <span className="text-xs text-gi-text-subtle">{status}</span>
}

function ShowMoreList<T>({ items, limit = 6, render }: { items: T[]; limit?: number; render: (item: T) => React.ReactNode }) {
  const [showAll, setShowAll] = useState(false)
  const visible = showAll ? items : items.slice(0, limit)
  return (
    <>
      {visible.map(render)}
      {items.length > limit && !showAll && (
        <button type="button" onClick={() => setShowAll(true)} className="mt-1 text-xs text-gi-yellow hover:text-gi-yellow-hover transition-colors">
          Prikaži još {items.length - limit}
        </button>
      )}
    </>
  )
}

// ── Category system ───────────────────────────────────────────────────────────

type CategoryId = 'medical' | 'safety' | 'contract' | 'payroll' | 'timesheet' | 'leave' | 'permit' | 'pension' | 'health'
type CatStatus = 'overdue' | 'urgent' | 'warning' | 'low' | 'ok'

const CATEGORIES: { id: CategoryId; label: string; radnikOnly?: boolean }[] = [
  { id: 'medical',   label: 'Liječnički pregled' },
  { id: 'safety',    label: 'Zaštita na radu',    radnikOnly: true },
  { id: 'contract',  label: 'Ugovor o radu' },
  { id: 'payroll',   label: 'Platne liste' },
  { id: 'timesheet', label: 'Evidencija radnika',  radnikOnly: true },
  { id: 'leave',     label: 'Godišnji odmor' },
  { id: 'permit',    label: 'Radne dozvole' },
  { id: 'pension',   label: 'Mirovinsko',          radnikOnly: true },
  { id: 'health',    label: 'Zdravstveno',          radnikOnly: true },
]

const NOTIF_TO_CATEGORY: Record<string, CategoryId> = {
  medical_warning: 'medical', medical_urgent: 'medical', medical_overdue: 'medical',
  safety_low: 'safety', safety_urgent: 'safety',
  contract_warning: 'contract', contract_overdue: 'contract',
  payroll_warning: 'payroll', payroll_overdue: 'payroll',
  timesheet_warning: 'timesheet', timesheet_overdue: 'timesheet',
  permit_low: 'permit', permit_warning: 'permit', permit_urgent: 'permit', permit_overdue: 'permit',
  pension_missing: 'pension',
  health_missing: 'health',
}

const PRIORITY_ORDER: Record<string, number> = { overdue: 4, urgent: 3, warning: 2, low: 1 }

function getCategoryStatus(catId: CategoryId, empId: string, notifications: DocNotification[]): CatStatus {
  const empNotifs = notifications.filter(n => n.employee_id === empId && NOTIF_TO_CATEGORY[n.type] === catId)
  if (!empNotifs.length) return 'ok'
  const best = empNotifs.reduce((a, b) => (PRIORITY_ORDER[b.priority] ?? 0) > (PRIORITY_ORDER[a.priority] ?? 0) ? b : a)
  return best.priority as CatStatus
}

function catStatusDot(s: CatStatus) {
  if (s === 'overdue') return <span className="w-1.5 h-1.5 rounded-full bg-red-500 shrink-0" />
  if (s === 'urgent')  return <span className="w-1.5 h-1.5 rounded-full bg-orange-400 shrink-0" />
  if (s === 'warning') return <span className="w-1.5 h-1.5 rounded-full bg-amber-400 shrink-0" />
  if (s === 'low')     return <span className="w-1.5 h-1.5 rounded-full bg-blue-400 shrink-0" />
  return null
}

// ── Section: Medical ──────────────────────────────────────────────────────────

function MedicalSection({ employeeId, initial, onRefresh }: { employeeId: string; initial?: MedicalExam; onRefresh: () => void }) {
  const [exams, setExams]             = useState<MedicalExam[]>([])
  const [historyOpen, setHistoryOpen] = useState(false)
  const [histLoading, setHistLoading] = useState(false)
  const [formOpen, setFormOpen]       = useState(false)
  const [completedDate, setCompleted] = useState('')
  const [expiresAt, setExpires]       = useState('')
  const [file, setFile]               = useState<File | null>(null)
  const [submitting, setSubmitting]   = useState(false)
  const [error, setError]             = useState<string | null>(null)

  const loadHistory = useCallback(async () => {
    setHistLoading(true)
    try { const r = await apiClient.get(`/documentation/employees/${employeeId}/medical-exams`); setExams(r.data.exams ?? []) }
    finally { setHistLoading(false) }
  }, [employeeId])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault(); setSubmitting(true); setError(null)
    try {
      let fileId: string | undefined
      if (file) fileId = await uploadDocFile(employeeId, file)
      await apiClient.post(`/documentation/employees/${employeeId}/medical-exams`, {
        completed_date: completedDate, expires_at: expiresAt, ...(fileId ? { document_id: fileId } : {}),
      })
      setFormOpen(false); setCompleted(''); setExpires(''); setFile(null); onRefresh(); loadHistory()
    } catch (e) { setError(errMsg(e)) } finally { setSubmitting(false) }
  }

  return (
    <div className="space-y-3">
      {initial ? (
        <div className="rounded-lg border border-gi-border bg-gi-card-elevated px-4 py-3 space-y-1">
          <p className="text-xs text-gi-text-subtle uppercase tracking-wider">Aktivan pregled</p>
          <div className="flex flex-wrap items-center gap-3">
            <div>
              <p className="text-xs text-gi-text-muted">Obavljen</p>
              <p className="text-sm text-gi-text">{formatDate(initial.completed_date)}</p>
            </div>
            <div>
              <p className="text-xs text-gi-text-muted">Istječe</p>
              <p className="text-sm text-gi-text">{formatDate(initial.expires_at)}</p>
            </div>
            {initial.document && (
              <div className="flex gap-2 ml-auto">
                <DocumentPreviewDialog fetchUrl={docPreviewUrl(initial.document.id)} filename={initial.document.original_filename} mimeType={initial.document.mime_type} />
              </div>
            )}
          </div>
        </div>
      ) : (
        <p className="text-sm text-gi-text-subtle italic">Nema evidencije</p>
      )}
      <div className="flex flex-wrap gap-2">
        <button onClick={() => setFormOpen(v => !v)} className="text-xs font-medium text-gi-navy bg-gi-yellow hover:bg-gi-yellow-hover px-3 py-1.5 rounded-lg transition-colors">
          {formOpen ? 'Zatvori' : 'Novi pregled'}
        </button>
        <button onClick={() => { if (!historyOpen) loadHistory(); setHistoryOpen(v => !v) }} className="text-xs text-gi-text-muted border border-gi-border hover:border-gi-text-subtle px-3 py-1.5 rounded-lg transition-colors">
          {historyOpen ? 'Sakrij povijest' : 'Prikaži povijest'}
        </button>
      </div>
      {formOpen && (
        <form onSubmit={handleSubmit} className="space-y-3 border-t border-gi-border pt-3">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Datum pregleda *</label>
              <input type="date" required value={completedDate} onChange={e => setCompleted(e.target.value)} className={inputCls} />
            </div>
            <div>
              <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Istječe *</label>
              <input type="date" required value={expiresAt} onChange={e => setExpires(e.target.value)} className={inputCls} />
            </div>
          </div>
          <div>
            <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Dokument (neobavezno)</label>
            <DocFilePicker file={file} onChange={setFile} />
          </div>
          {error && <p className="text-xs text-red-400">{error}</p>}
          <div className="flex gap-2">
            <button type="submit" disabled={submitting} className="text-xs font-semibold text-gi-navy bg-gi-yellow hover:bg-gi-yellow-hover px-3 py-1.5 rounded-lg transition-colors disabled:opacity-50">{submitting ? 'Sprema…' : 'Spremi'}</button>
            <button type="button" onClick={() => setFormOpen(false)} className="text-xs text-gi-text-muted border border-gi-border hover:border-gi-text-subtle px-3 py-1.5 rounded-lg transition-colors">Odustani</button>
          </div>
        </form>
      )}
      {historyOpen && (
        <div className="border-t border-gi-border pt-3 space-y-1.5">
          {histLoading && <p className="text-xs text-gi-text-subtle animate-pulse">Učitavanje…</p>}
          <ShowMoreList items={exams} render={(ex) => (
            <div key={ex.id} className="flex flex-wrap justify-between text-xs text-gi-text-muted gap-2">
              <span>Obavljen {formatDate(ex.completed_date)}</span>
              <span className="flex items-center gap-2 text-gi-text-subtle">
                istj. {formatDate(ex.expires_at)}
                {ex.document && <DocumentPreviewDialog fetchUrl={docPreviewUrl(ex.document.id)} filename={ex.document.original_filename} mimeType={ex.document.mime_type} />}
              </span>
            </div>
          )} />
          {!histLoading && exams.length === 0 && <p className="text-xs text-gi-text-subtle">Nema evidencije</p>}
        </div>
      )}
    </div>
  )
}

// ── Section: Safety ───────────────────────────────────────────────────────────

function SafetySection({ employeeId, record, onRefresh }: { employeeId: string; record?: SafetyRecord; onRefresh: () => void }) {
  const [file, setFile]           = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [verifying, setVerifying] = useState(false)
  const [error, setError]         = useState<string | null>(null)

  async function handleUpload() {
    if (!file) return; setUploading(true); setError(null)
    try { const fileId = await uploadDocFile(employeeId, file); await apiClient.put(`/documentation/employees/${employeeId}/safety`, { document_id: fileId }); setFile(null); onRefresh() }
    catch (e) { setError(errMsg(e)) } finally { setUploading(false) }
  }
  async function handleVerify() {
    setVerifying(true); setError(null)
    try { await apiClient.post(`/documentation/employees/${employeeId}/safety/verify`); onRefresh() }
    catch (e) { setError(errMsg(e)) } finally { setVerifying(false) }
  }

  return (
    <div className="space-y-3">
      {record ? (
        <div className="rounded-lg border border-gi-border bg-gi-card-elevated px-4 py-3 flex flex-wrap items-center gap-3">
          <RecordStatusBadge status={record.status} />
          {record.verified_at && <span className="text-xs text-gi-text-subtle">verificirano {formatDate(record.verified_at)}</span>}
          {record.document && <DocumentPreviewDialog fetchUrl={docPreviewUrl(record.document.id)} filename={record.document.original_filename} mimeType={record.document.mime_type} />}
        </div>
      ) : (
        <p className="text-sm text-gi-text-subtle italic">Nema evidencije</p>
      )}
      <div>
        <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Učitaj dokument</label>
        <DocFilePicker file={file} onChange={setFile} />
      </div>
      {file && (
        <button onClick={handleUpload} disabled={uploading} className="text-xs font-semibold text-gi-navy bg-gi-yellow hover:bg-gi-yellow-hover px-3 py-1.5 rounded-lg transition-colors disabled:opacity-50">
          {uploading ? 'Učitava…' : 'Učitaj'}
        </button>
      )}
      {record && record.status !== 'verified' && (
        <button onClick={handleVerify} disabled={verifying} className="text-xs font-medium text-gi-text border border-gi-border hover:border-gi-text-subtle px-3 py-1.5 rounded-lg transition-colors disabled:opacity-50">
          {verifying ? 'Verificira…' : 'Verificiraj'}
        </button>
      )}
      {error && <p className="text-xs text-red-400">{error}</p>}
    </div>
  )
}

// ── Section: Contract ─────────────────────────────────────────────────────────

function ContractSection({ employeeId, initial, onRefresh }: { employeeId: string; initial?: EmploymentContract; onRefresh: () => void }) {
  const [contracts, setContracts]     = useState<EmploymentContract[]>([])
  const [historyOpen, setHistoryOpen] = useState(false)
  const [histLoading, setHistLoading] = useState(false)
  const [formOpen, setFormOpen]       = useState(false)
  const [terminateOpen, setTermOpen]  = useState(false)
  const [contractType, setContType]   = useState<'neodredeno' | 'odredeno'>('neodredeno')
  const [expiresAt, setExpires]       = useState('')
  const [terminatedAt, setTermAt]     = useState('')
  const [file, setFile]               = useState<File | null>(null)
  const [termFile, setTermFile]       = useState<File | null>(null)
  const [submitting, setSubmitting]   = useState(false)
  const [terminating, setTerminating] = useState(false)
  const [error, setError]             = useState<string | null>(null)
  const [termError, setTermError]     = useState<string | null>(null)

  const loadHistory = useCallback(async () => {
    setHistLoading(true)
    try { const r = await apiClient.get(`/documentation/employees/${employeeId}/contracts`); setContracts(r.data.contracts ?? []) }
    finally { setHistLoading(false) }
  }, [employeeId])

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault(); setSubmitting(true); setError(null)
    try {
      let fileId: string | undefined
      if (file) fileId = await uploadDocFile(employeeId, file)
      await apiClient.post(`/documentation/employees/${employeeId}/contracts`, {
        contract_type: contractType,
        ...(contractType === 'odredeno' && expiresAt ? { expires_at: expiresAt } : {}),
        ...(fileId ? { document_id: fileId } : {}),
      })
      setFormOpen(false); setFile(null); setExpires(''); setContType('neodredeno'); onRefresh(); loadHistory()
    } catch (e) { setError(errMsg(e)) } finally { setSubmitting(false) }
  }

  async function handleTerminate(e: React.FormEvent) {
    e.preventDefault()
    if (!initial) return; setTerminating(true); setTermError(null)
    try {
      let termFileId: string | undefined
      if (termFile) termFileId = await uploadDocFile(employeeId, termFile)
      await apiClient.post(`/documentation/contracts/${initial.id}/terminate`, {
        terminated_at: terminatedAt, ...(termFileId ? { termination_document_id: termFileId } : {}),
      })
      setTermOpen(false); setTermAt(''); setTermFile(null); onRefresh(); loadHistory()
    } catch (e) { setTermError(errMsg(e)) } finally { setTerminating(false) }
  }

  return (
    <div className="space-y-3">
      {initial ? (
        <div className="rounded-lg border border-gi-border bg-gi-card-elevated px-4 py-3 space-y-1.5">
          <div className="flex flex-wrap gap-4">
            <div>
              <p className="text-xs text-gi-text-muted">Vrsta</p>
              <p className="text-sm text-gi-text">{initial.contract_type === 'odredeno' ? 'Na određeno' : 'Na neodređeno'}</p>
            </div>
            {initial.expires_at && (
              <div>
                <p className="text-xs text-gi-text-muted">Istječe</p>
                <p className="text-sm text-gi-text">{formatDate(initial.expires_at)}</p>
              </div>
            )}
            {initial.terminated_at && (
              <div>
                <p className="text-xs text-gi-text-muted">Raskinut</p>
                <p className="text-sm text-red-400">{formatDate(initial.terminated_at)}</p>
              </div>
            )}
          </div>
          <div className="flex gap-3 flex-wrap">
            {initial.document && <DocumentPreviewDialog fetchUrl={docPreviewUrl(initial.document.id)} filename={initial.document.original_filename} mimeType={initial.document.mime_type} triggerLabel="Pregledaj ugovor" />}
            {initial.termination_document && <DocumentPreviewDialog fetchUrl={docPreviewUrl(initial.termination_document.id)} filename={initial.termination_document.original_filename} mimeType={initial.termination_document.mime_type} triggerLabel="Pregledaj raskid" />}
          </div>
        </div>
      ) : (
        <p className="text-sm text-gi-text-subtle italic">Nema aktivnog ugovora</p>
      )}
      <div className="flex flex-wrap gap-2">
        <button onClick={() => setFormOpen(v => !v)} className="text-xs font-semibold text-gi-navy bg-gi-yellow hover:bg-gi-yellow-hover px-3 py-1.5 rounded-lg transition-colors">{formOpen ? 'Zatvori' : 'Novi ugovor'}</button>
        {initial && !initial.terminated_at && (
          <button onClick={() => setTermOpen(v => !v)} className="text-xs font-medium text-red-400 border border-red-900/50 hover:border-red-700/70 px-3 py-1.5 rounded-lg transition-colors">{terminateOpen ? 'Zatvori' : 'Raskinuti ugovor'}</button>
        )}
        <button onClick={() => { if (!historyOpen) loadHistory(); setHistoryOpen(v => !v) }} className="text-xs text-gi-text-muted border border-gi-border hover:border-gi-text-subtle px-3 py-1.5 rounded-lg transition-colors">{historyOpen ? 'Sakrij povijest' : 'Prikaži povijest'}</button>
      </div>
      {formOpen && (
        <form onSubmit={handleCreate} className="space-y-3 border-t border-gi-border pt-3">
          <div>
            <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Vrsta ugovora *</label>
            <select value={contractType} onChange={e => setContType(e.target.value as 'neodredeno' | 'odredeno')} className={inputCls}>
              <option value="neodredeno">Na neodređeno</option>
              <option value="odredeno">Na određeno</option>
            </select>
          </div>
          {contractType === 'odredeno' && (
            <div>
              <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Datum isteka *</label>
              <input type="date" required value={expiresAt} onChange={e => setExpires(e.target.value)} className={inputCls} />
            </div>
          )}
          <div>
            <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Dokument ugovora (neobavezno)</label>
            <DocFilePicker file={file} onChange={setFile} />
          </div>
          {error && <p className="text-xs text-red-400">{error}</p>}
          <div className="flex gap-2">
            <button type="submit" disabled={submitting} className="text-xs font-semibold text-gi-navy bg-gi-yellow hover:bg-gi-yellow-hover px-3 py-1.5 rounded-lg transition-colors disabled:opacity-50">{submitting ? 'Sprema…' : 'Spremi'}</button>
            <button type="button" onClick={() => setFormOpen(false)} className="text-xs text-gi-text-muted border border-gi-border px-3 py-1.5 rounded-lg transition-colors">Odustani</button>
          </div>
        </form>
      )}
      {terminateOpen && (
        <form onSubmit={handleTerminate} className="space-y-3 border-t border-red-900/40 pt-3 mt-1">
          <p className="text-xs font-medium text-red-400">Raskid aktivnog ugovora</p>
          <div>
            <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Datum raskida *</label>
            <input type="date" required value={terminatedAt} onChange={e => setTermAt(e.target.value)} className={inputCls} />
          </div>
          <div>
            <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Dokument raskida (neobavezno)</label>
            <DocFilePicker file={termFile} onChange={setTermFile} />
          </div>
          {termError && <p className="text-xs text-red-400">{termError}</p>}
          <div className="flex gap-2">
            <button type="submit" disabled={terminating} className="text-xs font-semibold text-white bg-red-700 hover:bg-red-600 px-3 py-1.5 rounded-lg transition-colors disabled:opacity-50">{terminating ? 'Raskida…' : 'Potvrdi raskid'}</button>
            <button type="button" onClick={() => setTermOpen(false)} className="text-xs text-gi-text-muted border border-gi-border px-3 py-1.5 rounded-lg transition-colors">Odustani</button>
          </div>
        </form>
      )}
      {historyOpen && (
        <div className="border-t border-gi-border pt-3 space-y-2">
          {histLoading && <p className="text-xs text-gi-text-subtle animate-pulse">Učitavanje…</p>}
          <ShowMoreList items={contracts} render={(c) => (
            <div key={c.id} className="text-xs text-gi-text-muted space-y-0.5">
              <div>
                {c.contract_type === 'odredeno' ? 'Na određeno' : 'Na neodređeno'}
                {c.expires_at && ` · istj. ${formatDate(c.expires_at)}`}
                {c.terminated_at && <span className="text-red-400"> · Raskinut {formatDate(c.terminated_at)}</span>}
              </div>
              <div className="flex gap-3">
                {c.document && <DocumentPreviewDialog fetchUrl={docPreviewUrl(c.document.id)} filename={c.document.original_filename} mimeType={c.document.mime_type} triggerLabel="Ugovor" />}
                {c.termination_document && <DocumentPreviewDialog fetchUrl={docPreviewUrl(c.termination_document.id)} filename={c.termination_document.original_filename} mimeType={c.termination_document.mime_type} triggerLabel="Raskid" />}
              </div>
            </div>
          )} />
          {!histLoading && contracts.length === 0 && <p className="text-xs text-gi-text-subtle">Nema evidencije</p>}
        </div>
      )}
    </div>
  )
}

// ── Section: Payroll ──────────────────────────────────────────────────────────

function PayrollSection({ employeeId, onRefresh }: { employeeId: string; onRefresh: () => void }) {
  const months = recentMonthOptions()
  const prevMonth = months[1]?.value ?? months[0]?.value ?? ''
  const [obls, setObls]               = useState<MonthlyObligation[]>([])
  const [listOpen, setListOpen]       = useState(false)
  const [oblLoading, setOblLoading]   = useState(false)
  const [formOpen, setFormOpen]       = useState(false)
  const [month, setMonth]             = useState(prevMonth)
  const [file, setFile]               = useState<File | null>(null)
  const [submitting, setSubmitting]   = useState(false)
  const [error, setError]             = useState<string | null>(null)

  const loadObls = useCallback(async () => {
    setOblLoading(true)
    try { const r = await apiClient.get(`/documentation/employees/${employeeId}/obligations`); setObls((r.data.obligations ?? []).filter((o: MonthlyObligation) => o.obligation_type === 'platna_lista')) }
    finally { setOblLoading(false) }
  }, [employeeId])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault(); setSubmitting(true); setError(null)
    try {
      let fileId: string | undefined
      if (file) fileId = await uploadDocFile(employeeId, file)
      const { year, month: m } = parseYM(month)
      await apiClient.post(`/documentation/employees/${employeeId}/payroll`, { year, month: m, ...(fileId ? { document_id: fileId } : {}) })
      setFormOpen(false); setFile(null); onRefresh(); loadObls()
    } catch (e) { setError(errMsg(e)) } finally { setSubmitting(false) }
  }

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap gap-2">
        <button onClick={() => setFormOpen(v => !v)} className="text-xs font-semibold text-gi-navy bg-gi-yellow hover:bg-gi-yellow-hover px-3 py-1.5 rounded-lg transition-colors">{formOpen ? 'Zatvori' : 'Označi završenom'}</button>
        <button onClick={() => { if (!listOpen) loadObls(); setListOpen(v => !v) }} className="text-xs text-gi-text-muted border border-gi-border hover:border-gi-text-subtle px-3 py-1.5 rounded-lg transition-colors">{listOpen ? 'Sakrij zapise' : 'Prikaži zapise'}</button>
      </div>
      {formOpen && (
        <form onSubmit={handleSubmit} className="space-y-3 border-t border-gi-border pt-3">
          <div>
            <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Mjesec *</label>
            <select value={month} onChange={e => setMonth(e.target.value)} className={inputCls}>
              {months.map(m => <option key={m.value} value={m.value}>{m.label}</option>)}
            </select>
          </div>
          <div>
            <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Dokument platne liste (neobavezno)</label>
            <DocFilePicker file={file} onChange={setFile} />
          </div>
          {error && <p className="text-xs text-red-400">{error}</p>}
          <div className="flex gap-2">
            <button type="submit" disabled={submitting} className="text-xs font-semibold text-gi-navy bg-gi-yellow hover:bg-gi-yellow-hover px-3 py-1.5 rounded-lg transition-colors disabled:opacity-50">{submitting ? 'Sprema…' : 'Spremi'}</button>
            <button type="button" onClick={() => setFormOpen(false)} className="text-xs text-gi-text-muted border border-gi-border px-3 py-1.5 rounded-lg transition-colors">Odustani</button>
          </div>
        </form>
      )}
      {listOpen && (
        <div className="border-t border-gi-border pt-3 space-y-1.5">
          {oblLoading && <p className="text-xs text-gi-text-subtle animate-pulse">Učitavanje…</p>}
          <ShowMoreList items={obls} render={(o) => (
            <div key={o.id} className="flex flex-wrap justify-between text-xs text-gi-text-muted gap-2">
              <span>{HR_MONTHS[o.month]} {o.year}</span>
              <div className="flex items-center gap-2">
                <RecordStatusBadge status={o.status} />
                {o.document && <DocumentPreviewDialog fetchUrl={docPreviewUrl(o.document.id)} filename={o.document.original_filename} mimeType={o.document.mime_type} />}
              </div>
            </div>
          )} />
          {!oblLoading && obls.length === 0 && <p className="text-xs text-gi-text-subtle">Nema evidencije</p>}
        </div>
      )}
    </div>
  )
}

// ── Section: Timesheet ────────────────────────────────────────────────────────

function TimesheetSection({ employeeId, onRefresh }: { employeeId: string; onRefresh: () => void }) {
  const months = recentMonthOptions()
  const prevMonth = months[1]?.value ?? months[0]?.value ?? ''
  const [obls, setObls]             = useState<MonthlyObligation[]>([])
  const [listOpen, setListOpen]     = useState(false)
  const [oblLoading, setOblLoading] = useState(false)
  const [formOpen, setFormOpen]     = useState(false)
  const [month, setMonth]           = useState(prevMonth)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError]           = useState<string | null>(null)

  const loadObls = useCallback(async () => {
    setOblLoading(true)
    try { const r = await apiClient.get(`/documentation/employees/${employeeId}/obligations`); setObls((r.data.obligations ?? []).filter((o: MonthlyObligation) => o.obligation_type === 'evidencija_radnika')) }
    finally { setOblLoading(false) }
  }, [employeeId])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault(); setSubmitting(true); setError(null)
    try {
      const { year, month: m } = parseYM(month)
      await apiClient.post(`/documentation/employees/${employeeId}/timesheet`, { year, month: m })
      setFormOpen(false); onRefresh(); loadObls()
    } catch (e) { setError(errMsg(e)) } finally { setSubmitting(false) }
  }

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap gap-2">
        <button onClick={() => setFormOpen(v => !v)} className="text-xs font-semibold text-gi-navy bg-gi-yellow hover:bg-gi-yellow-hover px-3 py-1.5 rounded-lg transition-colors">{formOpen ? 'Zatvori' : 'Potvrdi evidenciju'}</button>
        <button onClick={() => { if (!listOpen) loadObls(); setListOpen(v => !v) }} className="text-xs text-gi-text-muted border border-gi-border hover:border-gi-text-subtle px-3 py-1.5 rounded-lg transition-colors">{listOpen ? 'Sakrij zapise' : 'Prikaži zapise'}</button>
      </div>
      {formOpen && (
        <form onSubmit={handleSubmit} className="space-y-3 border-t border-gi-border pt-3">
          <div>
            <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Mjesec *</label>
            <select value={month} onChange={e => setMonth(e.target.value)} className={inputCls}>
              {months.map(m => <option key={m.value} value={m.value}>{m.label}</option>)}
            </select>
          </div>
          <p className="text-xs text-gi-text-subtle">Sati se automatski preuzimaju iz evidencije radnih sati.</p>
          {error && <p className="text-xs text-red-400">{error}</p>}
          <div className="flex gap-2">
            <button type="submit" disabled={submitting} className="text-xs font-semibold text-gi-navy bg-gi-yellow hover:bg-gi-yellow-hover px-3 py-1.5 rounded-lg transition-colors disabled:opacity-50">{submitting ? 'Potvrđuje…' : 'Potvrdi'}</button>
            <button type="button" onClick={() => setFormOpen(false)} className="text-xs text-gi-text-muted border border-gi-border px-3 py-1.5 rounded-lg transition-colors">Odustani</button>
          </div>
        </form>
      )}
      {listOpen && (
        <div className="border-t border-gi-border pt-3 space-y-1.5">
          {oblLoading && <p className="text-xs text-gi-text-subtle animate-pulse">Učitavanje…</p>}
          <ShowMoreList items={obls} render={(o) => (
            <div key={o.id} className="flex flex-wrap justify-between text-xs text-gi-text-muted gap-2">
              <span>{HR_MONTHS[o.month]} {o.year}</span>
              <div className="flex items-center gap-2">
                {o.hours != null && <span className="text-gi-text-subtle">{o.hours}h</span>}
                <RecordStatusBadge status={o.status} />
              </div>
            </div>
          )} />
          {!oblLoading && obls.length === 0 && <p className="text-xs text-gi-text-subtle">Nema evidencije</p>}
        </div>
      )}
    </div>
  )
}

// ── Section: Work permits ─────────────────────────────────────────────────────

function PermitSection({ employeeId, initial, onRefresh }: { employeeId: string; initial?: WorkPermit; onRefresh: () => void }) {
  const [permits, setPermits]         = useState<WorkPermit[]>([])
  const [historyOpen, setHistoryOpen] = useState(false)
  const [histLoading, setHistLoading] = useState(false)
  const [formOpen, setFormOpen]       = useState(false)
  const [enteredAt, setEnteredAt]     = useState('')
  const [expiresAt, setExpiresAt]     = useState('')
  const [file, setFile]               = useState<File | null>(null)
  const [submitting, setSubmitting]   = useState(false)
  const [error, setError]             = useState<string | null>(null)

  const loadHistory = useCallback(async () => {
    setHistLoading(true)
    try { const r = await apiClient.get(`/documentation/employees/${employeeId}/work-permits`); setPermits(r.data.permits ?? []) }
    finally { setHistLoading(false) }
  }, [employeeId])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault(); setSubmitting(true); setError(null)
    try {
      let fileId: string | undefined
      if (file) fileId = await uploadDocFile(employeeId, file)
      await apiClient.post(`/documentation/employees/${employeeId}/work-permits`, {
        entered_at: enteredAt, expires_at: expiresAt, ...(fileId ? { document_id: fileId } : {}),
      })
      setFormOpen(false); setEnteredAt(''); setExpiresAt(''); setFile(null); onRefresh(); loadHistory()
    } catch (e) { setError(errMsg(e)) } finally { setSubmitting(false) }
  }

  return (
    <div className="space-y-3">
      {initial ? (
        <div className="rounded-lg border border-gi-border bg-gi-card-elevated px-4 py-3 space-y-1">
          <div className="flex flex-wrap gap-4">
            <div><p className="text-xs text-gi-text-muted">Datum ulaska</p><p className="text-sm text-gi-text">{formatDate(initial.entered_at)}</p></div>
            <div><p className="text-xs text-gi-text-muted">Istječe</p><p className="text-sm text-gi-text">{formatDate(initial.expires_at)}</p></div>
          </div>
          {initial.document && <DocumentPreviewDialog fetchUrl={docPreviewUrl(initial.document.id)} filename={initial.document.original_filename} mimeType={initial.document.mime_type} />}
        </div>
      ) : (
        <p className="text-sm text-gi-text-subtle italic">Nema evidencije</p>
      )}
      <div className="flex flex-wrap gap-2">
        <button onClick={() => setFormOpen(v => !v)} className="text-xs font-semibold text-gi-navy bg-gi-yellow hover:bg-gi-yellow-hover px-3 py-1.5 rounded-lg transition-colors">{formOpen ? 'Zatvori' : 'Nova dozvola'}</button>
        <button onClick={() => { if (!historyOpen) loadHistory(); setHistoryOpen(v => !v) }} className="text-xs text-gi-text-muted border border-gi-border hover:border-gi-text-subtle px-3 py-1.5 rounded-lg transition-colors">{historyOpen ? 'Sakrij povijest' : 'Prikaži povijest'}</button>
      </div>
      {formOpen && (
        <form onSubmit={handleSubmit} className="space-y-3 border-t border-gi-border pt-3">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Datum ulaska *</label>
              <input type="date" required value={enteredAt} onChange={e => setEnteredAt(e.target.value)} className={inputCls} />
            </div>
            <div>
              <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Datum isteka *</label>
              <input type="date" required value={expiresAt} onChange={e => setExpiresAt(e.target.value)} className={inputCls} />
            </div>
          </div>
          <div>
            <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Dokument dozvole (neobavezno)</label>
            <DocFilePicker file={file} onChange={setFile} />
          </div>
          {error && <p className="text-xs text-red-400">{error}</p>}
          <div className="flex gap-2">
            <button type="submit" disabled={submitting} className="text-xs font-semibold text-gi-navy bg-gi-yellow hover:bg-gi-yellow-hover px-3 py-1.5 rounded-lg transition-colors disabled:opacity-50">{submitting ? 'Sprema…' : 'Spremi'}</button>
            <button type="button" onClick={() => setFormOpen(false)} className="text-xs text-gi-text-muted border border-gi-border px-3 py-1.5 rounded-lg transition-colors">Odustani</button>
          </div>
        </form>
      )}
      {historyOpen && (
        <div className="border-t border-gi-border pt-3 space-y-1.5">
          {histLoading && <p className="text-xs text-gi-text-subtle animate-pulse">Učitavanje…</p>}
          <ShowMoreList items={permits} render={(p) => (
            <div key={p.id} className="flex flex-wrap justify-between text-xs text-gi-text-muted gap-2">
              <span>Ulaz: {formatDate(p.entered_at)}</span>
              <span className="flex items-center gap-2 text-gi-text-subtle">
                istj. {formatDate(p.expires_at)}
                {p.document && <DocumentPreviewDialog fetchUrl={docPreviewUrl(p.document.id)} filename={p.document.original_filename} mimeType={p.document.mime_type} />}
              </span>
            </div>
          )} />
          {!histLoading && permits.length === 0 && <p className="text-xs text-gi-text-subtle">Nema evidencije</p>}
        </div>
      )}
    </div>
  )
}

// ── Section: Pension ──────────────────────────────────────────────────────────

function PensionSection({ employeeId, record, onRefresh }: { employeeId: string; record?: PensionRecord; onRefresh: () => void }) {
  const [file, setFile]           = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [verifying, setVerifying] = useState(false)
  const [error, setError]         = useState<string | null>(null)

  async function handleUpload() {
    if (!file) return; setUploading(true); setError(null)
    try { const fileId = await uploadDocFile(employeeId, file); await apiClient.put(`/documentation/employees/${employeeId}/pension`, { document_id: fileId }); setFile(null); onRefresh() }
    catch (e) { setError(errMsg(e)) } finally { setUploading(false) }
  }
  async function handleVerify() {
    setVerifying(true); setError(null)
    try { await apiClient.post(`/documentation/employees/${employeeId}/pension/verify`); onRefresh() }
    catch (e) { setError(errMsg(e)) } finally { setVerifying(false) }
  }

  return (
    <div className="space-y-3">
      {record ? (
        <div className="rounded-lg border border-gi-border bg-gi-card-elevated px-4 py-3 flex flex-wrap items-center gap-3">
          <RecordStatusBadge status={record.status} />
          {record.verified_at && <span className="text-xs text-gi-text-subtle">verificirano {formatDate(record.verified_at)}</span>}
          {record.document && <DocumentPreviewDialog fetchUrl={docPreviewUrl(record.document.id)} filename={record.document.original_filename} mimeType={record.document.mime_type} />}
        </div>
      ) : (
        <p className="text-sm text-gi-text-subtle italic">Nema evidencije</p>
      )}
      <div>
        <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Učitaj dokument</label>
        <DocFilePicker file={file} onChange={setFile} />
      </div>
      <div className="flex gap-2">
        {file && <button onClick={handleUpload} disabled={uploading} className="text-xs font-semibold text-gi-navy bg-gi-yellow hover:bg-gi-yellow-hover px-3 py-1.5 rounded-lg transition-colors disabled:opacity-50">{uploading ? 'Učitava…' : 'Učitaj'}</button>}
        {record && record.status !== 'verified' && <button onClick={handleVerify} disabled={verifying} className="text-xs font-medium text-gi-text border border-gi-border hover:border-gi-text-subtle px-3 py-1.5 rounded-lg transition-colors disabled:opacity-50">{verifying ? 'Verificira…' : 'Verificiraj'}</button>}
      </div>
      {error && <p className="text-xs text-red-400">{error}</p>}
    </div>
  )
}

// ── Section: Health ───────────────────────────────────────────────────────────

function HealthSection({ employeeId, record, onRefresh }: { employeeId: string; record?: HealthRecord; onRefresh: () => void }) {
  const [file, setFile]           = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [verifying, setVerifying] = useState(false)
  const [error, setError]         = useState<string | null>(null)

  async function handleUpload() {
    if (!file) return; setUploading(true); setError(null)
    try { const fileId = await uploadDocFile(employeeId, file); await apiClient.put(`/documentation/employees/${employeeId}/health`, { document_id: fileId }); setFile(null); onRefresh() }
    catch (e) { setError(errMsg(e)) } finally { setUploading(false) }
  }
  async function handleVerify() {
    setVerifying(true); setError(null)
    try { await apiClient.post(`/documentation/employees/${employeeId}/health/verify`); onRefresh() }
    catch (e) { setError(errMsg(e)) } finally { setVerifying(false) }
  }

  return (
    <div className="space-y-3">
      {record ? (
        <div className="rounded-lg border border-gi-border bg-gi-card-elevated px-4 py-3 flex flex-wrap items-center gap-3">
          <RecordStatusBadge status={record.status} />
          {record.verified_at && <span className="text-xs text-gi-text-subtle">verificirano {formatDate(record.verified_at)}</span>}
          {record.document && <DocumentPreviewDialog fetchUrl={docPreviewUrl(record.document.id)} filename={record.document.original_filename} mimeType={record.document.mime_type} />}
        </div>
      ) : (
        <p className="text-sm text-gi-text-subtle italic">Nema evidencije</p>
      )}
      <div>
        <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Učitaj dokument</label>
        <DocFilePicker file={file} onChange={setFile} />
      </div>
      <div className="flex gap-2">
        {file && <button onClick={handleUpload} disabled={uploading} className="text-xs font-semibold text-gi-navy bg-gi-yellow hover:bg-gi-yellow-hover px-3 py-1.5 rounded-lg transition-colors disabled:opacity-50">{uploading ? 'Učitava…' : 'Učitaj'}</button>}
        {record && record.status !== 'verified' && <button onClick={handleVerify} disabled={verifying} className="text-xs font-medium text-gi-text border border-gi-border hover:border-gi-text-subtle px-3 py-1.5 rounded-lg transition-colors disabled:opacity-50">{verifying ? 'Verificira…' : 'Verificiraj'}</button>}
      </div>
      {error && <p className="text-xs text-red-400">{error}</p>}
    </div>
  )
}

// ── Section: Annual leave ─────────────────────────────────────────────────────

function AnnualLeaveSection({ employeeId, onRefresh }: { employeeId: string; onRefresh: () => void }) {
  const [leaves, setLeaves]       = useState<AnnualLeave[]>([])
  const [open, setOpen]           = useState(false)
  const [loading, setLoading]     = useState(false)
  const [formOpen, setFormOpen]   = useState(false)
  const [dateFrom, setDateFrom]   = useState('')
  const [dateTo, setDateTo]       = useState('')
  const [file, setFile]           = useState<File | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError]         = useState<string | null>(null)

  const loadLeaves = useCallback(async () => {
    setLoading(true)
    try { const r = await apiClient.get(`/documentation/employees/${employeeId}/annual-leave`); setLeaves(r.data.leaves ?? []) }
    finally { setLoading(false) }
  }, [employeeId])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault(); setSubmitting(true); setError(null)
    try {
      let fileId: string | undefined
      if (file) fileId = await uploadDocFile(employeeId, file)
      await apiClient.post(`/documentation/employees/${employeeId}/annual-leave`, {
        ...(dateFrom ? { date_from: dateFrom } : {}),
        ...(dateTo ? { date_to: dateTo } : {}),
        ...(fileId ? { document_id: fileId } : {}),
      })
      setFormOpen(false); setDateFrom(''); setDateTo(''); setFile(null); onRefresh(); loadLeaves()
    } catch (e) { setError(errMsg(e)) } finally { setSubmitting(false) }
  }

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap gap-2">
        <button onClick={() => setFormOpen(v => !v)} className="text-xs font-semibold text-gi-navy bg-gi-yellow hover:bg-gi-yellow-hover px-3 py-1.5 rounded-lg transition-colors">{formOpen ? 'Zatvori' : 'Dodaj godišnji'}</button>
        <button onClick={() => { if (!open) loadLeaves(); setOpen(v => !v) }} className="text-xs text-gi-text-muted border border-gi-border hover:border-gi-text-subtle px-3 py-1.5 rounded-lg transition-colors">{open ? 'Sakrij zapise' : 'Prikaži zapise'}</button>
      </div>
      {formOpen && (
        <form onSubmit={handleSubmit} className="space-y-3 border-t border-gi-border pt-3">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Od (neobavezno)</label>
              <input type="date" value={dateFrom} onChange={e => setDateFrom(e.target.value)} className={inputCls} />
            </div>
            <div>
              <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Do (neobavezno)</label>
              <input type="date" value={dateTo} onChange={e => setDateTo(e.target.value)} className={inputCls} />
            </div>
          </div>
          <div>
            <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Dokument (neobavezno)</label>
            <DocFilePicker file={file} onChange={setFile} />
          </div>
          {error && <p className="text-xs text-red-400">{error}</p>}
          <div className="flex gap-2">
            <button type="submit" disabled={submitting} className="text-xs font-semibold text-gi-navy bg-gi-yellow hover:bg-gi-yellow-hover px-3 py-1.5 rounded-lg transition-colors disabled:opacity-50">{submitting ? 'Sprema…' : 'Spremi'}</button>
            <button type="button" onClick={() => setFormOpen(false)} className="text-xs text-gi-text-muted border border-gi-border px-3 py-1.5 rounded-lg transition-colors">Odustani</button>
          </div>
        </form>
      )}
      {open && (
        <div className="border-t border-gi-border pt-3 space-y-1.5">
          {loading && <p className="text-xs text-gi-text-subtle animate-pulse">Učitavanje…</p>}
          <ShowMoreList items={leaves} render={(l) => (
            <div key={l.id} className="flex flex-wrap items-center justify-between text-xs text-gi-text-muted gap-2">
              <span>{l.date_from ? formatDate(l.date_from) : '—'}{l.date_to ? ` → ${formatDate(l.date_to)}` : ''}</span>
              {l.document && <DocumentPreviewDialog fetchUrl={docPreviewUrl(l.document.id)} filename={l.document.original_filename} mimeType={l.document.mime_type} />}
            </div>
          )} />
          {!loading && leaves.length === 0 && <p className="text-xs text-gi-text-subtle">Nema evidencije</p>}
        </div>
      )}
    </div>
  )
}

// ── Employee detail panel ─────────────────────────────────────────────────────

function EmployeeDetailPanel({
  employeeId, notifications, onClose,
}: { employeeId: string; notifications: DocNotification[]; onClose: () => void }) {
  const [summary, setSummary]       = useState<EmployeeDocSummary | null>(null)
  const [loading, setLoading]       = useState(true)
  const [error, setError]           = useState<string | null>(null)
  const [activeCat, setActiveCat]   = useState<CategoryId>('medical')

  const refresh = useCallback(async () => {
    try { const r = await apiClient.get(`/documentation/employees/${employeeId}/summary`); setSummary(r.data.summary) }
    catch { /* background */ }
  }, [employeeId])

  useEffect(() => {
    setLoading(true); setError(null)
    refresh().finally(() => setLoading(false))
  }, [refresh])

  // Reset to first category when employee changes
  useEffect(() => { setActiveCat('medical') }, [employeeId])

  if (loading) return (
    <div className="flex items-center justify-center h-32">
      <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje…</p>
    </div>
  )
  if (error || !summary) return <p className="text-red-400 text-sm p-4">{error ?? 'Nema podataka'}</p>

  const { employee } = summary
  const isRadnik = employee.role === 'radnik'
  const visibleCats = CATEGORIES.filter(c => !c.radnikOnly || isRadnik)

  function renderCategoryContent() {
    if (!summary) return null
    switch (activeCat) {
      case 'medical':   return <MedicalSection   employeeId={employeeId} initial={summary.latest_medical}   onRefresh={refresh} />
      case 'safety':    return <SafetySection     employeeId={employeeId} record={summary.safety_record}     onRefresh={refresh} />
      case 'contract':  return <ContractSection   employeeId={employeeId} initial={summary.active_contract}  onRefresh={refresh} />
      case 'payroll':   return <PayrollSection    employeeId={employeeId}                                    onRefresh={refresh} />
      case 'timesheet': return <TimesheetSection  employeeId={employeeId}                                    onRefresh={refresh} />
      case 'leave':     return <AnnualLeaveSection employeeId={employeeId}                                   onRefresh={refresh} />
      case 'permit':    return <PermitSection     employeeId={employeeId} initial={summary.latest_permit}    onRefresh={refresh} />
      case 'pension':   return <PensionSection    employeeId={employeeId} record={summary.pension_record}    onRefresh={refresh} />
      case 'health':    return <HealthSection     employeeId={employeeId} record={summary.health_record}     onRefresh={refresh} />
      default: return null
    }
  }

  return (
    <div className="space-y-4">
      {/* Employee header */}
      <div className="flex items-start justify-between gap-2">
        <div>
          <h3 className="text-base font-semibold text-gi-text">{employee.first_name} {employee.last_name}</h3>
          <p className="text-xs text-gi-text-muted capitalize mt-0.5">{employee.role} · {employee.active ? 'Aktivan' : 'Neaktivan'}</p>
        </div>
        <button onClick={onClose} className="text-gi-text-subtle hover:text-gi-text p-1 rounded-lg transition-colors shrink-0">
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      {/* Category navigation */}
      <div className="flex flex-wrap gap-1.5">
        {visibleCats.map(cat => {
          const status = getCategoryStatus(cat.id, employeeId, notifications)
          const isActive = activeCat === cat.id
          return (
            <button
              key={cat.id}
              onClick={() => setActiveCat(cat.id)}
              className={[
                'inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-medium transition-colors whitespace-nowrap',
                isActive
                  ? 'bg-gi-yellow text-gi-navy'
                  : 'bg-gi-card-elevated border border-gi-border text-gi-text-muted hover:border-gi-text-subtle hover:text-gi-text',
              ].join(' ')}
            >
              {!isActive && catStatusDot(status)}
              {cat.label}
            </button>
          )
        })}
      </div>

      {/* Category content */}
      <div className="rounded-xl border border-gi-border bg-gi-card p-4">
        <h4 className="text-xs font-semibold text-gi-text-subtle uppercase tracking-wider mb-3">
          {CATEGORIES.find(c => c.id === activeCat)?.label}
        </h4>
        {renderCategoryContent()}
      </div>
    </div>
  )
}

// ── Main page ─────────────────────────────────────────────────────────────────

export default function DokumentacijaPage() {
  const { user, employee, isLoading: authLoading, logout } = useAuth()
  const router = useRouter()

  const [tab, setTab]                               = useState<'pregled' | 'obavijesti'>('pregled')
  const [employees, setEmployees]                   = useState<DocEmployee[]>([])
  const [notifications, setNotifications]           = useState<DocNotification[]>([])
  const [search, setSearch]                         = useState('')
  const [selectedEmployeeId, setSelectedEmployeeId] = useState<string | null>(null)
  const [dataLoading, setDataLoading]               = useState(true)

  useEffect(() => {
    if (!authLoading && user && user.role !== 'administracija' && user.role !== 'direktor') router.replace('/dashboard')
  }, [user, authLoading, router])

  const loadData = useCallback(async () => {
    setDataLoading(true)
    try {
      const [empRes, notifRes] = await Promise.all([
        apiClient.get('/documentation/employees'),
        apiClient.get('/documentation/notifications'),
      ])
      setEmployees(empRes.data.employees ?? [])
      setNotifications(notifRes.data.notifications ?? [])
    } catch { /* silent */ } finally { setDataLoading(false) }
  }, [])

  useEffect(() => {
    if (user?.role === 'administracija' || user?.role === 'direktor') loadData()
  }, [user, loadData])

  if (authLoading) return <LoadingScreen />
  if (!user || (user.role !== 'administracija' && user.role !== 'direktor')) return null

  const filteredEmployees = employees.filter(e =>
    `${e.first_name} ${e.last_name}`.toLowerCase().includes(search.toLowerCase())
  )

  const urgentCount   = notifications.filter(n => n.priority === 'urgent' || n.priority === 'overdue').length
  const warningCount  = notifications.filter(n => n.priority === 'warning').length
  const lowCount      = notifications.filter(n => n.priority === 'low').length

  function selectEmployee(id: string) { setSelectedEmployeeId(id); setTab('pregled') }

  const tabCls = (active: boolean) => [
    'shrink-0 px-4 py-2.5 text-sm font-medium border-b-2 transition-colors',
    active ? 'border-gi-yellow text-gi-yellow' : 'border-transparent text-gi-text-muted hover:text-gi-text',
  ].join(' ')

  return (
    <AppShell user={user} employee={employee} title="Dokumentacija" backHref="/dashboard" onLogout={logout}>
      <div className="max-w-7xl space-y-5">
        <div>
          <h1 className="text-xl font-bold text-gi-text tracking-tight">Dokumentacija</h1>
          <p className="text-sm text-gi-text-subtle mt-0.5">Evidencija dokumentacije zaposlenika</p>
        </div>

        {/* Summary stat cards */}
        {!dataLoading && notifications.length > 0 && (
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
            {urgentCount > 0 && (
              <button onClick={() => setTab('obavijesti')} className="text-left rounded-xl border border-red-800/50 bg-red-950/20 px-4 py-3 hover:bg-red-950/30 transition-colors">
                <p className="text-xl font-bold text-red-400">{urgentCount}</p>
                <p className="text-xs text-red-400/70 mt-0.5">Hitno / Prekoračeno</p>
              </button>
            )}
            {warningCount > 0 && (
              <button onClick={() => setTab('obavijesti')} className="text-left rounded-xl border border-amber-700/40 bg-amber-950/20 px-4 py-3 hover:bg-amber-950/30 transition-colors">
                <p className="text-xl font-bold text-amber-300">{warningCount}</p>
                <p className="text-xs text-amber-300/70 mt-0.5">Upozorenje</p>
              </button>
            )}
            {lowCount > 0 && (
              <button onClick={() => setTab('obavijesti')} className="text-left rounded-xl border border-blue-800/40 bg-blue-950/20 px-4 py-3 hover:bg-blue-950/30 transition-colors">
                <p className="text-xl font-bold text-blue-300">{lowCount}</p>
                <p className="text-xs text-blue-300/70 mt-0.5">Nisko</p>
              </button>
            )}
          </div>
        )}

        {/* Tabs */}
        <div className="flex border-b border-gi-border">
          <button onClick={() => setTab('pregled')} className={tabCls(tab === 'pregled')}>Pregled zaposlenika</button>
          <button onClick={() => setTab('obavijesti')} className={tabCls(tab === 'obavijesti')}>
            Obavijesti
            {urgentCount > 0 && (
              <span className="ml-1.5 inline-flex items-center justify-center w-4 h-4 text-[10px] font-bold rounded-full bg-red-600 text-white">
                {urgentCount > 9 ? '9+' : urgentCount}
              </span>
            )}
          </button>
        </div>

        {dataLoading ? (
          <div className="rounded-xl border border-gi-border bg-gi-card py-16 text-center">
            <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje dokumentacije…</p>
          </div>
        ) : tab === 'pregled' ? (
          // ── Pregled tab: 2-column layout ──────────────────────────────────────
          <div className="flex flex-col lg:flex-row gap-5 items-start">
            {/* Left: employee list (always visible on desktop, hidden on mobile when employee is selected) */}
            <div className={`w-full lg:w-64 shrink-0 lg:sticky lg:top-5 space-y-3 ${selectedEmployeeId ? 'hidden lg:block' : ''}`}>
              <input
                type="search"
                placeholder="Traži zaposlenika…"
                value={search}
                onChange={e => setSearch(e.target.value)}
                className="w-full bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2 text-sm text-gi-text placeholder:text-gi-text-subtle focus:outline-none focus:border-gi-yellow transition-colors"
              />
              <div className="space-y-1.5">
                {filteredEmployees.length === 0 && (
                  <p className="text-sm text-gi-text-subtle px-1">Nema rezultata</p>
                )}
                {filteredEmployees.map(emp => {
                  const empNotifs = notifications.filter(n => n.employee_id === emp.id)
                  const urgent    = empNotifs.filter(n => n.priority === 'urgent' || n.priority === 'overdue').length
                  const warn      = empNotifs.filter(n => n.priority === 'warning').length
                  const isSelected = selectedEmployeeId === emp.id
                  return (
                    <button
                      key={emp.id}
                      onClick={() => setSelectedEmployeeId(emp.id)}
                      className={[
                        'w-full text-left rounded-xl border px-3 py-2.5 transition-colors',
                        isSelected
                          ? 'border-gi-yellow bg-gi-yellow/10'
                          : urgent > 0
                            ? 'border-red-800/50 bg-gi-card hover:bg-gi-card-elevated'
                            : 'border-gi-border bg-gi-card hover:bg-gi-card-elevated',
                      ].join(' ')}
                    >
                      <div className="flex items-start justify-between gap-1.5">
                        <div className="min-w-0">
                          <p className={`text-sm font-medium truncate ${isSelected ? 'text-gi-yellow' : 'text-gi-text'}`}>
                            {emp.first_name} {emp.last_name}
                          </p>
                          <p className="text-xs text-gi-text-muted capitalize">{emp.role}</p>
                        </div>
                        <div className="flex gap-1 shrink-0 mt-0.5">
                          {urgent > 0 && <span className="text-[10px] font-bold bg-red-900/60 text-red-400 px-1.5 py-0.5 rounded-full">{urgent}</span>}
                          {warn   > 0 && <span className="text-[10px] font-bold bg-amber-900/60 text-amber-300 px-1.5 py-0.5 rounded-full">{warn}</span>}
                        </div>
                      </div>
                    </button>
                  )
                })}
              </div>
            </div>

            {/* Right: detail or overview */}
            <div className="flex-1 min-w-0">
              {selectedEmployeeId ? (
                <div className="rounded-xl border border-gi-border bg-gi-card p-4">
                  {/* Mobile: back button */}
                  <button
                    onClick={() => setSelectedEmployeeId(null)}
                    className="lg:hidden mb-3 flex items-center gap-1.5 text-xs text-gi-text-muted hover:text-gi-text transition-colors"
                  >
                    <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M15 19l-7-7 7-7" />
                    </svg>
                    Natrag na listu
                  </button>
                  <EmployeeDetailPanel
                    employeeId={selectedEmployeeId}
                    notifications={notifications}
                    onClose={() => setSelectedEmployeeId(null)}
                  />
                </div>
              ) : (
                // Overview: employees needing attention + all employees summary
                <div className="space-y-5">
                  {notifications.length === 0 ? (
                    <div className="rounded-xl border border-gi-border bg-gi-card py-12 flex flex-col items-center gap-3">
                      <svg className="w-8 h-8 text-gi-text-subtle" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                        <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                      <p className="text-gi-text-subtle text-sm">Svi dokumenti uredni — nema aktivnih upozorenja.</p>
                    </div>
                  ) : (
                    <div>
                      <h3 className="text-xs font-semibold text-gi-text-subtle uppercase tracking-wider mb-3">
                        Zahtijeva pažnju
                      </h3>
                      <div className="space-y-2">
                        {employees
                          .filter(e => notifications.some(n => n.employee_id === e.id))
                          .sort((a, b) => {
                            const topPrio = (empId: string) => {
                              const n = notifications.filter(n => n.employee_id === empId)
                              if (n.some(x => x.priority === 'overdue')) return 4
                              if (n.some(x => x.priority === 'urgent'))  return 3
                              if (n.some(x => x.priority === 'warning')) return 2
                              return 1
                            }
                            return topPrio(b.id) - topPrio(a.id)
                          })
                          .slice(0, 10)
                          .map(emp => {
                            const empNotifs = notifications.filter(n => n.employee_id === emp.id)
                            const topNotif = empNotifs.reduce((a, b) =>
                              (PRIORITY_ORDER[b.priority] ?? 0) > (PRIORITY_ORDER[a.priority] ?? 0) ? b : a
                            )
                            return (
                              <button
                                key={emp.id}
                                onClick={() => selectEmployee(emp.id)}
                                className="w-full text-left rounded-xl border border-gi-border bg-gi-card px-4 py-3 hover:bg-gi-card-elevated/60 transition-colors"
                              >
                                <div className="flex items-start justify-between gap-3">
                                  <div className="min-w-0">
                                    <p className="text-sm font-medium text-gi-text">{emp.first_name} {emp.last_name}</p>
                                    <p className="text-xs text-gi-text-subtle mt-0.5 truncate">{topNotif.message}</p>
                                  </div>
                                  <div className="shrink-0 flex items-center gap-2">
                                    <PriorityBadge priority={topNotif.priority} />
                                  </div>
                                </div>
                              </button>
                            )
                          })}
                      </div>
                    </div>
                  )}

                  {/* Category overview */}
                  <div>
                    <h3 className="text-xs font-semibold text-gi-text-subtle uppercase tracking-wider mb-3">Po kategoriji</h3>
                    <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
                      {CATEGORIES.map(cat => {
                        const catNotifs = notifications.filter(n => NOTIF_TO_CATEGORY[n.type] === cat.id)
                        const urgent    = catNotifs.filter(n => n.priority === 'overdue' || n.priority === 'urgent').length
                        const warn      = catNotifs.filter(n => n.priority === 'warning' || n.priority === 'low').length
                        return (
                          <div key={cat.id} className="rounded-xl border border-gi-border bg-gi-card px-3 py-2.5">
                            <p className="text-xs font-medium text-gi-text-muted">{cat.label}</p>
                            {urgent > 0 ? (
                              <p className="text-xs text-red-400 mt-1">{urgent} hitno/prekoračeno</p>
                            ) : warn > 0 ? (
                              <p className="text-xs text-amber-300 mt-1">{warn} upozorenje</p>
                            ) : (
                              <p className="text-xs text-emerald-400/70 mt-1">Uredno</p>
                            )}
                          </div>
                        )
                      })}
                    </div>
                  </div>
                </div>
              )}
            </div>
          </div>
        ) : (
          // ── Obavijesti tab ──────────────────────────────────────────────────
          <div className="space-y-5">
            {notifications.length === 0 ? (
              <div className="rounded-xl border border-gi-border bg-gi-card py-14 flex flex-col items-center gap-3">
                <svg className="w-8 h-8 text-gi-text-subtle" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <p className="text-gi-text-subtle text-sm">Nema aktivnih obavijesti.</p>
              </div>
            ) : (
              (['overdue', 'urgent', 'warning', 'low'] as const).map(priority => {
                const group = notifications.filter(n => n.priority === priority)
                if (group.length === 0) return null
                const labels: Record<string, string> = {
                  overdue: 'Prekoračeno', urgent: 'Hitno', warning: 'Upozorenje', low: 'Nisko',
                }
                return (
                  <div key={priority}>
                    <h4 className="text-xs font-semibold text-gi-text-subtle uppercase tracking-wider mb-2.5">
                      {labels[priority]} ({group.length})
                    </h4>
                    <div className="space-y-2">
                      {group.map((n, i) => (
                        <div
                          key={`${n.type}-${n.employee_id}-${i}`}
                          className="rounded-xl border border-gi-border bg-gi-card px-4 py-3 flex items-start gap-3"
                        >
                          <div className="flex-1 min-w-0 space-y-0.5">
                            <p className="text-sm text-gi-text">{n.message}</p>
                            <p className="text-xs text-gi-text-muted">{n.employee_name}</p>
                            {n.due_date && <p className="text-xs text-gi-text-subtle">Rok: {formatDate(n.due_date)}</p>}
                            {n.year != null && n.month != null && (
                              <p className="text-xs text-gi-text-subtle">{HR_MONTHS[n.month]} {n.year}</p>
                            )}
                          </div>
                          <div className="shrink-0 flex flex-col items-end gap-2">
                            <PriorityBadge priority={n.priority} />
                            <button
                              onClick={() => selectEmployee(n.employee_id)}
                              className="text-xs text-gi-yellow hover:text-gi-yellow-hover transition-colors whitespace-nowrap"
                            >
                              Otvori →
                            </button>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )
              })
            )}
          </div>
        )}
      </div>
    </AppShell>
  )
}
