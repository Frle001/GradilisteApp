import { CalendarDays, Plus, Users } from 'lucide-react'
import type { MeEmployee } from '@/hooks/useAuth'
import type { Shift } from '@/lib/types/schedule'
import type { ProjectListItem } from '@/lib/types/projects'
import { DAY_SHORT, DAY_FULL, MONTH_GEN, parseDateLocal, projectColor, hexToRgb } from './schedule-utils'

export default function MobileAgenda({
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
                isActive ? 'bg-blue-600 text-white' : isToday ? 'bg-slate-800 text-blue-400 border border-blue-800' : 'bg-slate-800 text-slate-400'
              }`}>
              <span className="text-[10px] uppercase tracking-wide">{DAY_SHORT[i]}</span>
              <span className={`text-lg font-bold leading-tight ${isActive ? 'text-white' : isToday ? 'text-blue-300' : 'text-slate-200'}`}>{d.getDate()}</span>
              {count > 0 && !isActive && <span className="absolute top-1 right-1 w-1.5 h-1.5 rounded-full bg-blue-500" />}
            </button>
          )
        })}
      </div>

      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-slate-300">
          {dayIdx >= 0 ? DAY_FULL[dayIdx] : ''}, {dt.getDate()}. {MONTH_GEN[dt.getMonth()]} {dt.getFullYear()}.
        </h2>
        {canManage && (
          <button type="button" onClick={() => onOpenCreate(mobileDay)}
            className="flex items-center gap-1 text-xs text-blue-400 hover:text-blue-300 transition">
            <Plus className="w-3.5 h-3.5" />Nova smjena
          </button>
        )}
      </div>

      {dayShifts.length === 0 ? (
        <div className="flex flex-col items-center gap-2 py-14 text-slate-700">
          <CalendarDays className="w-8 h-8" />
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
                  borderStyle: 'solid', borderWidth: '1px', borderColor: `rgba(51,65,85,0.5)`,
                  borderLeftColor: isCancelled ? '#475569' : color, borderLeftWidth: '4px',
                  background: `rgba(${r},${g},${b},${isSelected ? 0.18 : 0.08})`,
                }}>
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0 flex-1">
                    <div className={`font-semibold text-sm ${isCancelled ? 'line-through text-slate-500' : 'text-white'}`}>{shift.project_name}</div>
                    {addr && <div className="text-xs text-slate-400 mt-0.5">{addr}</div>}
                    {shift.assignments.length > 0 && (
                      <div className="text-xs text-slate-500 mt-1.5 flex items-center gap-1">
                        <Users className="w-3 h-3 flex-shrink-0" />
                        <span className="truncate">{shift.assignments.map(a => a.employee_name).join(', ')}</span>
                      </div>
                    )}
                  </div>
                  <div className="flex flex-col gap-1 flex-shrink-0">
                    {isMine && !isCancelled && <span className="text-[10px] bg-blue-600/40 text-blue-300 px-1.5 py-0.5 rounded-full font-medium">Moja</span>}
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
