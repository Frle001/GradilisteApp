import { CalendarDays, Plus } from 'lucide-react'
import type { MeEmployee } from '@/hooks/useAuth'
import type { Shift } from '@/lib/types/schedule'
import type { ProjectListItem } from '@/lib/types/projects'
import { DAY_SHORT, parseDateLocal, type EmpRow } from './schedule-utils'
import ShiftCard from './ShiftCard'

export default function ScheduleGrid({
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
    <div className="rounded-xl border border-slate-800 overflow-x-auto bg-slate-950">
      <table style={{ minWidth: 860, width: '100%', borderCollapse: 'collapse' }}>
        <thead>
          <tr style={{ position: 'sticky', top: 0, zIndex: 20 }}>
            <th
              className="bg-slate-900 px-4 py-3 text-left text-xs font-semibold text-slate-500 uppercase tracking-wider border-b border-r border-slate-800"
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
                  className={`bg-slate-900 px-2 py-2 text-center border-b border-l border-slate-800 ${isToday ? 'bg-blue-950/50' : ''}`}
                  style={{ minWidth: 115 }}
                >
                  <div className={`text-[10px] font-semibold uppercase tracking-wider ${isToday ? 'text-blue-400' : 'text-slate-500'}`}>
                    {DAY_SHORT[i]}
                  </div>
                  <div className={`text-base font-bold leading-tight ${isToday ? 'text-blue-300' : 'text-slate-200'}`}>
                    {dt.getDate()}
                  </div>
                  <div className="text-[10px] text-slate-600 font-normal">
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
              <td colSpan={8} className="px-6 py-16 text-center text-slate-600 text-sm">
                <div className="flex flex-col items-center gap-2">
                  <CalendarDays className="w-8 h-8 text-slate-800" />
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
                    className="bg-slate-950 px-3 py-2 align-middle border-b border-r border-slate-800/70 group-hover:bg-slate-900/50"
                    style={{ position: 'sticky', left: 0, zIndex: 10 }}
                  >
                    <div className="flex items-center gap-2">
                      <div className="w-7 h-7 rounded-full bg-slate-800 border border-slate-700 text-slate-400 text-[11px] font-bold flex items-center justify-center flex-shrink-0">
                        {initials}
                      </div>
                      <div className="min-w-0">
                        <div className="text-sm font-medium text-white truncate leading-tight">{row.name}</div>
                        <div className="text-[11px] text-slate-500 truncate">{row.role}</div>
                      </div>
                    </div>
                  </td>
                  {weekDays.map(day => {
                    const dayShifts = empShifts?.get(day) ?? []
                    const isToday = day === today
                    return (
                      <td
                        key={day}
                        className={`px-1 py-1 align-top border-b border-l border-slate-800/70 ${isToday ? 'bg-blue-950/10' : ''}`}
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
                              className={`flex items-center justify-center rounded-md border border-dashed transition text-slate-700 hover:border-slate-600 hover:text-slate-500 ${
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
