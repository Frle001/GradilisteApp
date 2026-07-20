'use client'

import { type ReportFilter, type ReportFilterOptions } from '@/lib/types/reports'

interface Props {
  filter: ReportFilter
  options: ReportFilterOptions | null
  onChange: (f: ReportFilter) => void
  showWorker?: boolean
  showMaterial?: boolean
  showActivityType?: boolean
  showVtk?: boolean
  loading?: boolean
  hideSelectFilters?: boolean
}

const ACTIVITY_TYPE_LABELS: Record<string, string> = {
  montaza: 'Montaža',
  demontaza: 'Demontaža',
  other: 'Ostalo',
}

const STATUS_LABELS: Record<string, string> = {
  draft: 'Nacrt',
  submitted: 'Predano',
  approved: 'Odobreno',
  rejected: 'Odbijeno',
}

function set(filter: ReportFilter, key: keyof ReportFilter, value: string | boolean | undefined): ReportFilter {
  return { ...filter, [key]: value || undefined, page: 1 }
}

function countActiveFilters(f: ReportFilter): number {
  return [
    f.project_id, f.poslovoda_id, f.worker_id, f.material_id,
    f.activity_type, f.status, f.date_from, f.date_to, f.search,
    f.is_vtk !== undefined ? 'vtk' : undefined,
  ].filter(Boolean).length
}

const inputCls = `
  w-full bg-slate-900 border border-slate-600 rounded-md px-3 py-2
  text-sm text-slate-100 placeholder-slate-500
  focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500
  disabled:opacity-40 disabled:cursor-not-allowed
  transition-colors
`.replace(/\s+/g, ' ').trim()

function FilterField({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <label className="text-xs font-medium text-slate-300 uppercase tracking-wider">{label}</label>
      {children}
    </div>
  )
}

export default function ReportFilters({
  filter,
  options,
  onChange,
  showWorker = false,
  showMaterial = false,
  showActivityType = false,
  showVtk = false,
  loading = false,
  hideSelectFilters = false,
}: Props) {
  const activeCount = countActiveFilters(filter)

  return (
    <div className="bg-slate-800/50 border border-slate-700 rounded-xl overflow-hidden">
      {/* Card header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-slate-700/60 bg-slate-800/60">
        <div className="flex items-center gap-2">
          <svg className="w-4 h-4 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2a1 1 0 01-.293.707L13 13.414V19a1 1 0 01-.553.894l-4 2A1 1 0 017 21v-7.586L3.293 6.707A1 1 0 013 6V4z" />
          </svg>
          <span className="text-sm font-medium text-slate-200">Filteri</span>
          {activeCount > 0 && (
            <span className="inline-flex items-center justify-center w-5 h-5 rounded-full bg-blue-600 text-white text-xs font-bold">
              {activeCount}
            </span>
          )}
        </div>
        {activeCount > 0 && (
          <button
            onClick={() => onChange({ page: 1 })}
            className="flex items-center gap-1.5 text-xs text-slate-400 hover:text-slate-100 bg-slate-700/50 hover:bg-slate-700 px-2.5 py-1 rounded-md transition-colors"
          >
            <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
            Poništi filtere
          </button>
        )}
      </div>

      {/* Filter grid */}
      <div className="p-4">
        <div className="grid grid-cols-2 gap-x-4 gap-y-3 sm:grid-cols-3 lg:grid-cols-4">
          {!hideSelectFilters && (
            <>
              <FilterField label="Gradilište">
                <select
                  disabled={loading || !options}
                  className={inputCls}
                  value={filter.project_id ?? ''}
                  onChange={e => onChange(set(filter, 'project_id', e.target.value))}
                >
                  <option value="">Sva gradilišta</option>
                  {options?.projects.map(p => (
                    <option key={p.id} value={p.id}>{p.name}</option>
                  ))}
                </select>
              </FilterField>

              <FilterField label="Poslovođa">
                <select
                  disabled={loading || !options}
                  className={inputCls}
                  value={filter.poslovoda_id ?? ''}
                  onChange={e => onChange(set(filter, 'poslovoda_id', e.target.value))}
                >
                  <option value="">Svi poslovođe</option>
                  {options?.poslovode.map(p => (
                    <option key={p.id} value={p.id}>{p.full_name}</option>
                  ))}
                </select>
              </FilterField>

              {showWorker && (
                <FilterField label="Radnik">
                  <select
                    disabled={loading || !options}
                    className={inputCls}
                    value={filter.worker_id ?? ''}
                    onChange={e => onChange(set(filter, 'worker_id', e.target.value))}
                  >
                    <option value="">Svi radnici</option>
                    {options?.workers.map(w => (
                      <option key={w.id} value={w.id}>{w.full_name}</option>
                    ))}
                  </select>
                </FilterField>
              )}

              {showMaterial && (
                <FilterField label="Materijal">
                  <select
                    disabled={loading || !options}
                    className={inputCls}
                    value={filter.material_id ?? ''}
                    onChange={e => onChange(set(filter, 'material_id', e.target.value))}
                  >
                    <option value="">Svi materijali</option>
                    {options?.materials
                      .filter(m => !filter.project_id || m.project_id === filter.project_id)
                      .map(m => (
                        <option key={m.id} value={m.id}>{m.material_name}</option>
                      ))}
                  </select>
                </FilterField>
              )}

              {showActivityType && (
                <FilterField label="Aktivnost">
                  <select
                    disabled={loading || !options}
                    className={inputCls}
                    value={filter.activity_type ?? ''}
                    onChange={e => onChange(set(filter, 'activity_type', e.target.value))}
                  >
                    <option value="">Sve aktivnosti</option>
                    {options?.activity_types.map(t => (
                      <option key={t} value={t}>{ACTIVITY_TYPE_LABELS[t] ?? t}</option>
                    ))}
                  </select>
                </FilterField>
              )}

              <FilterField label="Status">
                <select
                  disabled={loading || !options}
                  className={inputCls}
                  value={filter.status ?? ''}
                  onChange={e => onChange(set(filter, 'status', e.target.value))}
                >
                  <option value="">Svi statusi</option>
                  {(options?.statuses ?? ['draft', 'submitted', 'approved', 'rejected']).map(s => (
                    <option key={s} value={s}>{STATUS_LABELS[s] ?? s}</option>
                  ))}
                </select>
              </FilterField>
            </>
          )}

          <FilterField label="Datum od">
            <input
              type="date"
              disabled={loading}
              className={inputCls}
              value={filter.date_from ?? ''}
              onChange={e => onChange(set(filter, 'date_from', e.target.value))}
            />
          </FilterField>

          <FilterField label="Datum do">
            <input
              type="date"
              disabled={loading}
              className={inputCls}
              value={filter.date_to ?? ''}
              onChange={e => onChange(set(filter, 'date_to', e.target.value))}
            />
          </FilterField>

          {showVtk && (
            <FilterField label="VTK">
              <select
                disabled={loading}
                className={inputCls}
                value={filter.is_vtk === undefined ? '' : String(filter.is_vtk)}
                onChange={e => {
                  const v = e.target.value
                  onChange({ ...filter, is_vtk: v === '' ? undefined : v === 'true', page: 1 })
                }}
              >
                <option value="">Sve</option>
                <option value="true">Samo VTK</option>
                <option value="false">Nije VTK</option>
              </select>
            </FilterField>
          )}

          <FilterField label="Pretraga">
            <input
              type="text"
              placeholder="Gradilište, poslovođa…"
              disabled={loading}
              className={inputCls}
              value={filter.search ?? ''}
              onChange={e => onChange(set(filter, 'search', e.target.value))}
            />
          </FilterField>
        </div>
      </div>
    </div>
  )
}
