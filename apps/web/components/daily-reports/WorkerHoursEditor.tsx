'use client'

import { type FormDataWorker, type WorkerHoursEntry } from '@/lib/types/daily-reports'

interface Props {
  workers: FormDataWorker[]
  entries: WorkerHoursEntry[]
  onChange: (entries: WorkerHoursEntry[]) => void
  disabled?: boolean
}

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
    <div className="space-y-1">
      <div className="grid grid-cols-[1fr_100px_1fr] gap-x-3 px-2 pb-1 text-xs font-medium text-slate-500 uppercase tracking-wide">
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
            className="grid grid-cols-[1fr_100px_1fr] gap-x-3 items-center px-2 py-1.5 rounded hover:bg-slate-800/40"
          >
            <span className="text-sm text-slate-200 truncate">{w.full_name}</span>

            <div>
              <input
                type="number"
                min={0}
                max={24}
                step={0.5}
                placeholder="—"
                value={entry.hours}
                disabled={disabled}
                onChange={e => updateEntry(w.id, w.full_name, { hours: e.target.value })}
                className={`w-full bg-slate-800 border rounded px-2 py-1 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-blue-500 ${
                  hoursError ? 'border-red-500' : 'border-slate-700'
                } disabled:opacity-50`}
              />
              {hoursError && <p className="text-xs text-red-400 mt-0.5">{hoursError}</p>}
            </div>

            <input
              type="text"
              placeholder="Opcionalna napomena…"
              value={entry.notes}
              disabled={disabled}
              onChange={e => updateEntry(w.id, w.full_name, { notes: e.target.value })}
              className="w-full bg-slate-800 border border-slate-700 rounded px-2 py-1 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:opacity-50"
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
