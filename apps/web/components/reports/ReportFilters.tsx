'use client'

import { useState, useEffect } from 'react'
import type { ReactNode } from 'react'
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

function countAdvancedFilters(f: ReportFilter): number {
  return [
    f.poslovoda_id, f.material_id, f.activity_type, f.status,
    f.is_vtk !== undefined ? 'vtk' : undefined,
  ].filter(Boolean).length
}

const inputCls = [
  'w-full bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2',
  'text-sm text-gi-text',
  'focus:outline-none focus:border-gi-yellow',
  'disabled:opacity-40 disabled:cursor-not-allowed',
  'transition-colors',
].join(' ')

function FilterField({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <label className="text-xs font-medium text-gi-text-muted uppercase tracking-wider">{label}</label>
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
  const [showAdvanced, setShowAdvanced] = useState(false)

  const activeCount = countActiveFilters(filter)
  const advancedCount = countAdvancedFilters(filter)
  const hasAdvancedFields = showMaterial || showActivityType || showVtk

  // Auto-expand advanced section when an advanced filter is active (e.g. from URL state)
  useEffect(() => {
    if (advancedCount > 0) setShowAdvanced(true)
  }, [advancedCount])

  // ── Compact toolbar: dates + search only (Dnevnik) ────────────────────────
  if (hideSelectFilters) {
    return (
      <div className="flex flex-col sm:flex-row sm:items-end gap-2 sm:gap-3">
        <div className="sm:w-[152px] shrink-0">
          <FilterField label="Datum od">
            <input
              type="date" disabled={loading} className={inputCls}
              value={filter.date_from ?? ''}
              onChange={e => onChange(set(filter, 'date_from', e.target.value))}
            />
          </FilterField>
        </div>
        <div className="sm:w-[152px] shrink-0">
          <FilterField label="Datum do">
            <input
              type="date" disabled={loading} className={inputCls}
              value={filter.date_to ?? ''}
              onChange={e => onChange(set(filter, 'date_to', e.target.value))}
            />
          </FilterField>
        </div>
        <div className="flex-1 min-w-0">
          <FilterField label="Pretraga">
            <input
              type="text" placeholder="Gradilište, poslovođa…" disabled={loading}
              className={inputCls}
              value={filter.search ?? ''}
              onChange={e => onChange(set(filter, 'search', e.target.value))}
            />
          </FilterField>
        </div>
        {activeCount > 0 && (
          <div className="sm:self-end">
            <button
              onClick={() => onChange({ page: 1 })}
              className="flex items-center gap-1.5 h-[38px] text-xs text-gi-text-muted hover:text-gi-text bg-gi-card border border-gi-border hover:bg-gi-card-elevated px-3 rounded-lg transition-colors whitespace-nowrap"
            >
              <svg className="w-3 h-3 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
              Poništi
              <span className="inline-flex items-center justify-center w-4 h-4 rounded-full bg-gi-yellow text-gi-navy text-[10px] font-bold leading-none">
                {activeCount}
              </span>
            </button>
          </div>
        )}
      </div>
    )
  }

  // ── Card: primary visible + collapsible advanced (Knjiga) ─────────────────
  return (
    <div className="bg-gi-card border border-gi-border rounded-xl overflow-hidden">
      {/* Primary row: project + dates + search */}
      <div className="p-4">
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
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

          <FilterField label="Datum od">
            <input
              type="date" disabled={loading} className={inputCls}
              value={filter.date_from ?? ''}
              onChange={e => onChange(set(filter, 'date_from', e.target.value))}
            />
          </FilterField>

          <FilterField label="Datum do">
            <input
              type="date" disabled={loading} className={inputCls}
              value={filter.date_to ?? ''}
              onChange={e => onChange(set(filter, 'date_to', e.target.value))}
            />
          </FilterField>

          <FilterField label="Pretraga">
            <input
              type="text"
              placeholder={showMaterial ? 'Gradilište, materijal…' : 'Gradilište, poslovođa…'}
              disabled={loading} className={inputCls}
              value={filter.search ?? ''}
              onChange={e => onChange(set(filter, 'search', e.target.value))}
            />
          </FilterField>
        </div>
      </div>

      {/* Footer: advanced toggle + reset */}
      <div className="border-t border-gi-border px-4 py-2.5 flex items-center justify-between gap-3">
        {hasAdvancedFields ? (
          <button
            onClick={() => setShowAdvanced(v => !v)}
            className="flex items-center gap-2 text-xs text-gi-text-muted hover:text-gi-text transition-colors"
          >
            <svg
              className={`w-3.5 h-3.5 transition-transform duration-150 ${showAdvanced ? 'rotate-90' : ''}`}
              fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}
            >
              <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
            </svg>
            Napredni filteri
            {advancedCount > 0 && (
              <span className="inline-flex items-center justify-center w-4 h-4 rounded-full bg-gi-yellow text-gi-navy text-[10px] font-bold leading-none">
                {advancedCount}
              </span>
            )}
          </button>
        ) : (
          <span />
        )}
        {activeCount > 0 && (
          <button
            onClick={() => onChange({ page: 1 })}
            className="flex items-center gap-1.5 text-xs text-gi-text-muted hover:text-gi-text bg-gi-card-elevated hover:bg-gi-border px-2.5 py-1 rounded-lg transition-colors"
          >
            <svg className="w-3 h-3 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
            Poništi filtere
          </button>
        )}
      </div>

      {/* Advanced section: poslovođa, materijal, aktivnost, status, VTR */}
      {hasAdvancedFields && showAdvanced && (
        <div className="border-t border-gi-border p-4 bg-gi-workspace/40">
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
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

            {showVtk && (
              <FilterField label="VTR">
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
                  <option value="true">Samo VTR</option>
                  <option value="false">Nije VTR</option>
                </select>
              </FilterField>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
