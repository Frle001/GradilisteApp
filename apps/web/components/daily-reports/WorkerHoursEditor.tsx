'use client'

import { type FormDataWorker, type WorkerHoursEntry } from '@/lib/types/daily-reports'

interface Props {
  workers: FormDataWorker[]
  entries: WorkerHoursEntry[]
  onChange: (entries: WorkerHoursEntry[]) => void
  disabled?: boolean
}

const QUICK_HOURS = [0, 4, 8]

export default function WorkerHoursEditor({ workers, entries, onChange, disabled = false }: Props) {
  function getEntry(workerId: string): WorkerHoursEntry {
    return (
      entries.find(e => e.worker_id === workerId) ?? {
        worker_id: workerId,
        worker_name: '',
        hours: '',
        notes: '',
      }
    )
  }

  function updateEntry(workerId: string, workerName: string, patch: Partial<WorkerHoursEntry>) {
    const existing = entries.find(e => e.worker_id === workerId)
    if (existing) {
      onChange(entries.map(e => (e.worker_id === workerId ? { ...e, ...patch } : e)))
    } else {
      const base: WorkerHoursEntry = { worker_id: workerId, worker_name: workerName, hours: '', notes: '' }
      onChange([...entries, { ...base, ...patch }])
    }
  }

  if (workers.length === 0) {
    return (
      <div className="text-sm text-slate-500 py-3 text-center">
        Nema radnika dodijeljenih ovom poslovođi.
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {/* Desktop column headers — hidden on mobile */}
      <div className="hidden sm:grid sm:grid-cols-[1fr_140px_1fr] gap-x-3 px-2 pb-1 text-xs font-medium text-slate-500 uppercase tracking-wide">
        <span>Radnik</span>
        <span>Sati</span>
        <span>Napomena</span>
      </div>

      {workers.map(w => {
        const entry = getEntry(w.id)
        const hoursNum = parseFloat(entry.hours)
        const hoursError =
          entry.hours !== '' && (isNaN(hoursNum) || hoursNum < 0 || hoursNum > 24)
            ? 'Mora biti 0–24'
            : null

        return (
          <div
            key={w.id}
            className="rounded-lg bg-slate-800/40 hover:bg-slate-800/60 px-3 py-3 sm:px-2 sm:py-1.5 transition-colors
              sm:grid sm:grid-cols-[1fr_140px_1fr] sm:gap-x-3 sm:items-center sm:rounded-none sm:bg-transparent sm:hover:bg-slate-800/40"
          >
            {/* Worker name */}
            <span className="block text-sm font-medium text-slate-200 mb-2 sm:mb-0 truncate">
              {w.full_name}
            </span>

            {/* Hours — quick buttons + input */}
            <div className="flex items-center gap-2 mb-2 sm:mb-0">
              {/* Quick-fill buttons */}
              <div className="flex gap-1 shrink-0">
                {QUICK_HOURS.map(h => (
                  <button
                    key={h}
                    type="button"
                    disabled={disabled}
                    onClick={() => updateEntry(w.id, w.full_name, { hours: String(h) })}
                    className={`px-2 py-1 text-xs rounded border transition-colors disabled:opacity-40 ${
                      entry.hours === String(h)
                        ? 'bg-blue-600 border-blue-500 text-white'
                        : 'border-slate-700 text-slate-400 hover:border-slate-500 hover:text-slate-200'
                    }`}
                    aria-label={`${h} sati`}
                  >
                    {h}h
                  </button>
                ))}
              </div>
              {/* Manual input */}
              <div className="flex-1 min-w-0">
                <input
                  type="number"
                  inputMode="decimal"
                  min={0}
                  max={24}
                  step={0.5}
                  placeholder="—"
                  value={entry.hours}
                  disabled={disabled}
                  onChange={e => updateEntry(w.id, w.full_name, { hours: e.target.value })}
                  className={`w-full bg-slate-800 border rounded px-2 py-1.5 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-blue-500 ${
                    hoursError ? 'border-red-500' : 'border-slate-700'
                  } disabled:opacity-50`}
                />
                {hoursError && <p className="text-xs text-red-400 mt-0.5">{hoursError}</p>}
              </div>
            </div>

            {/* Notes */}
            <input
              type="text"
              placeholder="Napomena…"
              value={entry.notes}
              disabled={disabled}
              onChange={e => updateEntry(w.id, w.full_name, { notes: e.target.value })}
              className="w-full bg-slate-800 border border-slate-700 rounded px-2 py-1.5 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:opacity-50"
            />
          </div>
        )
      })}

      <p className="text-xs text-slate-600 px-2 pt-1">
        Ostavite sate prazne za radnike koji danas nisu radili.
      </p>
    </div>
  )
}
