'use client'

import { useState, useMemo } from 'react'
import { Search } from 'lucide-react'
import { type Employee, ROLE_LABELS } from '@/lib/types/employees'
import { type EmployeeForDate } from '@/lib/types/schedule'

export default function EmployeePicker({
  employees,
  availability,
  selected,
  onChange,
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
    return employees.filter(e =>
      `${e.first_name} ${e.last_name}`.toLowerCase().includes(q)
    )
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
        <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-slate-500" />
        <input
          type="text"
          placeholder="Pretraži zaposlenike..."
          value={search}
          onChange={e => setSearch(e.target.value)}
          className="w-full pl-8 pr-3 py-1.5 rounded-lg bg-slate-800 border border-slate-700 text-white text-sm focus:outline-none focus:border-blue-500"
        />
      </div>

      <div className="rounded-lg border border-slate-700 bg-slate-800 overflow-y-auto max-h-52 divide-y divide-slate-700/50">
        {filtered.length === 0 && (
          <div className="px-3 py-4 text-xs text-slate-500 text-center">Nema rezultata.</div>
        )}
        {filtered.map(emp => {
          const info = availability.get(emp.id)
          const isAssigned = !!info?.assigned
          const isChecked = selected.has(emp.id)
          return (
            <label
              key={emp.id}
              className={`flex items-start gap-3 px-3 py-2 transition-colors cursor-pointer ${
                isAssigned
                  ? 'bg-red-950/40 border-l-2 border-red-800/60 hover:bg-red-950/60'
                  : 'hover:bg-slate-700/50'
              }`}
            >
              <input
                type="checkbox"
                checked={isChecked}
                onChange={() => toggle(emp.id)}
                className="w-4 h-4 rounded text-blue-500 bg-slate-700 border-slate-600 flex-shrink-0 mt-0.5"
              />
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 flex-wrap">
                  <span className={`text-sm flex-1 truncate ${isAssigned ? 'text-red-300' : 'text-white'}`}>
                    {emp.first_name} {emp.last_name}
                  </span>
                  <span className="text-[11px] text-slate-500 flex-shrink-0">{ROLE_LABELS[emp.role] ?? emp.role}</span>
                  {isAssigned && (
                    <span className="flex-shrink-0 text-[10px] bg-red-900/60 text-red-300 border border-red-700/50 px-1.5 py-0.5 rounded-full font-semibold">
                      Dodijeljen
                    </span>
                  )}
                </div>
                {isAssigned && (
                  <div className="mt-0.5">
                    <p className="text-[11px] text-red-400/80">Već ima smjenu za ovaj datum</p>
                    {info?.project_name && (
                      <p className="text-[11px] text-red-400/60">Raspoređen na: {info.project_name}</p>
                    )}
                  </div>
                )}
              </div>
            </label>
          )
        })}
      </div>
    </div>
  )
}
