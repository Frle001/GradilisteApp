import type { Shift } from '@/lib/types/schedule'
import { projectColor, hexToRgb } from './schedule-utils'

export default function ShiftCard({
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
      <div className={`font-semibold truncate leading-tight ${isCancelled ? 'line-through text-slate-500' : 'text-white'}`}>
        {shift.project_name}
      </div>
      {address && !isCancelled && (
        <div className="text-slate-400 truncate text-[10px]">{address}</div>
      )}
      {(isMine && !isCancelled) && (
        <div className="mt-0.5">
          <span className="rounded-full bg-blue-600/40 text-blue-300 px-1 py-px text-[9px] font-medium">Moja</span>
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
