'use client'

import { type ActivityInputUI } from '@/lib/types/daily-reports'
import ActivityTypeBadge from './ActivityTypeBadge'

interface Props {
  activities: ActivityInputUI[]
  onRemove: (tempId: string) => void
  onEdit?: (activity: ActivityInputUI) => void
  disabled?: boolean
}

export default function ActivityListPreview({ activities, onRemove, onEdit, disabled = false }: Props) {
  if (activities.length === 0) {
    return (
      <div className="text-sm text-gi-text-subtle py-5 text-center border border-dashed border-gi-border rounded-lg">
        Nema dodanih aktivnosti.
      </div>
    )
  }

  return (
    <div className="space-y-1.5">
      {activities.map((a, idx) => (
        <div
          key={a._tempId}
          className="flex items-start gap-3 bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2.5"
        >
          <span className="text-xs text-gi-text-subtle w-5 pt-0.5 shrink-0 select-none">{idx + 1}.</span>

          <div className="flex-1 min-w-0 space-y-1">
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-sm text-gi-text font-medium truncate">{a._displayName}</span>
              <ActivityTypeBadge type={a.activity_type} isVtk={a.is_vtk} trackingType={a.tracking_type} />
            </div>
            <div className="flex flex-wrap gap-x-4 gap-y-0.5 text-xs text-gi-text-subtle">
              <span>
                Količina: <span className="text-gi-text-muted">{a.quantity} {a.unit}</span>
              </span>
              {a.notes && (
                <span className="truncate max-w-xs">
                  Napomena: <span className="text-gi-text-muted">{a.notes}</span>
                </span>
              )}
            </div>
          </div>

          {!disabled && (
            <div className="flex items-center gap-1 shrink-0 mt-0.5">
              {onEdit && (
                <button
                  type="button"
                  onClick={() => onEdit(a)}
                  className="text-gi-text-subtle hover:text-gi-yellow transition-colors p-0.5"
                  title="Uredi aktivnost"
                >
                  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                  </svg>
                </button>
              )}
              <button
                type="button"
                onClick={() => onRemove(a._tempId)}
                className="text-gi-text-subtle hover:text-red-400 transition-colors p-0.5"
                title="Ukloni aktivnost"
              >
                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          )}
        </div>
      ))}
    </div>
  )
}
