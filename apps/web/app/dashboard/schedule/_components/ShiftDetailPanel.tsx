import { X, Pencil, Users, Copy, Ban } from 'lucide-react'
import type { MeEmployee } from '@/hooks/useAuth'
import type { Shift } from '@/lib/types/schedule'
import { projectColor } from './schedule-utils'

export default function ShiftDetailPanel({
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
    <div className="mt-4 rounded-xl border border-slate-800 bg-slate-900 overflow-hidden">
      <div className="h-0.5" style={{ background: isCancelled ? '#475569' : color }} />
      <div className="p-4 sm:p-5">
        <div className="flex items-start justify-between gap-3 mb-4">
          <div>
            <p className="text-[10px] text-slate-500 uppercase tracking-wider mb-1">Detalji smjene</p>
            <div className="flex items-center gap-2 flex-wrap">
              <h3 className={`text-base font-semibold ${isCancelled ? 'line-through text-slate-400' : 'text-white'}`}>{shift.project_name}</h3>
              {isMine && !isCancelled && <span className="text-[10px] bg-blue-900/50 text-blue-300 px-1.5 py-0.5 rounded-full font-medium">Moja smjena</span>}
              {isCancelled && <span className="text-[10px] bg-red-900/50 text-red-400 px-1.5 py-0.5 rounded-full font-medium">Otkazana</span>}
            </div>
          </div>
          <button type="button" onClick={onClose} className="text-slate-500 hover:text-white transition p-1 -mr-1 -mt-1 flex-shrink-0">
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="grid grid-cols-2 sm:grid-cols-3 gap-x-6 gap-y-3 mb-4 text-sm">
          <div>
            <p className="text-[10px] text-slate-500 uppercase tracking-wider mb-0.5">Datum smjene</p>
            <p className="text-white">{shift.shift_date}</p>
          </div>
          {projectAddress && (
            <div>
              <p className="text-[10px] text-slate-500 uppercase tracking-wider mb-0.5">Lokacija</p>
              <p className="text-white">{projectAddress}</p>
            </div>
          )}
          <div>
            <p className="text-[10px] text-slate-500 uppercase tracking-wider mb-0.5">Raspoređeni radnici</p>
            <p className="text-white text-sm">
              {shift.assignments.length === 0
                ? <span className="text-slate-500">Nema raspoređenih</span>
                : shift.assignments.map(a => a.employee_name).join(', ')}
            </p>
          </div>
        </div>

        {shift.notes && (
          <div className="mb-4 p-3 rounded-lg bg-slate-800/50 border border-slate-700/40">
            <p className="text-[10px] text-slate-500 uppercase tracking-wider mb-1">Napomene / Opis rada</p>
            <p className="text-sm text-slate-300">{shift.notes}</p>
          </div>
        )}

        {canManage && !isCancelled && (
          <div className="flex flex-wrap gap-2">
            <button type="button" onClick={onEdit}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white text-xs font-medium transition">
              <Pencil className="w-3.5 h-3.5" />Uredi
            </button>
            <button type="button" onClick={onAssign}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white text-xs font-medium transition">
              <Users className="w-3.5 h-3.5" />Radnici
            </button>
            <button type="button" onClick={onDuplicate}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white text-xs font-medium transition">
              <Copy className="w-3.5 h-3.5" />Dupliciraj
            </button>
            <button type="button" onClick={onCancel} disabled={cancellingId === shift.id}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-red-900/50 text-slate-400 hover:text-red-300 text-xs font-medium transition disabled:opacity-50">
              <Ban className="w-3.5 h-3.5" />
              {cancellingId === shift.id ? 'Otkazivanje...' : 'Otkaži smjenu'}
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
