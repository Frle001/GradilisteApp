'use client'

import { useState, useEffect, useCallback, useMemo } from 'react'
import { ChevronLeft, ChevronRight, Plus, X, Users, Pencil, Ban, Copy, CalendarDays, Search } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import type { MeEmployee } from '@/hooks/useAuth'
import AppShell from '@/components/layout/AppShell'
import LoadingScreen from '@/components/ui/LoadingScreen'
import apiClient from '@/lib/api-client'
import { type Shift, type EmployeeForDate } from '@/lib/types/schedule'
import { type ProjectListItem } from '@/lib/types/projects'
import { type Employee, ROLE_LABELS } from '@/lib/types/employees'

// ── Date helpers ──────────────────────────────────────────────────────────

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

function parseDateLocal(s: string): Date {
  const [y, m, d] = s.split('-').map(Number)
  return new Date(y, m - 1, d)
}

const DAY_SHORT = ['Pon', 'Uto', 'Sri', 'Čet', 'Pet', 'Sub', 'Ned']
const DAY_FULL = ['Ponedjeljak', 'Utorak', 'Srijeda', 'Četvrtak', 'Petak', 'Subota', 'Nedjelja']
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

// ── Project colors ────────────────────────────────────────────────────────

const PALETTE = ['#3b82f6','#10b981','#f59e0b','#8b5cf6','#ef4444','#06b6d4','#f97316','#ec4899']

function projectColor(id: string): string {
  let h = 0
  for (let i = 0; i < id.length; i++) h = (h * 31 + id.charCodeAt(i)) >>> 0
  return PALETTE[h % PALETTE.length]
}

function hexToRgb(hex: string) {
  return { r: parseInt(hex.slice(1,3),16), g: parseInt(hex.slice(3,5),16), b: parseInt(hex.slice(5,7),16) }
}

// ── Role ordering ─────────────────────────────────────────────────────────

const ROLE_ORDER: Record<string, number> = { poslovoda: 1, radnik: 2, inzenjer: 3, direktor: 4, administracija: 5 }

// ── Types ─────────────────────────────────────────────────────────────────

interface EmpRow { id: string; name: string; role: string; rawRole: string }

interface ShiftForm {
  project_id: string
  shift_date: string
  notes: string
}

type ViewTab = 'calendar' | 'matrix'

const DEFAULT_FORM: ShiftForm = { project_id: '', shift_date: '', notes: '' }

// ── Employee picker ───────────────────────────────────────────────────────

function EmployeePicker({
  employees, availability, selected, onChange,
}: {
  employees: Employee[]
  availability: Map<string, EmployeeForDate>
  selected: Set<string>
  onChange: (next: Set<string>) => void
}) {
  const [search, setSearch] = useState('')

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    if (!q) return employees
    return employees.filter(e => `${e.first_name} ${e.last_name}`.toLowerCase().includes(q))
  }, [employees, search])

  function toggle(id: string) {
    const next = new Set(selected)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    onChange(next)
  }

  return (
    <div>
      <div className="relative mb-2">
        <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gi-text-subtle" />
        <input
          type="text"
          placeholder="Pretraži zaposlenike..."
          value={search}
          onChange={e => setSearch(e.target.value)}
          className="w-full pl-8 pr-3 py-1.5 rounded-lg bg-gi-card-elevated border border-gi-border text-gi-text text-sm focus:outline-none focus:border-gi-yellow"
        />
      </div>
      <div className="rounded-lg border border-gi-border bg-gi-card-elevated overflow-y-auto max-h-52 divide-y divide-gi-border/50">
        {filtered.length === 0 && (
          <div className="px-3 py-4 text-xs text-gi-text-subtle text-center">Nema rezultata.</div>
        )}
        {filtered.map(emp => {
          const info = availability.get(emp.id)
          const isAssigned = !!info?.assigned
          const isChecked = selected.has(emp.id)
          return (
            <label key={emp.id} className="flex items-start gap-3 px-3 py-2 hover:bg-gi-card transition-colors cursor-pointer">
              <input
                type="checkbox"
                checked={isChecked}
                onChange={() => toggle(emp.id)}
                className="w-4 h-4 rounded bg-gi-card-elevated border-gi-border flex-shrink-0 mt-0.5"
              />
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="text-sm text-gi-text flex-1 truncate">{emp.first_name} {emp.last_name}</span>
                  <span className="text-[11px] text-gi-text-subtle flex-shrink-0">{ROLE_LABELS[emp.role] ?? emp.role}</span>
                  {isAssigned && (
                    <span className="flex-shrink-0 text-[10px] bg-gi-yellow/15 text-gi-yellow border border-gi-yellow/30 px-1.5 py-0.5 rounded-full font-medium">
                      Ima smjenu
                    </span>
                  )}
                </div>
                {isAssigned && info?.project_name && (
                  <p className="text-[11px] text-gi-text-subtle mt-0.5">{info.project_name}</p>
                )}
              </div>
            </label>
          )
        })}
      </div>
    </div>
  )
}

// ── Shift card ────────────────────────────────────────────────────────────

function ShiftCard({
  shift, isSelected, isMine, address, onClick,
}: {
  shift: Shift; isSelected: boolean; isMine: boolean; address: string | null; onClick: () => void
}) {
  const isCancelled = shift.status === 'cancelled'
  const color = projectColor(shift.project_id)
  const { r, g, b } = hexToRgb(isCancelled ? '#475569' : color)

  return (
    <button
      type="button"
      onClick={onClick}
      className={`w-full text-left rounded-md px-2 py-1 text-xs border-l-[3px] transition-all ${
        isCancelled ? 'opacity-50 cursor-default' : 'cursor-pointer hover:brightness-110'
      } ${isSelected ? 'ring-1 ring-inset ring-white/30' : ''}`}
      style={{
        borderLeftColor: isCancelled ? '#475569' : color,
        background: `rgba(${r},${g},${b},${isSelected ? 0.25 : 0.12})`,
      }}
    >
      <div className={`font-semibold truncate leading-tight ${isCancelled ? 'line-through text-gi-text-subtle' : 'text-gi-text'}`}>
        {shift.project_name}
      </div>
      {address && !isCancelled && (
        <div className="text-gi-text-muted truncate text-[10px]">{address}</div>
      )}
      {isMine && !isCancelled && (
        <div className="mt-0.5">
          <span className="rounded-full bg-gi-yellow/20 text-gi-yellow px-1 py-px text-[9px] font-medium">Moja</span>
        </div>
      )}
      {isCancelled && (
        <div className="mt-0.5">
          <span className="rounded-full bg-red-900/40 text-red-400 px-1 py-px text-[9px] font-medium">Otkazana</span>
        </div>
      )}
    </button>
  )
}

// ── Week calendar (primary desktop view) ──────────────────────────────────

function WeekCalendar({
  weekDays, today, shiftsByDate, employee, canManage,
  selectedShift, projectById, onSelectShift, onOpenCreate,
}: {
  weekDays: string[]
  today: string
  shiftsByDate: Map<string, Shift[]>
  employee: MeEmployee | null
  canManage: boolean
  selectedShift: Shift | null
  projectById: Map<string, ProjectListItem>
  onSelectShift: (s: Shift) => void
  onOpenCreate: (date: string) => void
}) {
  return (
    <div className="rounded-xl border border-gi-border overflow-x-auto">
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(7, minmax(130px, 1fr))', minWidth: 910 }}>
        {/* Day headers */}
        {weekDays.map((day, i) => {
          const dt = parseDateLocal(day)
          const isToday = day === today
          return (
            <div
              key={day}
              className={`px-3 py-3 border-b border-gi-border ${i > 0 ? 'border-l border-gi-border' : ''} ${isToday ? 'bg-gi-yellow/[0.07]' : 'bg-gi-card'}`}
            >
              <div className={`text-[10px] font-semibold uppercase tracking-wider mb-0.5 ${isToday ? 'text-gi-yellow' : 'text-gi-text-subtle'}`}>
                {DAY_SHORT[i]}
              </div>
              <div className={`text-xl font-bold leading-tight ${isToday ? 'text-gi-yellow' : 'text-gi-text'}`}>
                {dt.getDate()}
              </div>
              <div className="text-[10px] text-gi-text-subtle">
                {String(dt.getDate()).padStart(2,'0')}.{String(dt.getMonth()+1).padStart(2,'0')}.
              </div>
            </div>
          )
        })}
        {/* Shift cells */}
        {weekDays.map((day, i) => {
          const dayShifts = shiftsByDate.get(day) ?? []
          const isToday = day === today
          return (
            <div
              key={`cell-${day}`}
              className={`px-1.5 py-2 min-h-[160px] ${i > 0 ? 'border-l border-gi-border' : ''} ${isToday ? 'bg-gi-yellow/[0.02]' : ''}`}
            >
              <div className="flex flex-col gap-1">
                {dayShifts.map(shift => (
                  <ShiftCard
                    key={shift.id}
                    shift={shift}
                    isSelected={selectedShift?.id === shift.id}
                    isMine={!!employee && shift.assignments.some(a => a.employee_id === employee.id)}
                    address={projectById.get(shift.project_id)?.address ?? null}
                    onClick={() => onSelectShift(shift)}
                  />
                ))}
                {canManage && (
                  <button
                    type="button"
                    onClick={() => onOpenCreate(day)}
                    className={`flex items-center justify-center rounded-md border border-dashed border-gi-border/40 hover:border-gi-border text-gi-text-subtle/40 hover:text-gi-text-subtle transition ${
                      dayShifts.length === 0 ? 'h-8 w-full' : 'h-5 w-full mt-0.5'
                    }`}
                    aria-label="Dodaj smjenu"
                  >
                    <Plus className="w-3 h-3" />
                  </button>
                )}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}

// ── Schedule matrix (secondary desktop view) ──────────────────────────────

function ScheduleGrid({
  weekDays, today, rows, shiftsByEmpDate, employee, canManage,
  selectedShift, projectById, onSelectShift, onOpenCreate,
}: {
  weekDays: string[]
  today: string
  rows: EmpRow[]
  shiftsByEmpDate: Map<string, Map<string, Shift[]>>
  employee: MeEmployee | null
  canManage: boolean
  selectedShift: Shift | null
  projectById: Map<string, ProjectListItem>
  onSelectShift: (s: Shift) => void
  onOpenCreate: (date: string, empId: string) => void
}) {
  return (
    <div className="rounded-xl border border-gi-border overflow-x-auto">
      <table style={{ minWidth: 860, width: '100%', borderCollapse: 'collapse' }}>
        <thead>
          <tr style={{ position: 'sticky', top: 0, zIndex: 20 }}>
            <th
              className="bg-gi-card px-4 py-3 text-left text-xs font-semibold text-gi-text-subtle uppercase tracking-wider border-b border-r border-gi-border"
              style={{ position: 'sticky', left: 0, zIndex: 30, minWidth: 175, width: 175 }}
            >
              Radnik
            </th>
            {weekDays.map((day, i) => {
              const dt = parseDateLocal(day)
              const isToday = day === today
              return (
                <th
                  key={day}
                  className={`bg-gi-card px-2 py-2 text-center border-b border-l border-gi-border ${isToday ? 'bg-gi-yellow/[0.07]' : ''}`}
                  style={{ minWidth: 115 }}
                >
                  <div className={`text-[10px] font-semibold uppercase tracking-wider ${isToday ? 'text-gi-yellow' : 'text-gi-text-subtle'}`}>
                    {DAY_SHORT[i]}
                  </div>
                  <div className={`text-base font-bold leading-tight ${isToday ? 'text-gi-yellow' : 'text-gi-text'}`}>
                    {dt.getDate()}
                  </div>
                  <div className="text-[10px] text-gi-text-subtle font-normal">
                    {String(dt.getDate()).padStart(2,'0')}.{String(dt.getMonth()+1).padStart(2,'0')}.
                  </div>
                </th>
              )
            })}
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 ? (
            <tr>
              <td colSpan={8} className="px-6 py-16 text-center text-gi-text-subtle text-sm">
                <div className="flex flex-col items-center gap-2">
                  <CalendarDays className="w-8 h-8 text-gi-border" />
                  Nema aktivnih zaposlenika za prikaz.
                </div>
              </td>
            </tr>
          ) : (
            rows.map((row) => {
              const empShifts = shiftsByEmpDate.get(row.id)
              const initials = row.name.split(' ').map((p: string) => p[0]).slice(0,2).join('').toUpperCase()
              return (
                <tr key={row.id} className="hover:bg-white/[0.015] transition-colors group">
                  <td
                    className="bg-gi-workspace px-3 py-2 align-middle border-b border-r border-gi-border/70 group-hover:bg-gi-card/50"
                    style={{ position: 'sticky', left: 0, zIndex: 10 }}
                  >
                    <div className="flex items-center gap-2">
                      <div className="w-7 h-7 rounded-full bg-gi-card-elevated border border-gi-border text-gi-text-muted text-[11px] font-bold flex items-center justify-center flex-shrink-0">
                        {initials}
                      </div>
                      <div className="min-w-0">
                        <div className="text-sm font-medium text-gi-text truncate leading-tight">{row.name}</div>
                        <div className="text-[11px] text-gi-text-subtle truncate">{row.role}</div>
                      </div>
                    </div>
                  </td>
                  {weekDays.map(day => {
                    const dayShifts = empShifts?.get(day) ?? []
                    const isToday = day === today
                    return (
                      <td
                        key={day}
                        className={`px-1 py-1 align-top border-b border-l border-gi-border/70 ${isToday ? 'bg-gi-yellow/[0.03]' : ''}`}
                        style={{ minWidth: 115, verticalAlign: 'top' }}
                      >
                        <div className="flex flex-col gap-0.5 min-h-[52px]">
                          {dayShifts.map(shift => (
                            <ShiftCard
                              key={shift.id}
                              shift={shift}
                              isSelected={selectedShift?.id === shift.id}
                              isMine={!!employee && shift.assignments.some(a => a.employee_id === employee.id)}
                              address={projectById.get(shift.project_id)?.address ?? null}
                              onClick={() => onSelectShift(shift)}
                            />
                          ))}
                          {canManage && (
                            <button
                              type="button"
                              onClick={() => onOpenCreate(day, row.id)}
                              className={`flex items-center justify-center rounded-md border border-dashed border-gi-border/40 hover:border-gi-border text-gi-text-subtle/40 hover:text-gi-text-subtle transition ${
                                dayShifts.length === 0 ? 'w-full h-8' : 'w-full h-5 mt-0.5'
                              }`}
                              aria-label="Dodaj smjenu"
                            >
                              <Plus className="w-3 h-3" />
                            </button>
                          )}
                        </div>
                      </td>
                    )
                  })}
                </tr>
              )
            })
          )}
        </tbody>
      </table>
    </div>
  )
}

// ── Mobile agenda ─────────────────────────────────────────────────────────

function MobileAgenda({
  weekDays, today, mobileDay, onSetMobileDay, shiftsByDate,
  employee, canManage, selectedShift, projectById, onSelectShift, onOpenCreate,
}: {
  weekDays: string[]; today: string; mobileDay: string
  onSetMobileDay: (d: string) => void
  shiftsByDate: Map<string, Shift[]>
  employee: MeEmployee | null; canManage: boolean
  selectedShift: Shift | null; projectById: Map<string, ProjectListItem>
  onSelectShift: (s: Shift) => void; onOpenCreate: (date: string) => void
}) {
  const dayIdx = weekDays.indexOf(mobileDay)
  const dayShifts = shiftsByDate.get(mobileDay) ?? []
  const dt = parseDateLocal(mobileDay)

  return (
    <div className="space-y-4">
      <div className="flex gap-1.5 overflow-x-auto pb-1">
        {weekDays.map((day, i) => {
          const d = parseDateLocal(day)
          const isToday = day === today
          const isActive = day === mobileDay
          const count = shiftsByDate.get(day)?.length ?? 0
          return (
            <button key={day} type="button" onClick={() => onSetMobileDay(day)}
              className={`flex-shrink-0 flex flex-col items-center px-3 py-2 rounded-xl text-xs font-medium transition min-w-[52px] relative ${
                isActive
                  ? 'bg-gi-yellow text-gi-navy'
                  : isToday
                  ? 'bg-gi-card border border-gi-yellow/40 text-gi-yellow'
                  : 'bg-gi-card text-gi-text-muted'
              }`}>
              <span className="text-[10px] uppercase tracking-wide">{DAY_SHORT[i]}</span>
              <span className={`text-lg font-bold leading-tight ${isActive ? 'text-gi-navy' : isToday ? 'text-gi-yellow' : 'text-gi-text'}`}>{d.getDate()}</span>
              {count > 0 && !isActive && <span className="absolute top-1 right-1 w-1.5 h-1.5 rounded-full bg-gi-yellow" />}
            </button>
          )
        })}
      </div>

      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-gi-text-muted">
          {dayIdx >= 0 ? DAY_FULL[dayIdx] : ''}, {dt.getDate()}. {MONTH_GEN[dt.getMonth()]} {dt.getFullYear()}.
        </h2>
        {canManage && (
          <button type="button" onClick={() => onOpenCreate(mobileDay)}
            className="flex items-center gap-1 text-xs text-gi-yellow hover:text-gi-yellow-hover transition">
            <Plus className="w-3.5 h-3.5" />Nova smjena
          </button>
        )}
      </div>

      {dayShifts.length === 0 ? (
        <div className="flex flex-col items-center gap-2 py-14 text-gi-text-subtle">
          <CalendarDays className="w-8 h-8 text-gi-border" />
          <p className="text-sm">Nema smjena za ovaj dan.</p>
        </div>
      ) : (
        <div className="space-y-2">
          {dayShifts.map(shift => {
            const isMine = !!employee && shift.assignments.some(a => a.employee_id === employee.id)
            const isCancelled = shift.status === 'cancelled'
            const color = projectColor(shift.project_id)
            const { r, g, b } = hexToRgb(isCancelled ? '#475569' : color)
            const isSelected = selectedShift?.id === shift.id
            const addr = projectById.get(shift.project_id)?.address
            return (
              <button key={shift.id} type="button" onClick={() => onSelectShift(shift)}
                className={`w-full text-left rounded-xl p-4 transition ${isCancelled ? 'opacity-60' : ''} ${isSelected ? 'ring-1 ring-white/20' : ''}`}
                style={{
                  borderStyle: 'solid', borderWidth: '1px', borderColor: 'rgba(50,68,94,0.5)',
                  borderLeftColor: isCancelled ? '#475569' : color, borderLeftWidth: '4px',
                  background: `rgba(${r},${g},${b},${isSelected ? 0.18 : 0.08})`,
                }}>
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0 flex-1">
                    <div className={`font-semibold text-sm ${isCancelled ? 'line-through text-gi-text-subtle' : 'text-gi-text'}`}>{shift.project_name}</div>
                    {addr && <div className="text-xs text-gi-text-muted mt-0.5">{addr}</div>}
                    {shift.assignments.length > 0 && (
                      <div className="text-xs text-gi-text-subtle mt-1.5 flex items-center gap-1">
                        <Users className="w-3 h-3 flex-shrink-0" />
                        <span className="truncate">{shift.assignments.map(a => a.employee_name).join(', ')}</span>
                      </div>
                    )}
                  </div>
                  <div className="flex flex-col gap-1 flex-shrink-0">
                    {isMine && !isCancelled && <span className="text-[10px] bg-gi-yellow/20 text-gi-yellow px-1.5 py-0.5 rounded-full font-medium">Moja</span>}
                    {isCancelled && <span className="text-[10px] bg-red-900/40 text-red-400 px-1.5 py-0.5 rounded-full font-medium">Otkazana</span>}
                  </div>
                </div>
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}

// ── Detail panel ──────────────────────────────────────────────────────────

function ShiftDetailPanel({
  shift, employee, canManage, cancellingId, projectAddress,
  onEdit, onCancel, onDuplicate, onAssign, onClose,
}: {
  shift: Shift; employee: MeEmployee | null; canManage: boolean
  cancellingId: string | null; projectAddress: string | null
  onEdit: () => void; onCancel: () => void; onDuplicate: () => void
  onAssign: () => void; onClose: () => void
}) {
  const isCancelled = shift.status === 'cancelled'
  const isMine = !!employee && shift.assignments.some(a => a.employee_id === employee.id)
  const color = projectColor(shift.project_id)

  return (
    <div className="mt-4 rounded-xl border border-gi-border bg-gi-card overflow-hidden">
      <div className="h-0.5" style={{ background: isCancelled ? '#475569' : color }} />
      <div className="p-4 sm:p-5">
        <div className="flex items-start justify-between gap-3 mb-4">
          <div>
            <p className="text-[10px] text-gi-text-subtle uppercase tracking-wider mb-1">Detalji smjene</p>
            <div className="flex items-center gap-2 flex-wrap">
              <h3 className={`text-base font-semibold ${isCancelled ? 'line-through text-gi-text-muted' : 'text-gi-text'}`}>{shift.project_name}</h3>
              {isMine && !isCancelled && <span className="text-[10px] bg-gi-yellow/20 text-gi-yellow px-1.5 py-0.5 rounded-full font-medium">Moja smjena</span>}
              {isCancelled && <span className="text-[10px] bg-red-900/50 text-red-400 px-1.5 py-0.5 rounded-full font-medium">Otkazana</span>}
            </div>
          </div>
          <button type="button" onClick={onClose} className="text-gi-text-subtle hover:text-gi-text transition p-1 -mr-1 -mt-1 flex-shrink-0">
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="grid grid-cols-2 sm:grid-cols-3 gap-x-6 gap-y-3 mb-4 text-sm">
          <div>
            <p className="text-[10px] text-gi-text-subtle uppercase tracking-wider mb-0.5">Datum smjene</p>
            <p className="text-gi-text">{shift.shift_date}</p>
          </div>
          {projectAddress && (
            <div>
              <p className="text-[10px] text-gi-text-subtle uppercase tracking-wider mb-0.5">Lokacija</p>
              <p className="text-gi-text">{projectAddress}</p>
            </div>
          )}
          <div>
            <p className="text-[10px] text-gi-text-subtle uppercase tracking-wider mb-0.5">Raspoređeni radnici</p>
            <p className="text-gi-text text-sm">
              {shift.assignments.length === 0
                ? <span className="text-gi-text-subtle">Nema raspoređenih</span>
                : shift.assignments.map(a => a.employee_name).join(', ')}
            </p>
          </div>
        </div>

        {shift.notes && (
          <div className="mb-4 p-3 rounded-lg bg-gi-card-elevated border border-gi-border/40">
            <p className="text-[10px] text-gi-text-subtle uppercase tracking-wider mb-1">Napomene / Opis rada</p>
            <p className="text-sm text-gi-text-muted">{shift.notes}</p>
          </div>
        )}

        {canManage && !isCancelled && (
          <div className="flex flex-wrap gap-2">
            <button type="button" onClick={onEdit}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-gi-card-elevated hover:bg-gi-border text-gi-text-muted hover:text-gi-text text-xs font-medium transition">
              <Pencil className="w-3.5 h-3.5" />Uredi
            </button>
            <button type="button" onClick={onAssign}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-gi-card-elevated hover:bg-gi-border text-gi-text-muted hover:text-gi-text text-xs font-medium transition">
              <Users className="w-3.5 h-3.5" />Radnici
            </button>
            <button type="button" onClick={onDuplicate}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-gi-card-elevated hover:bg-gi-border text-gi-text-muted hover:text-gi-text text-xs font-medium transition">
              <Copy className="w-3.5 h-3.5" />Dupliciraj
            </button>
            <button type="button" onClick={onCancel} disabled={cancellingId === shift.id}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-gi-card-elevated hover:bg-red-900/40 text-gi-text-muted hover:text-red-300 text-xs font-medium transition disabled:opacity-50">
              <Ban className="w-3.5 h-3.5" />
              {cancellingId === shift.id ? 'Otkazivanje...' : 'Otkaži smjenu'}
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

// ── Main page ─────────────────────────────────────────────────────────────

export default function SchedulePage() {
  const { user, employee, isLoading, logout } = useAuth()
  const router = useRouter()
  const canManage = user?.role === 'direktor' || user?.role === 'inzenjer'

  const [weekStart, setWeekStart] = useState<Date>(() => getMonday(new Date()))
  const [activeTab, setActiveTab] = useState<ViewTab>('calendar')

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

  const [showForm, setShowForm] = useState(false)
  const [editId, setEditId] = useState<string | null>(null)
  const [form, setForm] = useState<ShiftForm>(DEFAULT_FORM)
  const [formEmployees, setFormEmployees] = useState<Set<string>>(new Set())
  const [formError, setFormError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [formAvailability, setFormAvailability] = useState<Map<string, EmployeeForDate>>(new Map())
  const [formAvailLoading, setFormAvailLoading] = useState(false)

  const [selectedShift, setSelectedShift] = useState<Shift | null>(null)
  const [cancellingId, setCancellingId] = useState<string | null>(null)

  const [assignShift, setAssignShift] = useState<Shift | null>(null)
  const [selEmployees, setSelEmployees] = useState<Set<string>>(new Set())
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
  const isCurrentWeek = weekDays.includes(today)

  // ── Data fetching ──────────────────────────────────────────────────────

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

  useEffect(() => {
    if (!isLoading && user?.role === 'administracija') router.replace('/dashboard')
  }, [isLoading, user, router])

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

  // ── Derived data ──────────────────────────────────────────────────────

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

  // ── Navigation ────────────────────────────────────────────────────────

  function prevWeek() { setWeekStart(w => addDays(w, -7)) }
  function nextWeek() { setWeekStart(w => addDays(w, 7)) }
  function goToday() { setWeekStart(getMonday(new Date())); setMobileDay(toDateStr(new Date())) }

  // ── Form ──────────────────────────────────────────────────────────────

  function openCreate(date?: string) {
    setEditId(null)
    setForm({ ...DEFAULT_FORM, shift_date: date ?? dateFrom })
    setFormEmployees(new Set())
    setFormError(null)
    setShowForm(true)
    setSelectedShift(null)
  }

  function openEdit(shift: Shift) {
    setEditId(shift.id)
    setForm({ project_id: shift.project_id, shift_date: shift.shift_date, notes: shift.notes ?? '' })
    setFormEmployees(new Set())
    setFormError(null)
    setShowForm(true)
  }

  async function submitCreate() {
    setSaving(true)
    setFormError(null)
    try {
      await apiClient.post('/schedule/shifts', {
        project_id: form.project_id,
        shift_date: form.shift_date,
        notes: form.notes || null,
        employee_ids: [...formEmployees],
        override_overlaps: true,
      })
      setShowForm(false)
      fetchShifts()
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setFormError(e?.response?.data?.error ?? 'Greška pri kreiranju smjene.')
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

  // ── Cancel & Duplicate ────────────────────────────────────────────────

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

  // ── Assignment modal ──────────────────────────────────────────────────

  function openAssign(shift: Shift) {
    setAssignShift(shift)
    setSelEmployees(new Set(shift.assignments.map(a => a.employee_id)))
  }

  async function handleSaveAssignments() {
    if (!assignShift) return
    setAssigning(true)
    try {
      await apiClient.put(`/schedule/shifts/${assignShift.id}/assignments`, {
        employee_ids: [...selEmployees],
        override_overlaps: true,
      })
      setAssignShift(null)
      fetchShifts()
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      alert(e?.response?.data?.error ?? 'Greška pri raspoređivanju.')
    } finally {
      setAssigning(false)
    }
  }

  // ── Render ────────────────────────────────────────────────────────────

  if (isLoading) return <LoadingScreen />
  if (!user || user.role === 'administracija') return null

  const hasActiveFilters = !!(projectFilter || employeeFilter || roleFilter || myShiftsOnly)

  return (
    <AppShell user={user} employee={employee} title="Raspored smjena" backHref="/dashboard" onLogout={logout}>

      {/* ── Page header ── */}
      <div className="flex flex-col gap-3 mb-5">
        <div className="flex flex-col md:flex-row md:items-center gap-3">
          <div>
            <h1 className="text-xl font-bold text-gi-text">Raspored radova</h1>
            <p className="text-xs text-gi-text-subtle">Tjedni pregled</p>
          </div>

          <div className="flex items-center gap-1 md:ml-5">
            <button onClick={prevWeek} aria-label="Prethodni tjedan"
              className="p-2 rounded-lg text-gi-text-muted hover:text-gi-text hover:bg-gi-card transition">
              <ChevronLeft className="w-4 h-4" />
            </button>
            <span className="px-3 py-1.5 text-sm font-medium text-gi-text min-w-[200px] text-center">
              {weekLabel(weekStart)}
            </span>
            <button onClick={nextWeek} aria-label="Sljedeći tjedan"
              className="p-2 rounded-lg text-gi-text-muted hover:text-gi-text hover:bg-gi-card transition">
              <ChevronRight className="w-4 h-4" />
            </button>
            {!isCurrentWeek && (
              <button onClick={goToday}
                className="px-3 py-1.5 rounded-lg text-xs font-medium text-gi-yellow border border-gi-yellow/40 hover:border-gi-yellow/70 hover:bg-gi-yellow/10 transition">
                Danas
              </button>
            )}
          </div>

          {/* Tab switcher — desktop only */}
          <div className="hidden md:flex items-center gap-0.5 border border-gi-border rounded-lg p-0.5 md:ml-auto">
            <button
              type="button"
              onClick={() => setActiveTab('calendar')}
              className={`px-3 py-1.5 rounded-md text-xs font-medium transition ${
                activeTab === 'calendar'
                  ? 'bg-gi-card-elevated text-gi-text'
                  : 'text-gi-text-subtle hover:text-gi-text-muted'
              }`}
            >
              Tjedni pregled
            </button>
            <button
              type="button"
              onClick={() => setActiveTab('matrix')}
              className={`px-3 py-1.5 rounded-md text-xs font-medium transition ${
                activeTab === 'matrix'
                  ? 'bg-gi-card-elevated text-gi-text'
                  : 'text-gi-text-subtle hover:text-gi-text-muted'
              }`}
            >
              Po zaposlenicima
            </button>
          </div>

          {canManage && (
            <button onClick={() => openCreate()}
              className="flex items-center gap-1.5 px-4 py-2 rounded-lg bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy text-sm font-semibold transition">
              <Plus className="w-4 h-4" />Novi raspored
            </button>
          )}
        </div>

        {/* Filter bar */}
        <div className="flex flex-wrap items-center gap-2">
          <select value={projectFilter} onChange={e => setProjectFilter(e.target.value)}
            className="px-3 py-1.5 rounded-lg text-xs bg-gi-card-elevated border border-gi-border text-gi-text-muted focus:outline-none focus:border-gi-yellow">
            <option value="">Svi projekti</option>
            {projects.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
          </select>

          <select value={employeeFilter} onChange={e => setEmployeeFilter(e.target.value)}
            className="px-3 py-1.5 rounded-lg text-xs bg-gi-card-elevated border border-gi-border text-gi-text-muted focus:outline-none focus:border-gi-yellow">
            <option value="">Svi radnici</option>
            {empRows.map(r => <option key={r.id} value={r.id}>{r.name}</option>)}
          </select>

          <select value={roleFilter} onChange={e => setRoleFilter(e.target.value)}
            className="px-3 py-1.5 rounded-lg text-xs bg-gi-card-elevated border border-gi-border text-gi-text-muted focus:outline-none focus:border-gi-yellow">
            <option value="">Sve uloge</option>
            {Object.entries(ROLE_LABELS).map(([k, v]) => <option key={k} value={k}>{v}</option>)}
          </select>

          {employee && (
            <label className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs bg-gi-card-elevated border border-gi-border text-gi-text-muted cursor-pointer select-none">
              <input type="checkbox" checked={myShiftsOnly} onChange={e => setMyShiftsOnly(e.target.checked)} className="w-3 h-3" />
              Samo moje smjene
            </label>
          )}

          <label className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs bg-gi-card-elevated border border-gi-border text-gi-text-muted cursor-pointer select-none">
            <input type="checkbox" checked={showCancelled} onChange={e => setShowCancelled(e.target.checked)} className="w-3 h-3" />
            Prikaži otkazane
          </label>

          {hasActiveFilters && (
            <button onClick={() => { setProjectFilter(''); setEmployeeFilter(''); setRoleFilter(''); setMyShiftsOnly(false) }}
              className="px-3 py-1.5 rounded-lg text-xs text-gi-text-muted hover:text-gi-text hover:bg-gi-card transition">
              Poništi filtere
            </button>
          )}
        </div>
      </div>

      {/* ── Content ── */}
      {loading ? (
        <div className="flex items-center justify-center h-64 border border-gi-border rounded-xl bg-gi-card">
          <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje rasporeda...</p>
        </div>
      ) : error ? (
        <div className="flex items-center justify-center h-64 border border-red-900/30 rounded-xl">
          <p className="text-red-400 text-sm">{error}</p>
        </div>
      ) : (
        <>
          {/* Desktop views */}
          <div className="hidden md:block">
            {activeTab === 'calendar' ? (
              <WeekCalendar
                weekDays={weekDays}
                today={today}
                shiftsByDate={shiftsByDate}
                employee={employee}
                canManage={canManage}
                selectedShift={selectedShift}
                projectById={projectById}
                onSelectShift={setSelectedShift}
                onOpenCreate={date => openCreate(date)}
              />
            ) : (
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
            )}
          </div>

          {/* Mobile */}
          <div className="md:hidden">
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

          <div className="mt-4 flex flex-wrap items-center gap-4 text-xs text-gi-text-subtle">
            <span className="font-medium text-gi-text-muted">Legenda:</span>
            <span className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-full bg-gi-yellow inline-block" />Aktivna smjena</span>
            <span className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-full bg-gi-border inline-block" />Otkazana smjena</span>
            <span className="flex items-center gap-1.5"><span className="w-2 h-3 rounded-sm bg-gi-yellow/30 border-l-2 border-gi-yellow inline-block" />Moja smjena</span>
          </div>
        </>
      )}

      {/* ── Create / Edit modal ── */}
      {showForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70">
          <div className="bg-gi-card border border-gi-border rounded-2xl w-full max-w-lg shadow-2xl max-h-[90vh] flex flex-col">
            <div className="flex items-center justify-between p-4 border-b border-gi-border flex-shrink-0">
              <h3 className="text-sm font-semibold text-gi-text">{editId ? 'Uredi smjenu' : 'Nova smjena'}</h3>
              <button type="button" onClick={() => setShowForm(false)} className="text-gi-text-subtle hover:text-gi-text transition p-1">
                <X className="w-4 h-4" />
              </button>
            </div>

            <form onSubmit={handleFormSubmit} className="flex flex-col overflow-hidden flex-1">
              <div className="overflow-y-auto flex-1 p-4 space-y-4">
                {!editId && (
                  <div>
                    <label className="block text-xs text-gi-text-muted mb-1">Projekt *</label>
                    <select required value={form.project_id}
                      onChange={e => setForm(f => ({ ...f, project_id: e.target.value }))}
                      className="w-full px-3 py-2 rounded-lg bg-gi-card-elevated border border-gi-border text-gi-text text-sm focus:outline-none focus:border-gi-yellow">
                      <option value="">Odaberite projekt...</option>
                      {projects.map(p => <option key={p.id} value={p.id}>{p.name}{p.address ? ` — ${p.address}` : ''}</option>)}
                    </select>
                  </div>
                )}

                <div>
                  <label className="block text-xs text-gi-text-muted mb-1">Datum smjene *</label>
                  <input required type="date" value={form.shift_date}
                    onChange={e => setForm(f => ({ ...f, shift_date: e.target.value }))}
                    className="w-full px-3 py-2 rounded-lg bg-gi-card-elevated border border-gi-border text-gi-text text-sm focus:outline-none focus:border-gi-yellow" />
                </div>

                <div>
                  <label className="block text-xs text-gi-text-muted mb-1">Napomene / Opis rada</label>
                  <textarea rows={2} value={form.notes}
                    onChange={e => setForm(f => ({ ...f, notes: e.target.value }))}
                    placeholder="Opcionalno..."
                    className="w-full px-3 py-2 rounded-lg bg-gi-card-elevated border border-gi-border text-gi-text text-sm focus:outline-none focus:border-gi-yellow resize-none" />
                </div>

                {!editId && (
                  <div>
                    <div className="flex items-center justify-between mb-1.5">
                      <label className="text-xs text-gi-text-muted">
                        Zaposlenici *{' '}
                        <span className="text-gi-text-subtle">({formEmployees.size} odabrano)</span>
                      </label>
                      {formAvailLoading && <span className="text-[10px] text-gi-text-subtle animate-pulse">Učitavanje...</span>}
                    </div>
                    <EmployeePicker
                      employees={allEmployees}
                      availability={formAvailability}
                      selected={formEmployees}
                      onChange={setFormEmployees}
                    />
                  </div>
                )}

                {formError && <p className="text-sm text-red-400">{formError}</p>}
              </div>

              <div className="flex gap-2 p-4 border-t border-gi-border flex-shrink-0">
                <button type="submit" disabled={saving}
                  className="flex-1 py-2 rounded-lg bg-gi-yellow hover:bg-gi-yellow-hover disabled:opacity-50 text-gi-navy text-sm font-semibold transition">
                  {saving ? 'Sprema...' : editId ? 'Spremi izmjene' : 'Kreiraj smjenu'}
                </button>
                <button type="button" onClick={() => setShowForm(false)}
                  className="px-4 py-2 rounded-lg text-gi-text-muted hover:text-gi-text hover:bg-gi-card-elevated text-sm transition">
                  Odustani
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* ── Assignment modal ── */}
      {assignShift && (
        <div className="fixed inset-0 z-50 flex items-end md:items-center justify-center p-4 bg-black/70">
          <div className="bg-gi-card border border-gi-border rounded-2xl w-full max-w-md max-h-[85vh] flex flex-col shadow-2xl">
            <div className="flex items-start justify-between p-4 border-b border-gi-border">
              <div>
                <p className="text-sm font-semibold text-gi-text">Uredi raspoređene radnike</p>
                <p className="text-xs text-gi-text-muted mt-0.5">
                  {assignShift.project_name} · {assignShift.shift_date}
                </p>
              </div>
              <button type="button" onClick={() => setAssignShift(null)}
                className="text-gi-text-subtle hover:text-gi-text transition p-1">
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="overflow-y-auto flex-1 p-3">
              <EmployeePicker
                employees={allEmployees}
                availability={assignAvailability}
                selected={selEmployees}
                onChange={setSelEmployees}
              />
            </div>

            <div className="p-4 border-t border-gi-border flex gap-2">
              <button type="button" onClick={() => handleSaveAssignments()} disabled={assigning}
                className="flex-1 py-2 rounded-lg bg-gi-yellow hover:bg-gi-yellow-hover disabled:opacity-50 text-gi-navy text-sm font-semibold transition">
                {assigning ? 'Sprema...' : `Spremi (${selEmployees.size})`}
              </button>
              <button type="button" onClick={() => setAssignShift(null)}
                className="px-4 py-2 rounded-lg text-gi-text-muted hover:text-gi-text hover:bg-gi-card-elevated text-sm transition">
                Odustani
              </button>
            </div>
          </div>
        </div>
      )}
    </AppShell>
  )
}
