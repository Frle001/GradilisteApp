'use client'

import { type ActivityInputUI } from '@/lib/types/daily-reports'
import ActivityTypeBadge from './ActivityTypeBadge'

interface Props {
  activities: ActivityInputUI[]
  onRemove: (tempId: string) => void
  disabled?: boolean
}

export default function ActivityListPreview({ activities, onRemove, disabled = false }: Props) {
  if (activities.length === 0) {
    return (
      <div className="text-sm text-slate-500 py-4 text-center border border-dashed border-slate-700 rounded-lg">
        Nema dodanih aktivnosti.
      </div>
    )
  }

  return (
    <div className="space-y-1.5">
      {activities.map((a, idx) => (
        <div
          key={a._tempId}
          className="flex items-start gap-3 bg-slate-800/60 border border-slate-700 rounded-lg px-3 py-2.5"
        >
          <span className="text-xs text-slate-600 w-5 pt-0.5 shrink-0">{idx + 1}.</span>

          <div className="flex-1 min-w-0 space-y-1">
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-sm text-slate-100 font-medium truncate">{a._displayName}</span>
              <ActivityTypeBadge type={a.activity_type} isVtk={a.is_vtk} />
            </div>
            <div className="flex flex-wrap gap-x-4 gap-y-0.5 text-xs text-slate-400">
              <span>
                Količina: <span className="text-slate-200">{a.quantity} {a.unit}</span>
              </span>
              {a.notes && (
                <span className="truncate max-w-xs">
                  Napomena: <span className="text-slate-300">{a.notes}</span>
                </span>
              )}
            </div>
          </div>

          {!disabled && (
            <button
              type="button"
              onClick={() => onRemove(a._tempId)}
              className="text-slate-600 hover:text-red-400 transition-colors shrink-0 mt-0.5"
              title="Ukloni aktivnost"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          )}
        </div>
      ))}
    </div>
  )
}
