'use client'

import { useState, useEffect, useCallback, useMemo } from 'react'
import { ChevronLeft, ChevronRight, Plus, X } from 'lucide-react'
import { useAuth } from '@/hooks/useAuth'
import DashboardShell from '@/components/layout/DashboardShell'
import LoadingScreen from '@/components/ui/LoadingScreen'
import apiClient from '@/lib/api-client'
import { type Shift, type ShiftConflict, type EmployeeForDate } from '@/lib/types/schedule'
import { type ProjectListItem } from '@/lib/types/projects'
import { type Employee, ROLE_LABELS } from '@/lib/types/employees'
import EmployeePicker from './_components/EmployeePicker'
import ScheduleGrid from './_components/ScheduleGrid'
import MobileAgenda from './_components/MobileAgenda'
import ShiftDetailPanel from './_components/ShiftDetailPanel'
import { type EmpRow } from './_components/schedule-utils'

// ── Date helpers ─────────────────────────────────────────────────────────

function getMonday(d: Date): Date {
  const day = d.getDay()
  return new Date(d.getFullYear(), d.getMonth(), d.getDate() - day + (day === 0 ? -6 : 1))
}

function addDays(d: Date, n: number): Date {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate() + n)
}

function toDateStr(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

const MONTH_GEN = [
  'siječnja','veljače','ožujka','travnja','svibnja','lipnja',
  'srpnja','kolovoza','rujna','listopada','studenoga','prosinca',
]

function weekLabel(start: Date): string {
  const end = addDays(start, 6)
  if (start.getMonth() === end.getMonth()) {
    return `${start.getDate()}. – ${end.getDate()}. ${MONTH_GEN[end.getMonth()]} ${end.getFullYear()}.`
  }
  return `${start.getDate()}. ${MONTH_GEN[start.getMonth()]} – ${end.getDate()}. ${MONTH_GEN[end.getMonth()]} ${end.getFullYear()}.`
}

// ── Role ordering for matrix rows ─────────────────────────────────────────

const ROLE_ORDER: Record<string, number> = { poslovoda: 1, radnik: 2, inzenjer: 3, direktor: 4, administracija: 5 }

// ── Types ─────────────────────────────────────────────────────────────────

interface ShiftForm {
  project_id: string
  shift_date: string
  notes: string
}

const DEFAULT_FORM: ShiftForm = {
  project_id: '', shift_date: '', notes: '',
}

// ── Main page ─────────────────────────────────────────────────────────────

export default function SchedulePage() {
  const { user, employee, isLoading, logout } = useAuth()
  const canManage = user?.role === 'direktor' || user?.role === 'inzenjer'

  const [weekStart, setWeekStart] = useState<Date>(() => getMonday(new Date()))

  const [shifts, setShifts] = useState<Shift[]>([])
  const [allEmployees, setAllEmployees] = useState<Employee[]>([])
  const [projects, setProjects] = useState<ProjectListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [projectFilter, setProjectFilter] = useState('')
  const [employeeFilter, setEmployeeFilter] = useState('')
  const [roleFilter, setRoleFilter] = useState('')
  const [myShiftsOnly, setMyShiftsOnly] = useState(false)
  const [showCancelled, setShowCancelled] = useState(false)

  // Create / edit form
  const [showForm, setShowForm] = useState(false)
  const [editId, setEditId] = useState<string | null>(null)
  const [form, setForm] = useState<ShiftForm>(DEFAULT_FORM)
  const [formEmployees, setFormEmployees] = useState<Set<string>>(new Set())
  const [formError, setFormError] = useState<string | null>(null)
  const [formConflicts, setFormConflicts] = useState<ShiftConflict[]>([])
  const [saving, setSaving] = useState(false)
  // Availability data for the create-form employee picker
  const [formAvailability, setFormAvailability] = useState<Map<string, EmployeeForDate>>(new Map())
  const [formAvailLoading, setFormAvailLoading] = useState(false)

  const [selectedShift, setSelectedShift] = useState<Shift | null>(null)
  const [cancellingId, setCancellingId] = useState<string | null>(null)

  // Assignment modal
  const [assignShift, setAssignShift] = useState<Shift | null>(null)
  const [selEmployees, setSelEmployees] = useState<Set<string>>(new Set())
  const [assignConflicts, setAssignConflicts] = useState<ShiftConflict[]>([])
  const [assigning, setAssigning] = useState(false)
  const [assignAvailability, setAssignAvailability] = useState<Map<string, EmployeeForDate>>(new Map())

  const [mobileDay, setMobileDay] = useState<string>(() => toDateStr(new Date()))

  const weekDays = useMemo(() =>
    Array.from({ length: 7 }, (_, i) => toDateStr(addDays(weekStart, i))),
    [weekStart],
  )
  const today = toDateStr(new Date())
  const dateFrom = weekDays[0]
  const dateTo = weekDays[6]

  // ── Data fetching ─────────────────────────────────────────────────────────

  const fetchShifts = useCallback(async () => {
    if (!user) return
    setLoading(true)
    setError(null)
    try {
      const params = new URLSearchParams({ date_from: dateFrom, date_to: dateTo })
      if (projectFilter) params.set('project_id', projectFilter)
      const res = await apiClient.get(`/schedule/shifts?${params}`)
      setShifts(res.data.shifts ?? [])
    } catch {
      setError('Greška pri učitavanju rasporeda.')
    } finally {
      setLoading(false)
    }
  }, [user, dateFrom, dateTo, projectFilter])

  useEffect(() => { if (!isLoading && user) fetchShifts() }, [isLoading, user, fetchShifts])

  useEffect(() => {
    if (!user) return
    apiClient.get('/projects?status=active').then(r => setProjects(r.data.projects ?? [])).catch(() => {})
    apiClient.get('/employees?active=true').then(r => setAllEmployees(r.data.employees ?? [])).catch(() => {})
  }, [user])

  useEffect(() => {
    if (!weekDays.includes(mobileDay)) {
      setMobileDay(weekDays.includes(today) ? today : weekDays[0])
    }
  }, [weekDays, mobileDay, today])

  useEffect(() => {
    if (!selectedShift) return
    const fresh = shifts.find(s => s.id === selectedShift.id)
    setSelectedShift(fresh ?? null)
  }, [shifts]) // eslint-disable-line react-hooks/exhaustive-deps

  // Load availability when the create modal opens or the date changes.
  // showForm in deps ensures we re-fetch after the modal was closed (e.g. after
  // a successful shift creation that made one employee unavailable).
  useEffect(() => {
    if (!showForm || !form.shift_date || editId) {
      setFormAvailability(new Map())
      return
    }
    setFormAvailLoading(true)
    apiClient.get(`/schedule/employees-for-date?date=${form.shift_date}`)
      .then(r => {
        const map = new Map<string, EmployeeForDate>()
        for (const e of (r.data.employees ?? []) as EmployeeForDate[]) map.set(e.id, e)
        setFormAvailability(map)
      })
      .catch(() => setFormAvailability(new Map()))
      .finally(() => setFormAvailLoading(false))
  }, [form.shift_date, editId, showForm])

  // Load availability when assignment modal opens
  useEffect(() => {
    if (!assignShift) { setAssignAvailability(new Map()); return }
    apiClient.get(`/schedule/employees-for-date?date=${assignShift.shift_date}&exclude_shift_id=${assignShift.id}`)
      .then(r => {
        const map = new Map<string, EmployeeForDate>()
        for (const e of (r.data.employees ?? []) as EmployeeForDate[]) map.set(e.id, e)
        setAssignAvailability(map)
      })
      .catch(() => setAssignAvailability(new Map()))
  }, [assignShift])

  // ── Derived data ──────────────────────────────────────────────────────────

  const projectById = useMemo(() => new Map(projects.map(p => [p.id, p])), [projects])

  const empRows = useMemo<EmpRow[]>(() => {
    return [...allEmployees]
      .sort((a, b) => {
        const ao = ROLE_ORDER[a.role] ?? 9
        const bo = ROLE_ORDER[b.role] ?? 9
        if (ao !== bo) return ao - bo
        return `${a.last_name} ${a.first_name}`.localeCompare(`${b.last_name} ${b.first_name}`, 'hr')
      })
      .map(e => ({
        id: e.id,
        name: `${e.first_name} ${e.last_name}`,
        role: ROLE_LABELS[e.role] ?? e.role,
        rawRole: e.role,
      }))
  }, [allEmployees])

  const filteredRows = useMemo(() => {
    let rows = empRows
    if (myShiftsOnly && employee) rows = rows.filter(r => r.id === employee.id)
    if (employeeFilter) rows = rows.filter(r => r.id === employeeFilter)
    if (roleFilter) rows = rows.filter(r => r.rawRole === roleFilter)
    return rows
  }, [empRows, myShiftsOnly, employee, employeeFilter, roleFilter])

  const visibleShifts = useMemo(
    () => shifts.filter(s => showCancelled || s.status !== 'cancelled'),
    [shifts, showCancelled],
  )

  const shiftsByEmpDate = useMemo(() => {
    const map = new Map<string, Map<string, Shift[]>>()
    for (const shift of visibleShifts) {
      for (const a of shift.assignments) {
        if (!map.has(a.employee_id)) map.set(a.employee_id, new Map())
        const dm = map.get(a.employee_id)!
        if (!dm.has(shift.shift_date)) dm.set(shift.shift_date, [])
        const bucket = dm.get(shift.shift_date)!
        if (!bucket.includes(shift)) bucket.push(shift)
      }
    }
    return map
  }, [visibleShifts])

  const shiftsByDate = useMemo(() => {
    const map = new Map<string, Shift[]>()
    weekDays.forEach(d => map.set(d, []))
    const seen = new Set<string>()
    for (const s of visibleShifts) {
      if (seen.has(s.id)) continue
      seen.add(s.id)
      const bucket = map.get(s.shift_date)
      if (bucket) bucket.push(s)
    }
    return map
  }, [visibleShifts, weekDays])

  // ── Navigation ────────────────────────────────────────────────────────────

  function prevWeek() { setWeekStart(w => addDays(w, -7)) }
  function nextWeek() { setWeekStart(w => addDays(w, 7)) }
  function goToday() { setWeekStart(getMonday(new Date())); setMobileDay(toDateStr(new Date())) }

  // ── Form ──────────────────────────────────────────────────────────────────

  function openCreate(date?: string) {
    setEditId(null)
    setForm({ ...DEFAULT_FORM, shift_date: date ?? dateFrom })
    setFormEmployees(new Set())
    setFormError(null)
    setFormConflicts([])
    setShowForm(true)
    setSelectedShift(null)
  }

  function openEdit(shift: Shift) {
    setEditId(shift.id)
    setForm({
      project_id: shift.project_id,
      shift_date: shift.shift_date,
      notes: shift.notes ?? '',
    })
    setFormEmployees(new Set())
    setFormError(null)
    setFormConflicts([])
    setShowForm(true)
  }

  async function submitCreate() {
    setSaving(true)
    setFormError(null)
    setFormConflicts([])
    try {
      await apiClient.post('/schedule/shifts', {
        project_id: form.project_id,
        shift_date: form.shift_date,
        notes: form.notes || null,
        employee_ids: [...formEmployees],
        override_overlaps: false,
      })
      setShowForm(false)
      fetchShifts()
    } catch (err: unknown) {
      const e = err as { response?: { status?: number; data?: { error?: string; conflicts?: ShiftConflict[] } } }
      if (e?.response?.status === 409) {
        const conflicts = e?.response?.data?.conflicts
        if (conflicts && conflicts.length > 0) {
          setFormConflicts(conflicts)
        } else {
          setFormError(e?.response?.data?.error ?? 'Greška: konflikt pri kreiranju smjene.')
        }
      } else {
        setFormError(e?.response?.data?.error ?? 'Greška pri kreiranju smjene.')
      }
    } finally {
      setSaving(false)
    }
  }

  async function submitEdit() {
    setSaving(true)
    setFormError(null)
    try {
      await apiClient.patch(`/schedule/shifts/${editId}`, {
        shift_date: form.shift_date,
        notes: form.notes || null,
      })
      setShowForm(false)
      fetchShifts()
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setFormError(e?.response?.data?.error ?? 'Greška pri ažuriranju smjene.')
    } finally {
      setSaving(false)
    }
  }

  async function handleFormSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!editId && formEmployees.size === 0) {
      setFormError('Odaberite barem jednog zaposlenika.')
      return
    }
    if (editId) await submitEdit()
    else await submitCreate()
  }

  // ── Cancel & Duplicate ────────────────────────────────────────────────────

  async function handleCancel(shift: Shift) {
    if (!confirm(`Otkazati smjenu na projektu "${shift.project_name}" za datum ${shift.shift_date}?`)) return
    setCancellingId(shift.id)
    try {
      await apiClient.post(`/schedule/shifts/${shift.id}/cancel`)
      setSelectedShift(null)
      fetchShifts()
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      alert(e?.response?.data?.error ?? 'Greška pri otkazivanju smjene.')
    } finally {
      setCancellingId(null)
    }
  }

  async function handleDuplicate(shift: Shift) {
    const target = prompt('Unesite ciljni datum (GGGG-MM-DD):', shift.shift_date)
    if (!target) return
    try {
      await apiClient.post(`/schedule/shifts/${shift.id}/duplicate`, { target_date: target })
      fetchShifts()
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      alert(e?.response?.data?.error ?? 'Greška pri dupliciranju.')
    }
  }

  // ── Assignment modal ──────────────────────────────────────────────────────

  function openAssign(shift: Shift) {
    setAssignShift(shift)
    setSelEmployees(new Set(shift.assignments.map(a => a.employee_id)))
    setAssignConflicts([])
  }

  async function handleSaveAssignments() {
    if (!assignShift) return
    setAssigning(true)
    try {
      await apiClient.put(`/schedule/shifts/${assignShift.id}/assignments`, {
        employee_ids: [...selEmployees],
        override_overlaps: false,
      })
      setAssignShift(null)
      setAssignConflicts([])
      fetchShifts()
    } catch (err: unknown) {
      const e = err as { response?: { status?: number; data?: { conflicts?: ShiftConflict[]; error?: string } } }
      if (e?.response?.status === 409) {
        const conflicts = e?.response?.data?.conflicts
        if (conflicts && conflicts.length > 0) {
          setAssignConflicts(conflicts)
        } else {
          alert(e?.response?.data?.error ?? 'Greška: konflikt pri raspoređivanju.')
        }
      } else {
        alert(e?.response?.data?.error ?? 'Greška pri raspoređivanju.')
      }
    } finally {
      setAssigning(false)
    }
  }

  // ── Render ────────────────────────────────────────────────────────────────

  if (isLoading) return <LoadingScreen />
  if (!user) return null

  const hasActiveFilters = !!(projectFilter || employeeFilter || roleFilter || myShiftsOnly)

  return (
    <DashboardShell user={user} employee={employee} title="Raspored radova" backHref="/dashboard" onLogout={logout} wide>

      {/* ── Page header ── */}
      <div className="flex flex-col gap-3 mb-5">
        <div className="flex flex-col sm:flex-row sm:items-center gap-3">
          <div>
            <h1 className="text-xl font-bold text-white">Raspored radova</h1>
            <p className="text-xs text-slate-500">Tjedni pregled</p>
          </div>

          <div className="flex items-center gap-1 sm:ml-5">
            <button onClick={prevWeek} aria-label="Prethodni tjedan"
              className="p-2 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition">
              <ChevronLeft className="w-4 h-4" />
            </button>
            <button onClick={goToday}
              className="px-3 py-1.5 rounded-lg text-sm font-medium text-slate-200 hover:bg-slate-800 transition min-w-[200px] text-center">
              {weekLabel(weekStart)}
            </button>
            <button onClick={nextWeek} aria-label="Sljedeći tjedan"
              className="p-2 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition">
              <ChevronRight className="w-4 h-4" />
            </button>
          </div>

          {canManage && (
            <button onClick={() => openCreate()}
              className="sm:ml-auto flex items-center gap-1.5 px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-sm font-medium transition">
              <Plus className="w-4 h-4" />Novi raspored
            </button>
          )}
        </div>

        {/* Filter bar */}
        <div className="flex flex-wrap items-center gap-2">
          <select value={projectFilter} onChange={e => setProjectFilter(e.target.value)}
            className="px-3 py-1.5 rounded-lg text-xs bg-slate-800 border border-slate-700 text-slate-300 focus:outline-none focus:border-blue-500">
            <option value="">Svi projekti</option>
            {projects.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
          </select>

          <select value={employeeFilter} onChange={e => setEmployeeFilter(e.target.value)}
            className="px-3 py-1.5 rounded-lg text-xs bg-slate-800 border border-slate-700 text-slate-300 focus:outline-none focus:border-blue-500">
            <option value="">Svi radnici</option>
            {empRows.map(r => <option key={r.id} value={r.id}>{r.name}</option>)}
          </select>

          <select value={roleFilter} onChange={e => setRoleFilter(e.target.value)}
            className="px-3 py-1.5 rounded-lg text-xs bg-slate-800 border border-slate-700 text-slate-300 focus:outline-none focus:border-blue-500">
            <option value="">Sve uloge</option>
            {Object.entries(ROLE_LABELS).map(([k, v]) => <option key={k} value={k}>{v}</option>)}
          </select>

          {employee && (
            <label className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs bg-slate-800 border border-slate-700 text-slate-300 cursor-pointer select-none">
              <input type="checkbox" checked={myShiftsOnly} onChange={e => setMyShiftsOnly(e.target.checked)} className="w-3 h-3" />
              Samo moje smjene
            </label>
          )}

          <label className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs bg-slate-800 border border-slate-700 text-slate-300 cursor-pointer select-none">
            <input type="checkbox" checked={showCancelled} onChange={e => setShowCancelled(e.target.checked)} className="w-3 h-3" />
            Prikaži otkazane
          </label>

          {hasActiveFilters && (
            <button onClick={() => { setProjectFilter(''); setEmployeeFilter(''); setRoleFilter(''); setMyShiftsOnly(false) }}
              className="px-3 py-1.5 rounded-lg text-xs text-slate-400 hover:text-white hover:bg-slate-700 transition">
              Poništi filtere
            </button>
          )}
        </div>
      </div>

      {/* ── Loading / error ── */}
      {loading ? (
        <div className="flex items-center justify-center h-64 border border-slate-800 rounded-xl">
          <p className="text-slate-500 text-sm animate-pulse">Učitavanje rasporeda...</p>
        </div>
      ) : error ? (
        <div className="flex items-center justify-center h-64 border border-red-900/30 rounded-xl">
          <p className="text-red-400 text-sm">{error}</p>
        </div>
      ) : (
        <>
          <div className="hidden sm:block">
            <ScheduleGrid
              weekDays={weekDays}
              today={today}
              rows={filteredRows}
              shiftsByEmpDate={shiftsByEmpDate}
              employee={employee}
              canManage={canManage}
              selectedShift={selectedShift}
              projectById={projectById}
              onSelectShift={setSelectedShift}
              onOpenCreate={openCreate}
            />
          </div>

          <div className="sm:hidden">
            <MobileAgenda
              weekDays={weekDays}
              today={today}
              mobileDay={mobileDay}
              onSetMobileDay={setMobileDay}
              shiftsByDate={shiftsByDate}
              employee={employee}
              canManage={canManage}
              selectedShift={selectedShift}
              projectById={projectById}
              onSelectShift={setSelectedShift}
              onOpenCreate={date => openCreate(date)}
            />
          </div>

          {selectedShift && (
            <ShiftDetailPanel
              shift={selectedShift}
              employee={employee}
              canManage={canManage}
              cancellingId={cancellingId}
              projectAddress={projectById.get(selectedShift.project_id)?.address ?? null}
              onEdit={() => openEdit(selectedShift)}
              onCancel={() => handleCancel(selectedShift)}
              onDuplicate={() => handleDuplicate(selectedShift)}
              onAssign={() => openAssign(selectedShift)}
              onClose={() => setSelectedShift(null)}
            />
          )}

          <div className="mt-4 flex flex-wrap items-center gap-4 text-xs text-slate-500">
            <span className="font-medium text-slate-400">Legenda:</span>
            <span className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-full bg-blue-500 inline-block" />Aktivna smjena</span>
            <span className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-full bg-slate-600 inline-block" />Otkazana smjena</span>
            <span className="flex items-center gap-1.5"><span className="w-2 h-3 rounded-sm bg-blue-500/30 border-l-2 border-blue-500 inline-block" />Moja smjena</span>
          </div>
        </>
      )}

      {/* ── Create / Edit form modal ── */}
      {showForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70">
          <div className="bg-slate-900 border border-slate-700 rounded-2xl w-full max-w-lg shadow-2xl max-h-[90vh] flex flex-col">
            <div className="flex items-center justify-between p-4 border-b border-slate-800 flex-shrink-0">
              <h3 className="text-sm font-semibold text-white">{editId ? 'Uredi smjenu' : 'Nova smjena'}</h3>
              <button type="button" onClick={() => setShowForm(false)} className="text-slate-500 hover:text-white transition p-1">
                <X className="w-4 h-4" />
              </button>
            </div>

            <form onSubmit={handleFormSubmit} className="flex flex-col overflow-hidden flex-1">
              <div className="overflow-y-auto flex-1 p-4 space-y-4">
                {/* Project (create only) */}
                {!editId && (
                  <div>
                    <label className="block text-xs text-slate-400 mb-1">Projekt *</label>
                    <select required value={form.project_id}
                      onChange={e => setForm(f => ({ ...f, project_id: e.target.value }))}
                      className="w-full px-3 py-2 rounded-lg bg-slate-800 border border-slate-700 text-white text-sm focus:outline-none focus:border-blue-500">
                      <option value="">Odaberite projekt...</option>
                      {projects.map(p => <option key={p.id} value={p.id}>{p.name}{p.address ? ` — ${p.address}` : ''}</option>)}
                    </select>
                  </div>
                )}

                {/* Date */}
                <div>
                  <label className="block text-xs text-slate-400 mb-1">Datum smjene *</label>
                  <input required type="date" value={form.shift_date}
                    onChange={e => setForm(f => ({ ...f, shift_date: e.target.value }))}
                    className="w-full px-3 py-2 rounded-lg bg-slate-800 border border-slate-700 text-white text-sm focus:outline-none focus:border-blue-500" />
                </div>

                {/* Notes */}
                <div>
                  <label className="block text-xs text-slate-400 mb-1">Napomene / Opis rada</label>
                  <textarea rows={2} value={form.notes}
                    onChange={e => setForm(f => ({ ...f, notes: e.target.value }))}
                    placeholder="Opcionalno..." className="w-full px-3 py-2 rounded-lg bg-slate-800 border border-slate-700 text-white text-sm focus:outline-none focus:border-blue-500 resize-none" />
                </div>

                {/* Employee picker (create only) */}
                {!editId && (
                  <div>
                    <div className="flex items-center justify-between mb-1.5">
                      <label className="text-xs text-slate-400">
                        Pretraži zaposlenike *{' '}
                        <span className="text-slate-500">({formEmployees.size} odabrano)</span>
                      </label>
                      {formAvailLoading && <span className="text-[10px] text-slate-500 animate-pulse">Učitavanje...</span>}
                    </div>
                    <EmployeePicker
                      employees={allEmployees}
                      availability={formAvailability}
                      selected={formEmployees}
                      onChange={setFormEmployees}
                    />
                  </div>
                )}

                {/* Conflict error (409 from server) */}
                {formConflicts.length > 0 && (
                  <div className="p-3 rounded-lg bg-red-900/30 border border-red-700/50 text-xs text-red-300 space-y-1">
                    <p className="font-semibold">Zaposlenici su već raspoređeni za taj datum:</p>
                    {formConflicts.map((c, i) => (
                      <p key={i}>{c.employee_name} — raspoređen/a na: {c.project_name}</p>
                    ))}
                    <p className="text-red-400/70 mt-1">Uklonite te zaposlenike iz odabira ili odaberite drugi datum.</p>
                  </div>
                )}

                {formError && <p className="text-sm text-red-400">{formError}</p>}
              </div>

              <div className="flex gap-2 p-4 border-t border-slate-800 flex-shrink-0">
                <button type="submit" disabled={saving}
                  className="flex-1 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white text-sm font-medium transition">
                  {saving ? 'Sprema...' : editId ? 'Spremi izmjene' : 'Kreiraj smjenu'}
                </button>
                <button type="button" onClick={() => setShowForm(false)}
                  className="px-4 py-2 rounded-lg text-slate-400 hover:text-white hover:bg-slate-700 text-sm transition">
                  Odustani
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* ── Assignment modal ── */}
      {assignShift && (
        <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-4 bg-black/70">
          <div className="bg-slate-900 border border-slate-700 rounded-2xl w-full max-w-md max-h-[85vh] flex flex-col shadow-2xl">
            <div className="flex items-start justify-between p-4 border-b border-slate-800">
              <div>
                <p className="text-sm font-semibold text-white">Uredi raspoređene radnike</p>
                <p className="text-xs text-slate-400 mt-0.5">
                  {assignShift.project_name} · {assignShift.shift_date}
                </p>
              </div>
              <button type="button" onClick={() => { setAssignShift(null); setAssignConflicts([]) }}
                className="text-slate-500 hover:text-white transition p-1">
                <X className="w-5 h-5" />
              </button>
            </div>

            {assignConflicts.length > 0 && (
              <div className="mx-4 mt-3 p-3 rounded-lg bg-red-900/30 border border-red-700/50 text-xs text-red-300 space-y-1">
                <p className="font-semibold">Zaposlenici su već raspoređeni za taj datum:</p>
                {assignConflicts.map((c, i) => (
                  <p key={i}>{c.employee_name} — raspoređen/a na: {c.project_name}</p>
                ))}
                <p className="text-red-400/70 mt-1">Uklonite te zaposlenike iz odabira.</p>
              </div>
            )}

            <div className="overflow-y-auto flex-1 p-3">
              <EmployeePicker
                employees={allEmployees}
                availability={assignAvailability}
                selected={selEmployees}
                onChange={setSelEmployees}
              />
            </div>

            <div className="p-4 border-t border-slate-800 flex gap-2">
              <button type="button" onClick={() => handleSaveAssignments()} disabled={assigning}
                className="flex-1 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white text-sm font-medium transition">
                {assigning ? 'Sprema...' : `Spremi (${selEmployees.size})`}
              </button>
              <button type="button" onClick={() => { setAssignShift(null); setAssignConflicts([]) }}
                className="px-4 py-2 rounded-lg text-slate-400 hover:text-white hover:bg-slate-700 text-sm transition">
                Odustani
              </button>
            </div>
          </div>
        </div>
      )}
    </DashboardShell>
  )
}
