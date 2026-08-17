'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import DashboardShell from '@/components/layout/DashboardShell'
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

// ── Types & constants ─────────────────────────────────────────────────────────

type Tab = 'pregled' | 'obavijesti'

const TABS: { id: Tab; label: string }[] = [
  { id: 'pregled', label: 'Pregled zaposlenika' },
  { id: 'obavijesti', label: 'Obavijesti' },
]

const HR_MONTHS = ['', 'Siječanj', 'Veljača', 'Ožujak', 'Travanj', 'Svibanj', 'Lipanj',
  'Srpanj', 'Kolovoz', 'Rujan', 'Listopad', 'Studeni', 'Prosinac']

const inputCls = 'w-full px-3 py-2 rounded-lg bg-zinc-900 border border-zinc-700 text-sm text-zinc-200 placeholder-zinc-500 focus:outline-none focus:border-violet-500'
const btnPrimary = 'px-3 py-1.5 rounded-lg bg-violet-600 hover:bg-violet-700 text-white text-xs font-medium disabled:opacity-50 disabled:cursor-not-allowed transition'
const btnSecondary = 'px-3 py-1.5 rounded-lg bg-zinc-700 hover:bg-zinc-600 text-zinc-200 text-xs font-medium transition'
const btnDanger = 'px-3 py-1.5 rounded-lg bg-red-700 hover:bg-red-600 text-white text-xs font-medium disabled:opacity-50 transition'

// ── Helpers ───────────────────────────────────────────────────────────────────

function formatDate(s: string | null | undefined) {
  if (!s) return '—'
  const d = new Date(s)
  if (isNaN(d.getTime())) return s
  return d.toLocaleDateString('hr-HR')
}

function priorityBadge(priority: string) {
  switch (priority) {
    case 'low': return <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-blue-900/60 text-blue-300">Nisko</span>
    case 'warning': return <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-amber-900/60 text-amber-300">Upozorenje</span>
    case 'urgent': return <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-orange-900/60 text-orange-300">Hitno</span>
    case 'overdue': return <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-red-900/60 text-red-400">Prekoračeno</span>
    default: return null
  }
}

function statusBadge(status: string) {
  switch (status) {
    case 'pending': return <span className="text-xs text-zinc-400">Na čekanju</span>
    case 'uploaded': return <span className="text-xs text-amber-400">Učitano</span>
    case 'verified': return <span className="text-xs text-green-400">Verificirano</span>
    case 'completed': return <span className="text-xs text-green-400">Završeno</span>
    default: return <span className="text-xs text-zinc-500">{status}</span>
  }
}

function recentMonthOptions(n = 24) {
  const now = new Date()
  const out: { value: string; label: string }[] = []
  for (let i = 0; i < n; i++) {
    const d = new Date(now.getFullYear(), now.getMonth() - i, 1)
    const y = d.getFullYear()
    const m = d.getMonth() + 1
    out.push({ value: `${y}-${String(m).padStart(2, '0')}`, label: `${HR_MONTHS[m]} ${y}` })
  }
  return out
}

// Upload a file via the two-step protocol; returns the file_id.
async function uploadDocFile(employeeId: string, file: File): Promise<string> {
  const fd = new FormData()
  fd.append('file', file)
  const { data } = await apiClient.post(`/documentation/employees/${employeeId}/files`, fd)
  return data.file_id as string
}

function errMsg(e: unknown): string {
  return (e as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Neočekivana greška'
}

// ── Shared UI pieces ──────────────────────────────────────────────────────────

function SectionCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-xl border border-zinc-700/50 bg-zinc-800/50 p-4">
      <h4 className="text-sm font-semibold text-zinc-300 mb-3">{title}</h4>
      {children}
    </div>
  )
}

function FileInput({ file, onChange }: { file: File | null; onChange: (f: File | null) => void }) {
  const ref = useRef<HTMLInputElement>(null)
  return (
    <div className="flex items-center gap-2">
      <input ref={ref} type="file" className="hidden" onChange={e => onChange(e.target.files?.[0] ?? null)} />
      <button type="button" onClick={() => ref.current?.click()} className={btnSecondary}>
        {file ? 'Promijeni datoteku' : 'Odaberi datoteku'}
      </button>
      {file && <span className="text-xs text-zinc-400 truncate max-w-[160px]">{file.name}</span>}
    </div>
  )
}

// ── Medical exams ─────────────────────────────────────────────────────────────

function MedicalSection({ employeeId, initial, onRefresh }: {
  employeeId: string; initial?: MedicalExam; onRefresh: () => void
}) {
  const [exams, setExams] = useState<MedicalExam[]>([])
  const [historyOpen, setHistoryOpen] = useState(false)
  const [historyLoading, setHistoryLoading] = useState(false)
  const [formOpen, setFormOpen] = useState(false)
  const [completedDate, setCompletedDate] = useState('')
  const [expiresAt, setExpiresAt] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const loadHistory = useCallback(async () => {
    setHistoryLoading(true)
    try {
      const r = await apiClient.get(`/documentation/employees/${employeeId}/medical-exams`)
      setExams(r.data.exams ?? [])
    } finally {
      setHistoryLoading(false)
    }
  }, [employeeId])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSubmitting(true); setError(null)
    try {
      let fileId: string | undefined
      if (file) fileId = await uploadDocFile(employeeId, file)
      await apiClient.post(`/documentation/employees/${employeeId}/medical-exams`, {
        completed_date: completedDate,
        expires_at: expiresAt,
        ...(fileId ? { document_id: fileId } : {}),
      })
      setFormOpen(false); setCompletedDate(''); setExpiresAt(''); setFile(null)
      onRefresh()
      loadHistory()
    } catch (e) { setError(errMsg(e)) }
    finally { setSubmitting(false) }
  }

  return (
    <SectionCard title="Liječnički pregled">
      {initial
        ? <p className="text-sm text-zinc-200 mb-2">Istječe: <span className="font-medium">{formatDate(initial.expires_at)}</span></p>
        : <p className="text-sm text-zinc-400 italic mb-2">Nema evidencije</p>}
      <div className="flex flex-wrap items-center gap-2 mb-2">
        <button onClick={() => setFormOpen(v => !v)} className={btnPrimary}>
          {formOpen ? 'Zatvori' : 'Novi pregled'}
        </button>
        <button
          onClick={() => { if (!historyOpen) loadHistory(); setHistoryOpen(v => !v) }}
          className="text-xs text-violet-400 hover:underline"
        >
          {historyOpen ? 'Sakrij povijest' : 'Prikaži povijest'}
        </button>
      </div>
      {formOpen && (
        <form onSubmit={handleSubmit} className="space-y-2 border-t border-zinc-700 pt-3">
          <div>
            <label className="text-xs text-zinc-400 mb-1 block">Datum pregleda *</label>
            <input type="date" required value={completedDate} onChange={e => setCompletedDate(e.target.value)} className={inputCls} />
          </div>
          <div>
            <label className="text-xs text-zinc-400 mb-1 block">Istječe *</label>
            <input type="date" required value={expiresAt} onChange={e => setExpiresAt(e.target.value)} className={inputCls} />
          </div>
          <div>
            <label className="text-xs text-zinc-400 mb-1 block">Dokument (neobavezno)</label>
            <FileInput file={file} onChange={setFile} />
          </div>
          {error && <p className="text-xs text-red-400">{error}</p>}
          <div className="flex gap-2">
            <button type="submit" disabled={submitting} className={btnPrimary}>{submitting ? 'Sprema...' : 'Spremi'}</button>
            <button type="button" onClick={() => setFormOpen(false)} className={btnSecondary}>Odustani</button>
          </div>
        </form>
      )}
      {historyOpen && (
        <div className="mt-2 space-y-1 max-h-40 overflow-y-auto">
          {historyLoading && <p className="text-xs text-zinc-500">Učitavanje...</p>}
          {exams.map(ex => (
            <div key={ex.id} className="flex justify-between text-xs text-zinc-300">
              <span>{formatDate(ex.completed_date)}</span>
              <span className="text-zinc-400">istj. {formatDate(ex.expires_at)}</span>
            </div>
          ))}
          {!historyLoading && exams.length === 0 && <p className="text-xs text-zinc-500">Nema evidencije</p>}
        </div>
      )}
    </SectionCard>
  )
}

// ── Safety record ─────────────────────────────────────────────────────────────

function SafetySection({ employeeId, record, isRadnik, onRefresh }: {
  employeeId: string; record?: SafetyRecord; isRadnik: boolean; onRefresh: () => void
}) {
  const [file, setFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [verifying, setVerifying] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!isRadnik) return null

  async function handleUpload() {
    if (!file) return
    setUploading(true); setError(null)
    try {
      const fileId = await uploadDocFile(employeeId, file)
      await apiClient.put(`/documentation/employees/${employeeId}/safety`, { document_id: fileId })
      setFile(null); onRefresh()
    } catch (e) { setError(errMsg(e)) }
    finally { setUploading(false) }
  }

  async function handleVerify() {
    setVerifying(true); setError(null)
    try {
      await apiClient.post(`/documentation/employees/${employeeId}/safety/verify`)
      onRefresh()
    } catch (e) { setError(errMsg(e)) }
    finally { setVerifying(false) }
  }

  return (
    <SectionCard title="Zaštita na radu">
      {record
        ? <div className="flex items-center gap-2 mb-2">
            {statusBadge(record.status)}
            {record.verified_at && <span className="text-xs text-zinc-400">verificirano {formatDate(record.verified_at)}</span>}
          </div>
        : <p className="text-sm text-zinc-400 italic mb-2">Nema evidencije</p>}
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <FileInput file={file} onChange={setFile} />
          {file && (
            <button onClick={handleUpload} disabled={uploading} className={btnPrimary}>
              {uploading ? 'Učitava...' : 'Učitaj'}
            </button>
          )}
        </div>
        {record && record.status !== 'verified' && (
          <button onClick={handleVerify} disabled={verifying} className={btnPrimary}>
            {verifying ? 'Verificira...' : 'Verificiraj'}
          </button>
        )}
        {error && <p className="text-xs text-red-400">{error}</p>}
      </div>
    </SectionCard>
  )
}

// ── Contracts ─────────────────────────────────────────────────────────────────

function ContractSection({ employeeId, initial, onRefresh }: {
  employeeId: string; initial?: EmploymentContract; onRefresh: () => void
}) {
  const [contracts, setContracts] = useState<EmploymentContract[]>([])
  const [historyOpen, setHistoryOpen] = useState(false)
  const [historyLoading, setHistoryLoading] = useState(false)
  const [formOpen, setFormOpen] = useState(false)
  const [terminateOpen, setTerminateOpen] = useState(false)
  const [contractType, setContractType] = useState<'neodredeno' | 'odredeno'>('neodredeno')
  const [expiresAt, setExpiresAt] = useState('')
  const [terminatedAt, setTerminatedAt] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const [termFile, setTermFile] = useState<File | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [terminating, setTerminating] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [termError, setTermError] = useState<string | null>(null)

  const loadHistory = useCallback(async () => {
    setHistoryLoading(true)
    try {
      const r = await apiClient.get(`/documentation/employees/${employeeId}/contracts`)
      setContracts(r.data.contracts ?? [])
    } finally {
      setHistoryLoading(false)
    }
  }, [employeeId])

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    setSubmitting(true); setError(null)
    try {
      let fileId: string | undefined
      if (file) fileId = await uploadDocFile(employeeId, file)
      await apiClient.post(`/documentation/employees/${employeeId}/contracts`, {
        contract_type: contractType,
        ...(contractType === 'odredeno' && expiresAt ? { expires_at: expiresAt } : {}),
        ...(fileId ? { document_id: fileId } : {}),
      })
      setFormOpen(false); setFile(null); setExpiresAt(''); setContractType('neodredeno')
      onRefresh(); loadHistory()
    } catch (e) { setError(errMsg(e)) }
    finally { setSubmitting(false) }
  }

  async function handleTerminate(e: React.FormEvent) {
    e.preventDefault()
    if (!initial) return
    setTerminating(true); setTermError(null)
    try {
      let termFileId: string | undefined
      if (termFile) termFileId = await uploadDocFile(employeeId, termFile)
      await apiClient.post(`/documentation/contracts/${initial.id}/terminate`, {
        terminated_at: terminatedAt,
        ...(termFileId ? { termination_document_id: termFileId } : {}),
      })
      setTerminateOpen(false); setTerminatedAt(''); setTermFile(null)
      onRefresh(); loadHistory()
    } catch (e) { setTermError(errMsg(e)) }
    finally { setTerminating(false) }
  }

  return (
    <SectionCard title="Ugovor o radu">
      {initial
        ? <div className="text-sm text-zinc-200 space-y-0.5 mb-2">
            <p>Tip: <span className="font-medium">{initial.contract_type === 'odredeno' ? 'Na određeno' : 'Na neodređeno'}</span></p>
            {initial.expires_at && <p>Istječe: <span className="font-medium">{formatDate(initial.expires_at)}</span></p>}
            {initial.terminated_at && <p className="text-red-400">Raskinut: {formatDate(initial.terminated_at)}</p>}
          </div>
        : <p className="text-sm text-zinc-400 italic mb-2">Nema aktivnog ugovora</p>}
      <div className="flex flex-wrap items-center gap-2 mb-2">
        <button onClick={() => setFormOpen(v => !v)} className={btnPrimary}>
          {formOpen ? 'Zatvori' : 'Novi ugovor'}
        </button>
        {initial && !initial.terminated_at && (
          <button onClick={() => setTerminateOpen(v => !v)} className={btnDanger}>
            {terminateOpen ? 'Zatvori' : 'Raskinuti ugovor'}
          </button>
        )}
        <button
          onClick={() => { if (!historyOpen) loadHistory(); setHistoryOpen(v => !v) }}
          className="text-xs text-violet-400 hover:underline"
        >
          {historyOpen ? 'Sakrij povijest' : 'Prikaži povijest'}
        </button>
      </div>
      {formOpen && (
        <form onSubmit={handleCreate} className="space-y-2 border-t border-zinc-700 pt-3">
          <div>
            <label className="text-xs text-zinc-400 mb-1 block">Vrsta ugovora *</label>
            <select value={contractType} onChange={e => setContractType(e.target.value as 'neodredeno' | 'odredeno')} className={inputCls}>
              <option value="neodredeno">Na neodređeno</option>
              <option value="odredeno">Na određeno</option>
            </select>
          </div>
          {contractType === 'odredeno' && (
            <div>
              <label className="text-xs text-zinc-400 mb-1 block">Datum isteka *</label>
              <input type="date" required value={expiresAt} onChange={e => setExpiresAt(e.target.value)} className={inputCls} />
            </div>
          )}
          <div>
            <label className="text-xs text-zinc-400 mb-1 block">Dokument ugovora (neobavezno)</label>
            <FileInput file={file} onChange={setFile} />
          </div>
          {error && <p className="text-xs text-red-400">{error}</p>}
          <div className="flex gap-2">
            <button type="submit" disabled={submitting} className={btnPrimary}>{submitting ? 'Sprema...' : 'Spremi'}</button>
            <button type="button" onClick={() => setFormOpen(false)} className={btnSecondary}>Odustani</button>
          </div>
        </form>
      )}
      {terminateOpen && (
        <form onSubmit={handleTerminate} className="space-y-2 border-t border-red-700/40 pt-3 mt-2">
          <p className="text-xs text-red-400 font-medium">Raskid aktivnog ugovora</p>
          <div>
            <label className="text-xs text-zinc-400 mb-1 block">Datum raskida *</label>
            <input type="date" required value={terminatedAt} onChange={e => setTerminatedAt(e.target.value)} className={inputCls} />
          </div>
          <div>
            <label className="text-xs text-zinc-400 mb-1 block">Dokument raskida (neobavezno)</label>
            <FileInput file={termFile} onChange={setTermFile} />
          </div>
          {termError && <p className="text-xs text-red-400">{termError}</p>}
          <div className="flex gap-2">
            <button type="submit" disabled={terminating} className={btnDanger}>{terminating ? 'Raskida...' : 'Potvrdi raskid'}</button>
            <button type="button" onClick={() => setTerminateOpen(false)} className={btnSecondary}>Odustani</button>
          </div>
        </form>
      )}
      {historyOpen && (
        <div className="mt-2 space-y-1 max-h-40 overflow-y-auto">
          {historyLoading && <p className="text-xs text-zinc-500">Učitavanje...</p>}
          {contracts.map(c => (
            <div key={c.id} className="text-xs text-zinc-300">
              {c.contract_type === 'odredeno' ? 'Na određeno' : 'Na neodređeno'}
              {c.expires_at && ` · istj. ${formatDate(c.expires_at)}`}
              {c.terminated_at && <span className="text-red-400"> · Raskinut {formatDate(c.terminated_at)}</span>}
            </div>
          ))}
          {!historyLoading && contracts.length === 0 && <p className="text-xs text-zinc-500">Nema evidencije</p>}
        </div>
      )}
    </SectionCard>
  )
}

// ── Monthly obligations (payroll + timesheet) ─────────────────────────────────

function ObligationsSection({ employeeId, isRadnik, onRefresh }: {
  employeeId: string; isRadnik: boolean; onRefresh: () => void
}) {
  const [obls, setObls] = useState<MonthlyObligation[]>([])
  const [listOpen, setListOpen] = useState(false)
  const [oblLoading, setOblLoading] = useState(false)
  const [payrollFormOpen, setPayrollFormOpen] = useState(false)
  const [timesheetFormOpen, setTimesheetFormOpen] = useState(false)
  const months = recentMonthOptions()
  const prevMonth = months[1]?.value ?? months[0]?.value ?? ''
  const [payrollMonth, setPayrollMonth] = useState(prevMonth)
  const [timesheetMonth, setTimesheetMonth] = useState(prevMonth)
  const [payrollFile, setPayrollFile] = useState<File | null>(null)
  const [payrollSubmitting, setPayrollSubmitting] = useState(false)
  const [timesheetSubmitting, setTimesheetSubmitting] = useState(false)
  const [payrollError, setPayrollError] = useState<string | null>(null)
  const [timesheetError, setTimesheetError] = useState<string | null>(null)

  const loadObls = useCallback(async () => {
    setOblLoading(true)
    try {
      const r = await apiClient.get(`/documentation/employees/${employeeId}/obligations`)
      setObls(r.data.obligations ?? [])
    } finally {
      setOblLoading(false)
    }
  }, [employeeId])

  function parseYM(val: string) {
    const [y, m] = val.split('-').map(Number)
    return { year: y, month: m }
  }

  async function handlePayroll(e: React.FormEvent) {
    e.preventDefault()
    setPayrollSubmitting(true); setPayrollError(null)
    try {
      let fileId: string | undefined
      if (payrollFile) fileId = await uploadDocFile(employeeId, payrollFile)
      const { year, month } = parseYM(payrollMonth)
      await apiClient.post(`/documentation/employees/${employeeId}/payroll`, {
        year, month, ...(fileId ? { document_id: fileId } : {}),
      })
      setPayrollFormOpen(false); setPayrollFile(null)
      onRefresh(); loadObls()
    } catch (e) { setPayrollError(errMsg(e)) }
    finally { setPayrollSubmitting(false) }
  }

  async function handleTimesheet(e: React.FormEvent) {
    e.preventDefault()
    setTimesheetSubmitting(true); setTimesheetError(null)
    try {
      const { year, month } = parseYM(timesheetMonth)
      await apiClient.post(`/documentation/employees/${employeeId}/timesheet`, { year, month })
      setTimesheetFormOpen(false)
      onRefresh(); loadObls()
    } catch (e) { setTimesheetError(errMsg(e)) }
    finally { setTimesheetSubmitting(false) }
  }

  const payroll = obls.filter(o => o.obligation_type === 'platna_lista')
  const timesheet = obls.filter(o => o.obligation_type === 'evidencija_radnika')

  const oblList = (items: MonthlyObligation[], type: 'platna' | 'timesheet') => (
    <div className="mt-2 space-y-1 max-h-40 overflow-y-auto">
      {oblLoading && <p className="text-xs text-zinc-500">Učitavanje...</p>}
      {items.map(o => (
        <div key={o.id} className="flex justify-between text-xs text-zinc-300">
          <span>{HR_MONTHS[o.month]} {o.year}</span>
          <div className="flex items-center gap-2">
            {type === 'timesheet' && o.hours != null && <span className="text-zinc-400">{o.hours}h</span>}
            {statusBadge(o.status)}
          </div>
        </div>
      ))}
      {!oblLoading && items.length === 0 && <p className="text-xs text-zinc-500">Nema evidencije</p>}
    </div>
  )

  return (
    <>
      <SectionCard title="Platne liste">
        <div className="flex flex-wrap items-center gap-2 mb-2">
          <button onClick={() => setPayrollFormOpen(v => !v)} className={btnPrimary}>
            {payrollFormOpen ? 'Zatvori' : 'Označi završenom'}
          </button>
          <button
            onClick={() => { if (!listOpen) loadObls(); setListOpen(v => !v) }}
            className="text-xs text-violet-400 hover:underline"
          >
            {listOpen ? 'Sakrij zapise' : 'Prikaži zapise'}
          </button>
        </div>
        {payrollFormOpen && (
          <form onSubmit={handlePayroll} className="space-y-2 border-t border-zinc-700 pt-3">
            <div>
              <label className="text-xs text-zinc-400 mb-1 block">Mjesec *</label>
              <select value={payrollMonth} onChange={e => setPayrollMonth(e.target.value)} className={inputCls}>
                {months.map(m => <option key={m.value} value={m.value}>{m.label}</option>)}
              </select>
            </div>
            <div>
              <label className="text-xs text-zinc-400 mb-1 block">Dokument platne liste (neobavezno)</label>
              <FileInput file={payrollFile} onChange={setPayrollFile} />
            </div>
            {payrollError && <p className="text-xs text-red-400">{payrollError}</p>}
            <div className="flex gap-2">
              <button type="submit" disabled={payrollSubmitting} className={btnPrimary}>{payrollSubmitting ? 'Sprema...' : 'Spremi'}</button>
              <button type="button" onClick={() => setPayrollFormOpen(false)} className={btnSecondary}>Odustani</button>
            </div>
          </form>
        )}
        {listOpen && oblList(payroll, 'platna')}
      </SectionCard>

      {isRadnik && (
        <SectionCard title="Evidencija radnika">
          <div className="flex flex-wrap items-center gap-2 mb-2">
            <button onClick={() => setTimesheetFormOpen(v => !v)} className={btnPrimary}>
              {timesheetFormOpen ? 'Zatvori' : 'Potvrdi evidenciju'}
            </button>
            <button
              onClick={() => { if (!listOpen) loadObls(); setListOpen(v => !v) }}
              className="text-xs text-violet-400 hover:underline"
            >
              {listOpen ? 'Sakrij zapise' : 'Prikaži zapise'}
            </button>
          </div>
          {timesheetFormOpen && (
            <form onSubmit={handleTimesheet} className="space-y-2 border-t border-zinc-700 pt-3">
              <div>
                <label className="text-xs text-zinc-400 mb-1 block">Mjesec *</label>
                <select value={timesheetMonth} onChange={e => setTimesheetMonth(e.target.value)} className={inputCls}>
                  {months.map(m => <option key={m.value} value={m.value}>{m.label}</option>)}
                </select>
              </div>
              <p className="text-xs text-zinc-500">Sati se automatski preuzimaju iz evidencije radnih sati.</p>
              {timesheetError && <p className="text-xs text-red-400">{timesheetError}</p>}
              <div className="flex gap-2">
                <button type="submit" disabled={timesheetSubmitting} className={btnPrimary}>{timesheetSubmitting ? 'Sprema...' : 'Potvrdi'}</button>
                <button type="button" onClick={() => setTimesheetFormOpen(false)} className={btnSecondary}>Odustani</button>
              </div>
            </form>
          )}
          {listOpen && oblList(timesheet, 'timesheet')}
        </SectionCard>
      )}
    </>
  )
}

// ── Work permits ──────────────────────────────────────────────────────────────

function PermitSection({ employeeId, initial, onRefresh }: {
  employeeId: string; initial?: WorkPermit; onRefresh: () => void
}) {
  const [permits, setPermits] = useState<WorkPermit[]>([])
  const [historyOpen, setHistoryOpen] = useState(false)
  const [historyLoading, setHistoryLoading] = useState(false)
  const [formOpen, setFormOpen] = useState(false)
  const [enteredAt, setEnteredAt] = useState('')
  const [expiresAt, setExpiresAt] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const loadHistory = useCallback(async () => {
    setHistoryLoading(true)
    try {
      const r = await apiClient.get(`/documentation/employees/${employeeId}/work-permits`)
      setPermits(r.data.permits ?? [])
    } finally {
      setHistoryLoading(false)
    }
  }, [employeeId])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSubmitting(true); setError(null)
    try {
      let fileId: string | undefined
      if (file) fileId = await uploadDocFile(employeeId, file)
      await apiClient.post(`/documentation/employees/${employeeId}/work-permits`, {
        entered_at: enteredAt,
        expires_at: expiresAt,
        ...(fileId ? { document_id: fileId } : {}),
      })
      setFormOpen(false); setEnteredAt(''); setExpiresAt(''); setFile(null)
      onRefresh(); loadHistory()
    } catch (e) { setError(errMsg(e)) }
    finally { setSubmitting(false) }
  }

  return (
    <SectionCard title="Radna dozvola">
      {initial
        ? <p className="text-sm text-zinc-200 mb-2">Istječe: <span className="font-medium">{formatDate(initial.expires_at)}</span></p>
        : <p className="text-sm text-zinc-400 italic mb-2">Nema evidencije</p>}
      <div className="flex flex-wrap items-center gap-2 mb-2">
        <button onClick={() => setFormOpen(v => !v)} className={btnPrimary}>
          {formOpen ? 'Zatvori' : 'Nova dozvola'}
        </button>
        <button
          onClick={() => { if (!historyOpen) loadHistory(); setHistoryOpen(v => !v) }}
          className="text-xs text-violet-400 hover:underline"
        >
          {historyOpen ? 'Sakrij povijest' : 'Prikaži povijest'}
        </button>
      </div>
      {formOpen && (
        <form onSubmit={handleSubmit} className="space-y-2 border-t border-zinc-700 pt-3">
          <div>
            <label className="text-xs text-zinc-400 mb-1 block">Datum ulaska *</label>
            <input type="date" required value={enteredAt} onChange={e => setEnteredAt(e.target.value)} className={inputCls} />
          </div>
          <div>
            <label className="text-xs text-zinc-400 mb-1 block">Datum isteka *</label>
            <input type="date" required value={expiresAt} onChange={e => setExpiresAt(e.target.value)} className={inputCls} />
          </div>
          <div>
            <label className="text-xs text-zinc-400 mb-1 block">Dokument dozvole (neobavezno)</label>
            <FileInput file={file} onChange={setFile} />
          </div>
          {error && <p className="text-xs text-red-400">{error}</p>}
          <div className="flex gap-2">
            <button type="submit" disabled={submitting} className={btnPrimary}>{submitting ? 'Sprema...' : 'Spremi'}</button>
            <button type="button" onClick={() => setFormOpen(false)} className={btnSecondary}>Odustani</button>
          </div>
        </form>
      )}
      {historyOpen && (
        <div className="mt-2 space-y-1 max-h-40 overflow-y-auto">
          {historyLoading && <p className="text-xs text-zinc-500">Učitavanje...</p>}
          {permits.map(p => (
            <div key={p.id} className="flex justify-between text-xs text-zinc-300">
              <span>Ulaz: {formatDate(p.entered_at)}</span>
              <span className="text-zinc-400">istj. {formatDate(p.expires_at)}</span>
            </div>
          ))}
          {!historyLoading && permits.length === 0 && <p className="text-xs text-zinc-500">Nema evidencije</p>}
        </div>
      )}
    </SectionCard>
  )
}

// ── Pension ───────────────────────────────────────────────────────────────────

function PensionSection({ employeeId, record, isRadnik, onRefresh }: {
  employeeId: string; record?: PensionRecord; isRadnik: boolean; onRefresh: () => void
}) {
  const [file, setFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [verifying, setVerifying] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!isRadnik) return null

  async function handleUpload() {
    if (!file) return
    setUploading(true); setError(null)
    try {
      const fileId = await uploadDocFile(employeeId, file)
      await apiClient.put(`/documentation/employees/${employeeId}/pension`, { document_id: fileId })
      setFile(null); onRefresh()
    } catch (e) { setError(errMsg(e)) }
    finally { setUploading(false) }
  }

  async function handleVerify() {
    setVerifying(true); setError(null)
    try {
      await apiClient.post(`/documentation/employees/${employeeId}/pension/verify`)
      onRefresh()
    } catch (e) { setError(errMsg(e)) }
    finally { setVerifying(false) }
  }

  return (
    <SectionCard title="Mirovinsko osiguranje">
      {record
        ? <div className="flex items-center gap-2 mb-2">
            {statusBadge(record.status)}
            {record.verified_at && <span className="text-xs text-zinc-400">verificirano {formatDate(record.verified_at)}</span>}
          </div>
        : <p className="text-sm text-zinc-400 italic mb-2">Nema evidencije</p>}
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <FileInput file={file} onChange={setFile} />
          {file && (
            <button onClick={handleUpload} disabled={uploading} className={btnPrimary}>
              {uploading ? 'Učitava...' : 'Učitaj'}
            </button>
          )}
        </div>
        {record && record.status !== 'verified' && (
          <button onClick={handleVerify} disabled={verifying} className={btnPrimary}>
            {verifying ? 'Verificira...' : 'Verificiraj'}
          </button>
        )}
        {error && <p className="text-xs text-red-400">{error}</p>}
      </div>
    </SectionCard>
  )
}

// ── Health ────────────────────────────────────────────────────────────────────

function HealthSection({ employeeId, record, isRadnik, onRefresh }: {
  employeeId: string; record?: HealthRecord; isRadnik: boolean; onRefresh: () => void
}) {
  const [file, setFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [verifying, setVerifying] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!isRadnik) return null

  async function handleUpload() {
    if (!file) return
    setUploading(true); setError(null)
    try {
      const fileId = await uploadDocFile(employeeId, file)
      await apiClient.put(`/documentation/employees/${employeeId}/health`, { document_id: fileId })
      setFile(null); onRefresh()
    } catch (e) { setError(errMsg(e)) }
    finally { setUploading(false) }
  }

  async function handleVerify() {
    setVerifying(true); setError(null)
    try {
      await apiClient.post(`/documentation/employees/${employeeId}/health/verify`)
      onRefresh()
    } catch (e) { setError(errMsg(e)) }
    finally { setVerifying(false) }
  }

  return (
    <SectionCard title="Zdravstveno osiguranje">
      {record
        ? <div className="flex items-center gap-2 mb-2">
            {statusBadge(record.status)}
            {record.verified_at && <span className="text-xs text-zinc-400">verificirano {formatDate(record.verified_at)}</span>}
          </div>
        : <p className="text-sm text-zinc-400 italic mb-2">Nema evidencije</p>}
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <FileInput file={file} onChange={setFile} />
          {file && (
            <button onClick={handleUpload} disabled={uploading} className={btnPrimary}>
              {uploading ? 'Učitava...' : 'Učitaj'}
            </button>
          )}
        </div>
        {record && record.status !== 'verified' && (
          <button onClick={handleVerify} disabled={verifying} className={btnPrimary}>
            {verifying ? 'Verificira...' : 'Verificiraj'}
          </button>
        )}
        {error && <p className="text-xs text-red-400">{error}</p>}
      </div>
    </SectionCard>
  )
}

// ── Annual leave ──────────────────────────────────────────────────────────────

function AnnualLeaveSection({ employeeId, onRefresh }: { employeeId: string; onRefresh: () => void }) {
  const [leaves, setLeaves] = useState<AnnualLeave[]>([])
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [formOpen, setFormOpen] = useState(false)
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const loadLeaves = useCallback(async () => {
    setLoading(true)
    try {
      const r = await apiClient.get(`/documentation/employees/${employeeId}/annual-leave`)
      setLeaves(r.data.leaves ?? [])
    } finally {
      setLoading(false)
    }
  }, [employeeId])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSubmitting(true); setError(null)
    try {
      let fileId: string | undefined
      if (file) fileId = await uploadDocFile(employeeId, file)
      await apiClient.post(`/documentation/employees/${employeeId}/annual-leave`, {
        ...(dateFrom ? { date_from: dateFrom } : {}),
        ...(dateTo ? { date_to: dateTo } : {}),
        ...(fileId ? { document_id: fileId } : {}),
      })
      setFormOpen(false); setDateFrom(''); setDateTo(''); setFile(null)
      onRefresh(); loadLeaves()
    } catch (e) { setError(errMsg(e)) }
    finally { setSubmitting(false) }
  }

  return (
    <SectionCard title="Godišnji odmor">
      <div className="flex flex-wrap items-center gap-2 mb-2">
        <button onClick={() => setFormOpen(v => !v)} className={btnPrimary}>
          {formOpen ? 'Zatvori' : 'Dodaj godišnji'}
        </button>
        <button
          onClick={() => { if (!open) loadLeaves(); setOpen(v => !v) }}
          className="text-xs text-violet-400 hover:underline"
        >
          {open ? 'Sakrij zapise' : 'Prikaži zapise'}
        </button>
      </div>
      {formOpen && (
        <form onSubmit={handleSubmit} className="space-y-2 border-t border-zinc-700 pt-3">
          <div>
            <label className="text-xs text-zinc-400 mb-1 block">Od (neobavezno)</label>
            <input type="date" value={dateFrom} onChange={e => setDateFrom(e.target.value)} className={inputCls} />
          </div>
          <div>
            <label className="text-xs text-zinc-400 mb-1 block">Do (neobavezno)</label>
            <input type="date" value={dateTo} onChange={e => setDateTo(e.target.value)} className={inputCls} />
          </div>
          <div>
            <label className="text-xs text-zinc-400 mb-1 block">Dokument (neobavezno)</label>
            <FileInput file={file} onChange={setFile} />
          </div>
          {error && <p className="text-xs text-red-400">{error}</p>}
          <div className="flex gap-2">
            <button type="submit" disabled={submitting} className={btnPrimary}>{submitting ? 'Sprema...' : 'Spremi'}</button>
            <button type="button" onClick={() => setFormOpen(false)} className={btnSecondary}>Odustani</button>
          </div>
        </form>
      )}
      {open && (
        <div className="mt-2 space-y-1 max-h-40 overflow-y-auto">
          {loading && <p className="text-xs text-zinc-500">Učitavanje...</p>}
          {leaves.map(l => (
            <div key={l.id} className="text-xs text-zinc-300">
              {l.date_from ? formatDate(l.date_from) : '—'}
              {l.date_to ? ` → ${formatDate(l.date_to)}` : ''}
            </div>
          ))}
          {!loading && leaves.length === 0 && <p className="text-xs text-zinc-500">Nema evidencije</p>}
        </div>
      )}
    </SectionCard>
  )
}

// ── Employee detail panel ─────────────────────────────────────────────────────

function EmployeeDetailPanel({ employeeId, onClose }: { employeeId: string; onClose: () => void }) {
  const [summary, setSummary] = useState<EmployeeDocSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    try {
      const r = await apiClient.get(`/documentation/employees/${employeeId}/summary`)
      setSummary(r.data.summary)
    } catch { /* background refresh, ignore */ }
  }, [employeeId])

  useEffect(() => {
    setLoading(true); setError(null)
    refresh().finally(() => setLoading(false))
  }, [refresh])

  if (loading) return <div className="flex items-center justify-center h-32"><p className="text-zinc-400 text-sm">Učitavanje...</p></div>
  if (error || !summary) return <p className="text-red-400 text-sm p-4">{error ?? 'Nema podataka'}</p>

  const { employee } = summary
  const isRadnik = employee.role === 'radnik'

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold text-white">{employee.first_name} {employee.last_name}</h3>
          <p className="text-sm text-zinc-400 capitalize">{employee.role}</p>
        </div>
        <button onClick={onClose} className="text-zinc-500 hover:text-zinc-300 text-sm">✕ Zatvori</button>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        <MedicalSection employeeId={employeeId} initial={summary.latest_medical} onRefresh={refresh} />
        <SafetySection employeeId={employeeId} record={summary.safety_record} isRadnik={isRadnik} onRefresh={refresh} />
        <ContractSection employeeId={employeeId} initial={summary.active_contract} onRefresh={refresh} />
        <PermitSection employeeId={employeeId} initial={summary.latest_permit} onRefresh={refresh} />
        <ObligationsSection employeeId={employeeId} isRadnik={isRadnik} onRefresh={refresh} />
        <PensionSection employeeId={employeeId} record={summary.pension_record} isRadnik={isRadnik} onRefresh={refresh} />
        <HealthSection employeeId={employeeId} record={summary.health_record} isRadnik={isRadnik} onRefresh={refresh} />
        <AnnualLeaveSection employeeId={employeeId} onRefresh={refresh} />
      </div>
    </div>
  )
}

// ── Main page ─────────────────────────────────────────────────────────────────

const OVERDUE_DISPLAY_LIMIT = 6

export default function DokumentacijaPage() {
  const { user, employee, isLoading: authLoading, logout } = useAuth()
  const router = useRouter()

  const [tab, setTab] = useState<Tab>('pregled')
  const [employees, setEmployees] = useState<DocEmployee[]>([])
  const [notifications, setNotifications] = useState<DocNotification[]>([])
  const [search, setSearch] = useState('')
  const [selectedEmployeeId, setSelectedEmployeeId] = useState<string | null>(null)
  const [dataLoading, setDataLoading] = useState(true)
  const [showAllOverdue, setShowAllOverdue] = useState(false)

  useEffect(() => {
    if (!authLoading && user && user.role !== 'administracija') {
      router.replace('/dashboard')
    }
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
    } catch { /* silently fail */ }
    finally { setDataLoading(false) }
  }, [])

  useEffect(() => {
    if (user?.role === 'administracija') loadData()
  }, [user, loadData])

  if (authLoading) return <LoadingScreen />
  if (!user || user.role !== 'administracija') return null

  const filteredEmployees = employees.filter(e =>
    `${e.first_name} ${e.last_name}`.toLowerCase().includes(search.toLowerCase())
  )

  const urgentCount = notifications.filter(n => n.priority === 'urgent' || n.priority === 'overdue').length

  return (
    <DashboardShell title="Dokumentacija" user={user} employee={employee} onLogout={logout} backHref="/dashboard">
      {/* Tabs */}
      <div className="flex gap-1 bg-zinc-800/50 rounded-xl p-1 w-fit mb-6">
        {TABS.map(t => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
              tab === t.id ? 'bg-violet-600 text-white' : 'text-zinc-400 hover:text-zinc-200'
            }`}
          >
            {t.label}
            {t.id === 'obavijesti' && urgentCount > 0 && (
              <span className="ml-1.5 bg-red-600 text-white text-xs rounded-full px-1.5 py-0.5">{urgentCount}</span>
            )}
          </button>
        ))}
      </div>

      {dataLoading ? (
        <div className="flex items-center justify-center h-48">
          <p className="text-zinc-400">Učitavanje...</p>
        </div>
      ) : tab === 'pregled' ? (
        <div className="space-y-4">
          <input
            type="text"
            placeholder="Pretraži zaposlenike..."
            value={search}
            onChange={e => setSearch(e.target.value)}
            className="w-full max-w-sm px-3 py-2 rounded-lg bg-zinc-800 border border-zinc-700 text-sm text-zinc-200 placeholder-zinc-500 focus:outline-none focus:border-violet-500"
          />
          {selectedEmployeeId ? (
            <div className="rounded-xl border border-zinc-700/50 bg-zinc-900/50 p-4">
              <EmployeeDetailPanel
                employeeId={selectedEmployeeId}
                onClose={() => setSelectedEmployeeId(null)}
              />
            </div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
              {filteredEmployees.map(emp => {
                const empNotifs = notifications.filter(n => n.employee_id === emp.id)
                const urgent = empNotifs.filter(n => n.priority === 'urgent' || n.priority === 'overdue').length
                const warn = empNotifs.filter(n => n.priority === 'warning').length
                return (
                  <button
                    key={emp.id}
                    onClick={() => setSelectedEmployeeId(emp.id)}
                    className="text-left rounded-xl border border-zinc-700/50 bg-zinc-800/50 p-4 hover:border-violet-500/50 hover:bg-zinc-800 transition-colors"
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div>
                        <p className="font-medium text-zinc-100">{emp.first_name} {emp.last_name}</p>
                        <p className="text-xs text-zinc-400 capitalize mt-0.5">{emp.role}</p>
                      </div>
                      <div className="flex flex-col items-end gap-1 shrink-0">
                        {urgent > 0 && (
                          <span className="text-xs bg-red-900/60 text-red-400 px-1.5 py-0.5 rounded-full">{urgent} hitno</span>
                        )}
                        {warn > 0 && (
                          <span className="text-xs bg-amber-900/60 text-amber-300 px-1.5 py-0.5 rounded-full">{warn} upoz.</span>
                        )}
                      </div>
                    </div>
                  </button>
                )
              })}
              {filteredEmployees.length === 0 && (
                <p className="text-sm text-zinc-400 col-span-full">Nema rezultata</p>
              )}
            </div>
          )}
        </div>
      ) : (
        /* Obavijesti tab */
        <div className="space-y-4">
          {notifications.length === 0 && (
            <p className="text-zinc-400 text-sm">Nema aktivnih obavijesti.</p>
          )}
          {(['overdue', 'urgent', 'warning', 'low'] as const).map(priority => {
            const group = notifications.filter(n => n.priority === priority)
            if (group.length === 0) return null
            const labels: Record<string, string> = {
              overdue: 'Prekoračeno', urgent: 'Hitno', warning: 'Upozorenje', low: 'Nisko',
            }
            const displayed = priority === 'overdue' && !showAllOverdue
              ? group.slice(0, OVERDUE_DISPLAY_LIMIT)
              : group
            const hidden = priority === 'overdue' ? group.length - displayed.length : 0
            return (
              <div key={priority}>
                <h4 className="text-xs font-semibold text-zinc-500 uppercase tracking-wider mb-2">
                  {labels[priority]} ({group.length})
                </h4>
                <div className="space-y-2">
                  {displayed.map((n, i) => (
                    <div
                      key={`${n.type}-${n.employee_id}-${i}`}
                      className="flex items-start gap-3 rounded-xl border border-zinc-700/50 bg-zinc-800/50 p-3"
                    >
                      <div className="flex-1 min-w-0">
                        <p className="text-sm text-zinc-100">{n.message}</p>
                        <p className="text-xs text-zinc-400 mt-0.5">{n.employee_name}</p>
                      </div>
                      <div className="shrink-0">{priorityBadge(n.priority)}</div>
                    </div>
                  ))}
                  {hidden > 0 && (
                    <button
                      onClick={() => setShowAllOverdue(true)}
                      className="text-xs text-violet-400 hover:underline"
                    >
                      Prikaži {hidden} više prekoračenih...
                    </button>
                  )}
                  {showAllOverdue && priority === 'overdue' && group.length > OVERDUE_DISPLAY_LIMIT && (
                    <button
                      onClick={() => setShowAllOverdue(false)}
                      className="text-xs text-zinc-500 hover:underline"
                    >
                      Prikaži manje
                    </button>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      )}
    </DashboardShell>
  )
}
