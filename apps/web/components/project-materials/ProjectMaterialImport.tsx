'use client'

import { useRef, useState } from 'react'
import apiClient from '@/lib/api-client'
import {
  type ImportAnalyzeResponse,
  type ImportMapping,
  type ImportField,
  type WizardPreviewResponse,
  type WizardPreviewRow,
  type WizardConfirmResponse,
  type ConfirmRow,
  FIELD_LABELS,
} from '@/lib/types/project-materials'

interface Props {
  projectId: string
  onConfirmed: () => void
}

type Step = 'idle' | 'uploading' | 'mapping' | 'previewing' | 'preview' | 'confirming' | 'done'
type ViewMode = 'compact' | 'table'
type FilterChip = 'all' | 'valid' | 'invalid' | 'included' | 'skipped'

// EditablePreviewRow extends the parsed preview row with an `include` flag
// so directors can opt-in or opt-out individual rows before confirming.
type EditablePreviewRow = WizardPreviewRow & { include: boolean }

const FIELD_OPTIONS: ImportField[] = [
  'material_name', 'quantity', 'unit',
  'unit_price', 'total_price', 'category', 'note', 'position', 'ignore',
]

const CONFIDENCE_BADGE: Record<string, string> = {
  high:   'bg-green-900/50 text-green-300 border border-green-800',
  medium: 'bg-amber-900/50 text-amber-300 border border-amber-800',
  low:    'bg-slate-800 text-slate-400 border border-slate-700',
  none:   '',
}

function cx(...parts: (string | false | null | undefined)[]): string {
  return parts.filter(Boolean).join(' ')
}

function validateRow(row: EditablePreviewRow): string[] {
  const errs: string[] = []
  if (!row.material_name.trim()) errs.push('Naziv je obavezan')
  if (!(row.quantity > 0)) errs.push('Količina mora biti > 0')
  if (!row.unit.trim()) errs.push('JM je obavezna')
  if (row.unit_price != null && row.unit_price < 0) errs.push('Jed. cijena ne može biti negativna')
  if (row.total_price != null && row.total_price < 0) errs.push('Ukupna cijena ne može biti negativna')
  return errs
}

export default function ProjectMaterialImport({ projectId, onConfirmed }: Props) {
  const fileRef = useRef<HTMLInputElement>(null)
  const rowCounterRef = useRef(0)

  const [step, setStep] = useState<Step>('idle')
  const [error, setError] = useState<string | null>(null)

  // Step 1
  const [analyzeResp, setAnalyzeResp] = useState<ImportAnalyzeResponse | null>(null)

  // Step 2
  const [mapping, setMapping] = useState<ImportMapping>({})

  // Step 3 — editable rows replace direct use of previewResp.rows
  const [previewResp, setPreviewResp] = useState<WizardPreviewResponse | null>(null)
  const [editableRows, setEditableRows] = useState<EditablePreviewRow[]>([])

  // Step 4
  const [confirmResult, setConfirmResult] = useState<WizardConfirmResponse | null>(null)

  // Preview UI state
  const [viewMode, setViewMode] = useState<ViewMode>('compact')
  const [expandedRows, setExpandedRows] = useState<Set<number>>(new Set())
  const [filterChip, setFilterChip] = useState<FilterChip>('all')
  const [searchQuery, setSearchQuery] = useState('')

  // ── Row helpers ──────────────────────────────────────────────────────────────

  function updateRow(rowNum: number, updates: Partial<EditablePreviewRow>) {
    setEditableRows(prev => prev.map(r => {
      if (r.row_number !== rowNum) return r
      const merged = { ...r, ...updates }
      // Auto-calc total_price when qty or unit_price changes and total was previously null
      if (('quantity' in updates || 'unit_price' in updates) && r.total_price == null) {
        if (merged.unit_price != null && merged.unit_price >= 0 && merged.quantity > 0) {
          merged.total_price = Math.round(merged.quantity * merged.unit_price * 100) / 100
        }
      }
      return merged
    }))
  }

  function deleteRow(rowNum: number) {
    setEditableRows(prev => prev.filter(r => r.row_number !== rowNum))
  }

  function toggleExpand(rowNum: number) {
    setExpandedRows(prev => {
      const next = new Set(prev)
      if (next.has(rowNum)) next.delete(rowNum)
      else next.add(rowNum)
      return next
    })
  }

  function addRow() {
    rowCounterRef.current += 1
    const id = -(rowCounterRef.current)
    setEditableRows(prev => [...prev, {
      row_number: id,
      source_row_numbers: [],
      row_type: 'item',
      position: null,
      material_name: '',
      quantity: 1,
      unit: 'kom',
      unit_price: null,
      total_price: null,
      category: null,
      note: null,
      warnings: [],
      errors: [],
      status: 'valid',
      include: true,
    }])
  }

  // ── Step 1: upload & analyze ─────────────────────────────────────────────────

  async function handleUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    setError(null)
    setStep('uploading')

    const form = new FormData()
    form.append('file', file)

    try {
      const res = await apiClient.post(
        `/projects/${projectId}/materials/import/analyze`,
        form,
        { headers: { 'Content-Type': 'multipart/form-data' } },
      )
      const data = res.data as ImportAnalyzeResponse

      const initial: ImportMapping = {}
      for (const h of data.headers) {
        if (h.confidence >= 0.85 && h.suggested_field) {
          initial[h.original] = h.suggested_field as ImportField
        } else {
          initial[h.original] = 'ignore'
        }
      }
      setAnalyzeResp(data)
      setMapping(initial)
      setStep('mapping')
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } }
      setError(err?.response?.data?.error ?? 'Greška pri učitavanju datoteke.')
      setStep('idle')
    }

    if (fileRef.current) fileRef.current.value = ''
  }

  // ── Step 2: send mapping, get preview ────────────────────────────────────────

  async function handlePreview() {
    if (!analyzeResp) return
    setError(null)
    setStep('previewing')

    try {
      const res = await apiClient.post(`/projects/${projectId}/materials/import/preview`, {
        import_job_id: analyzeResp.import_job_id,
        mapping,
      })
      const data = res.data as WizardPreviewResponse
      setPreviewResp(data)
      rowCounterRef.current = 0
      setEditableRows(
        (data.rows ?? []).map(row => ({
          ...row,
          include: row.status === 'valid' || row.row_type === 'item',
        }))
      )
      setFilterChip('all')
      setSearchQuery('')
      setExpandedRows(new Set())
      setStep('preview')
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } }
      setError(err?.response?.data?.error ?? 'Greška pri generiranju pregleda.')
      setStep('mapping')
    }
  }

  // ── Step 3: confirm with edited rows ─────────────────────────────────────────

  async function handleConfirm() {
    if (!analyzeResp) return
    setError(null)
    setStep('confirming')

    try {
      const rows: ConfirmRow[] = editableRows.map(r => ({
        row_number: r.row_number,
        position: r.position ?? null,
        material_name: r.material_name,
        quantity: r.quantity,
        unit: r.unit,
        unit_price: r.unit_price ?? null,
        total_price: r.total_price ?? null,
        category: r.category ?? null,
        note: r.note ?? null,
        include: r.include,
      }))

      const res = await apiClient.post(`/projects/${projectId}/materials/import/confirm`, {
        import_job_id: analyzeResp.import_job_id,
        rows,
      })
      setConfirmResult(res.data as WizardConfirmResponse)
      setStep('done')
      onConfirmed()
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } }
      setError(err?.response?.data?.error ?? 'Greška pri potvrdi uvoza.')
      setStep('preview')
    }
  }

  function handleReset() {
    setStep('idle')
    setAnalyzeResp(null)
    setMapping({})
    setPreviewResp(null)
    setEditableRows([])
    rowCounterRef.current = 0
    setConfirmResult(null)
    setError(null)
  }

  function handleBackToMapping() {
    setPreviewResp(null)
    setEditableRows([])
    setError(null)
    setStep('mapping')
  }

  // ── Render ────────────────────────────────────────────────────────────────────

  return (
    <div className="space-y-4">

      {/* Error banner */}
      {error && (
        <div className="bg-red-950 border border-red-800 rounded-lg px-3 py-2">
          <p className="text-red-300 text-sm">{error}</p>
        </div>
      )}

      {/* Step 1: file picker */}
      {step === 'idle' && (
        <div>
          <label className="cursor-pointer inline-flex items-center gap-2 px-4 py-2 bg-slate-800 border border-slate-700 text-slate-300 text-sm rounded-lg hover:bg-slate-700 transition">
            <span>Odaberi Excel datoteku</span>
            <input
              ref={fileRef}
              type="file"
              accept=".xlsx,.xls"
              className="hidden"
              onChange={handleUpload}
            />
          </label>
          <p className="text-slate-600 text-xs mt-1">Prihvaćeni formati: .xlsx, .xls</p>
        </div>
      )}

      {step === 'uploading' && (
        <p className="text-slate-400 text-sm animate-pulse">Čitanje datoteke i prepoznavanje stupaca...</p>
      )}

      {/* Step 2: column mapping */}
      {(step === 'mapping' || step === 'previewing') && analyzeResp && (() => {
        const REQUIRED: ImportField[] = ['material_name', 'quantity', 'unit']
        const mappedValues = Object.values(mapping)
        const missingRequired = REQUIRED.filter(f => !mappedValues.includes(f))

        const fieldCounts = mappedValues.reduce<Record<string, number>>((acc, f) => {
          if (f !== 'ignore') acc[f] = (acc[f] ?? 0) + 1
          return acc
        }, {})
        const duplicatedFields = Object.entries(fieldCounts)
          .filter(([, n]) => n > 1)
          .map(([f]) => f)

        const canPreview = missingRequired.length === 0 && duplicatedFields.length === 0

        return (
          <div className="space-y-4">
            <div className="flex items-start justify-between gap-3">
              <div>
                <p className="text-white text-sm font-medium">{analyzeResp.sheet_name}</p>
                <p className="text-slate-400 text-xs">
                  {analyzeResp.headers.length} stupaca · Zaglavlje na retku {analyzeResp.header_row_number} · Dodjeli svaki stupac odgovarajućem polju
                </p>
              </div>
              <button onClick={handleReset} className="text-slate-500 hover:text-slate-300 text-xs transition shrink-0">
                Odustani
              </button>
            </div>

            {/* Sample rows preview */}
            {analyzeResp.sample_rows.length > 0 && (
              <div className="overflow-x-auto border border-slate-800 rounded-lg">
                <table className="w-full text-xs">
                  <thead className="bg-slate-900/80">
                    <tr className="border-b border-slate-800 text-slate-500 uppercase tracking-wide">
                      <th className="text-left px-3 py-1.5">#</th>
                      {analyzeResp.headers.map((h) => (
                        <th key={h.index} className="text-left px-3 py-1.5 whitespace-nowrap">{h.original}</th>
                      ))}
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-800/50">
                    {analyzeResp.sample_rows.map((row) => (
                      <tr key={row.row_number}>
                        <td className="px-3 py-1.5 text-slate-600">{row.row_number}</td>
                        {row.values.map((v, i) => (
                          <td key={i} className="px-3 py-1.5 text-slate-300 max-w-[120px] truncate">{v || '—'}</td>
                        ))}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {/* Mapping dropdowns */}
            <div className="space-y-2">
              <p className="text-slate-500 text-xs uppercase tracking-wide">Dodjela stupaca</p>
              {analyzeResp.headers.map((h) => {
                const conf = h.confidence_label
                const isDup = mapping[h.original] !== 'ignore' && (fieldCounts[mapping[h.original]] ?? 0) > 1
                const reasonTitle = h.reasons?.join(' · ') ?? ''
                return (
                  <div key={h.index} className="flex items-center gap-3 flex-wrap sm:flex-nowrap">
                    <div className="flex items-center gap-2 min-w-0 flex-1">
                      <span className="text-slate-300 text-sm truncate" title={h.original}>{h.original}</span>
                      {conf !== 'none' && (
                        <span
                          className={`text-xs px-1.5 py-0.5 rounded shrink-0 cursor-help ${CONFIDENCE_BADGE[conf]}`}
                          title={reasonTitle || undefined}
                        >
                          {conf === 'high' ? 'visoka' : conf === 'medium' ? 'srednja' : 'niska'}
                        </span>
                      )}
                    </div>
                    <span className="text-slate-600 shrink-0">→</span>
                    <select
                      value={mapping[h.original] ?? 'ignore'}
                      onChange={(e) => setMapping((m) => ({ ...m, [h.original]: e.target.value as ImportField }))}
                      className={`px-2.5 py-1.5 bg-slate-800 rounded-lg text-white text-sm focus:outline-none focus:ring-1 focus:ring-slate-500 shrink-0 border ${isDup ? 'border-amber-600' : 'border-slate-700'}`}
                    >
                      {FIELD_OPTIONS.map((f) => (
                        <option key={f} value={f}>{FIELD_LABELS[f]}</option>
                      ))}
                    </select>
                  </div>
                )
              })}
            </div>

            {/* Validation warnings */}
            {(missingRequired.length > 0 || duplicatedFields.length > 0) && (
              <div className="space-y-1">
                {missingRequired.length > 0 && (
                  <p className="text-amber-400 text-xs">
                    Obavezna polja nisu dodijeljena: {missingRequired.map(f => FIELD_LABELS[f as ImportField]).join(', ')}
                  </p>
                )}
                {duplicatedFields.length > 0 && (
                  <p className="text-amber-400 text-xs">
                    Isto polje dodijeljeno više puta: {duplicatedFields.map(f => FIELD_LABELS[f as ImportField]).join(', ')}
                  </p>
                )}
              </div>
            )}

            <button
              onClick={handlePreview}
              disabled={!canPreview || step === 'previewing'}
              title={!canPreview ? 'Dodjeli sva obavezna polja prije pregleda' : undefined}
              className="px-4 py-2 bg-white text-slate-900 font-semibold rounded-lg text-sm hover:bg-slate-100 transition disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {step === 'previewing' ? 'Generiranje pregleda...' : 'Pregled podataka →'}
            </button>
          </div>
        )
      })()}

      {/* Step 3: preview with compact / table modes */}
      {(step === 'preview' || step === 'confirming') && previewResp && (() => {
        const summary = previewResp.summary ?? {
          total_rows: 0, valid_rows: 0, invalid_rows: 0,
          total_scanned: 0, skipped_categories: 0, skipped_continuations: 0, skipped_subtotals: 0,
        }
        const includedRows = editableRows.filter(r => r.include)
        const skippedRows = editableRows.filter(r => !r.include)

        // Precompute validation once per render
        const rowErrors = new Map<number, string[]>(
          editableRows.map(r => [r.row_number, validateRow(r)])
        )

        const invalidIncluded = includedRows.filter(r => (rowErrors.get(r.row_number) ?? []).length > 0)
        const canConfirm = includedRows.length > 0 && invalidIncluded.length === 0 && step !== 'confirming'

        const estimatedTotal = includedRows.reduce((sum, r) => sum + (r.total_price ?? 0), 0)
        const hasAnyTotal = includedRows.some(r => r.total_price != null)
        const hasPosition = editableRows.some(r => r.position != null && r.position !== '')

        const skipParts: string[] = []
        if (summary.skipped_categories > 0) skipParts.push(`${summary.skipped_categories} kategorija`)
        if (summary.skipped_continuations > 0) skipParts.push(`${summary.skipped_continuations} nastavaka`)
        if (summary.skipped_subtotals > 0) skipParts.push(`${summary.skipped_subtotals} ukupno-redaka`)

        const inp = 'w-full rounded border border-slate-700/50 bg-slate-900/60 px-2 py-1.5 text-sm text-slate-100 placeholder:text-slate-600 focus:border-blue-500/70 focus:bg-slate-900 focus:outline-none transition-colors disabled:cursor-not-allowed disabled:opacity-50'
        const inpErr = 'border-red-500/60 focus:border-red-500/60'
        const colSpanCount = hasPosition ? 11 : 10

        const chipCounts = {
          all: editableRows.length,
          included: includedRows.length,
          skipped: skippedRows.length,
          valid: editableRows.filter(r => (rowErrors.get(r.row_number) ?? []).length === 0).length,
          invalid: editableRows.filter(r => (rowErrors.get(r.row_number) ?? []).length > 0).length,
        }
        const chipLabels: Record<FilterChip, string> = {
          all: `Sve (${chipCounts.all})`,
          included: `Uključeno (${chipCounts.included})`,
          skipped: `Preskočeno (${chipCounts.skipped})`,
          valid: `Ispravno (${chipCounts.valid})`,
          invalid: `Greške (${chipCounts.invalid})`,
        }

        const filteredRows = editableRows.filter(row => {
          const live = rowErrors.get(row.row_number) ?? []
          if (filterChip === 'valid' && live.length > 0) return false
          if (filterChip === 'invalid' && live.length === 0) return false
          if (filterChip === 'included' && !row.include) return false
          if (filterChip === 'skipped' && row.include) return false
          if (searchQuery.trim()) {
            const q = searchQuery.toLowerCase()
            if (!row.material_name.toLowerCase().includes(q) && !(row.category?.toLowerCase().includes(q) ?? false)) return false
          }
          return true
        })

        const noResultsMsg = (searchQuery || filterChip !== 'all')
          ? 'Nema rezultata za odabrani filter.'
          : 'Nema redaka. Dodaj stavku ručno.'

        return (
          <div className="space-y-3">
            {/* Header */}
            <div className="flex items-start justify-between gap-3">
              <div>
                <p className="text-white text-sm font-medium">Pregled i uređivanje uvoza</p>
                <p className="text-slate-400 text-xs">
                  {editableRows.length} redaka ·{' '}
                  <span className="text-green-400">{includedRows.length} za uvoz</span>
                  {skippedRows.length > 0 && (
                    <> · <span className="text-slate-500">{skippedRows.length} preskočenih</span></>
                  )}
                  {invalidIncluded.length > 0 && (
                    <> · <span className="text-red-400">{invalidIncluded.length} s greškama</span></>
                  )}
                  {hasAnyTotal && (
                    <> · Procjena: {estimatedTotal.toLocaleString('hr-HR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} EUR</>
                  )}
                </p>
                {skipParts.length > 0 && (
                  <p className="text-slate-600 text-xs mt-0.5">Preskočeno iz Excela: {skipParts.join(' · ')}</p>
                )}
              </div>
              <button
                onClick={handleBackToMapping}
                disabled={step === 'confirming'}
                className="text-slate-500 hover:text-slate-300 text-xs transition disabled:opacity-30 shrink-0"
              >
                ← Nazad na dodjelu
              </button>
            </div>

            {/* Toolbar: view mode + filter chips + search */}
            <div className="flex flex-wrap items-center gap-2">
              {/* View mode toggle */}
              <div className="flex items-center gap-0.5 bg-slate-900 border border-slate-800 rounded-lg p-0.5 shrink-0">
                {(['compact', 'table'] as ViewMode[]).map(mode => (
                  <button
                    key={mode}
                    onClick={() => setViewMode(mode)}
                    className={cx(
                      'px-3 py-1 rounded text-xs transition',
                      viewMode === mode ? 'bg-slate-700 text-white' : 'text-slate-400 hover:text-slate-300',
                    )}
                  >
                    {mode === 'compact' ? 'Kompaktno' : 'Tablica'}
                  </button>
                ))}
              </div>

              {/* Filter chips */}
              {(['all', 'included', 'skipped', 'valid', 'invalid'] as FilterChip[]).map(chip => (
                <button
                  key={chip}
                  onClick={() => setFilterChip(chip)}
                  className={cx(
                    'px-2.5 py-1 rounded-full text-xs transition whitespace-nowrap',
                    filterChip === chip
                      ? 'bg-slate-700 text-white'
                      : 'bg-slate-900 text-slate-400 hover:text-slate-300 border border-slate-800',
                  )}
                >
                  {chipLabels[chip]}
                </button>
              ))}

              {/* Search */}
              <input
                value={searchQuery}
                onChange={ev => setSearchQuery(ev.target.value)}
                placeholder="Pretraži naziv ili kategoriju..."
                className="flex-1 min-w-[160px] px-3 py-1 rounded-lg bg-slate-900 border border-slate-800 text-xs text-slate-200 placeholder:text-slate-600 focus:outline-none focus:border-slate-700 transition-colors"
              />
            </div>

            {/* ── Compact mode ─────────────────────────────────────────────── */}
            {viewMode === 'compact' && (
              <div className="space-y-1 max-h-[65vh] overflow-y-auto pr-0.5">
                {filteredRows.length === 0 ? (
                  <p className="text-slate-500 text-sm py-6 text-center">{noResultsMsg}</p>
                ) : filteredRows.map(row => {
                  const live = rowErrors.get(row.row_number) ?? []
                  const isIncluded = row.include
                  const hasLiveErr = isIncluded && live.length > 0
                  const isExpanded = expandedRows.has(row.row_number)
                  const warnings = row.warnings ?? []
                  const sourceRows = row.source_row_numbers?.filter(n => n > 0) ?? []

                  return (
                    <div
                      key={row.row_number}
                      className={cx(
                        'rounded-lg border transition-colors',
                        !isIncluded && 'opacity-50',
                        hasLiveErr
                          ? 'border-red-800/60 bg-red-950/15'
                          : isExpanded
                            ? 'border-slate-700 bg-slate-900/50'
                            : 'border-slate-800 bg-slate-900/20 hover:border-slate-700',
                      )}
                    >
                      {/* Collapsed header — click anywhere to expand */}
                      <div
                        className="flex items-center gap-2.5 px-3 py-2 cursor-pointer select-none"
                        onClick={() => toggleExpand(row.row_number)}
                      >
                        <input
                          type="checkbox"
                          checked={row.include}
                          onChange={ev => updateRow(row.row_number, { include: ev.target.checked })}
                          className="h-4 w-4 accent-green-500 cursor-pointer shrink-0"
                          onClick={ev => ev.stopPropagation()}
                        />

                        <span className={cx(
                          'flex-1 text-sm font-medium truncate min-w-0',
                          !isIncluded ? 'text-slate-500 line-through' : hasLiveErr ? 'text-red-300' : 'text-slate-100',
                        )}>
                          {row.material_name || <span className="italic font-normal text-red-400">Naziv obavezan</span>}
                        </span>

                        <span className="text-xs text-slate-400 whitespace-nowrap shrink-0">
                          {row.quantity} {row.unit}
                        </span>

                        {row.total_price != null && (
                          <span className="text-xs text-slate-400 whitespace-nowrap shrink-0">
                            {row.total_price.toLocaleString('hr-HR', { minimumFractionDigits: 2 })} €
                          </span>
                        )}

                        {row.category && (
                          <span
                            className="hidden sm:inline-block text-[11px] px-2 py-0.5 rounded-full bg-slate-800 text-slate-400 whitespace-nowrap shrink-0 max-w-[120px] truncate"
                            title={row.category}
                          >
                            {row.category}
                          </span>
                        )}

                        {hasLiveErr && (
                          <span className="text-[11px] text-red-400 whitespace-nowrap shrink-0">✕ Greška</span>
                        )}
                        {!hasLiveErr && warnings.length > 0 && (
                          <span className="text-[11px] text-amber-400 whitespace-nowrap shrink-0">⚠</span>
                        )}

                        <span className="text-slate-600 text-[11px] shrink-0 ml-1">
                          {isExpanded ? '▲' : '▼'}
                        </span>
                      </div>

                      {/* Expanded editing area */}
                      {isExpanded && (
                        <div className="border-t border-slate-800 px-3 py-3 space-y-3">
                          {/* Read-only metadata */}
                          {(sourceRows.length > 0 || row.row_number < 0) && (
                            <div className="flex flex-wrap gap-4 text-xs text-slate-600">
                              {sourceRows.length > 0 && (
                                <span>Izvorni reci: {sourceRows.join(', ')}</span>
                              )}
                              {row.row_number < 0 && (
                                <span className="text-slate-500 italic">Ručno dodana stavka</span>
                              )}
                            </div>
                          )}

                          {/* Editable fields */}
                          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-2">
                            {hasPosition && (
                              <label className="flex flex-col gap-1">
                                <span className="text-[11px] text-slate-500 uppercase tracking-wide">Pozicija</span>
                                <input
                                  value={row.position ?? ''}
                                  onChange={ev => updateRow(row.row_number, { position: ev.target.value || null })}
                                  disabled={!isIncluded || step === 'confirming'}
                                  className={inp}
                                  placeholder="—"
                                />
                              </label>
                            )}
                            <label className="flex flex-col gap-1 col-span-2">
                              <span className="text-[11px] text-slate-500 uppercase tracking-wide">Naziv materijala *</span>
                              <input
                                value={row.material_name}
                                onChange={ev => updateRow(row.row_number, { material_name: ev.target.value })}
                                disabled={!isIncluded || step === 'confirming'}
                                className={cx(inp, isIncluded && !row.material_name.trim() && inpErr)}
                                placeholder="Naziv materijala..."
                              />
                            </label>
                            <label className="flex flex-col gap-1">
                              <span className="text-[11px] text-slate-500 uppercase tracking-wide">Količina *</span>
                              <input
                                type="number"
                                value={row.quantity === 0 ? '' : row.quantity}
                                onChange={ev => { const q = parseFloat(ev.target.value); updateRow(row.row_number, { quantity: isNaN(q) ? 0 : q }) }}
                                disabled={!isIncluded || step === 'confirming'}
                                min="0" step="any"
                                className={cx(inp, 'text-right', isIncluded && !(row.quantity > 0) && inpErr)}
                                placeholder="0"
                              />
                            </label>
                            <label className="flex flex-col gap-1">
                              <span className="text-[11px] text-slate-500 uppercase tracking-wide">JM *</span>
                              <input
                                value={row.unit}
                                onChange={ev => updateRow(row.row_number, { unit: ev.target.value })}
                                disabled={!isIncluded || step === 'confirming'}
                                className={cx(inp, isIncluded && !row.unit.trim() && inpErr)}
                                placeholder="jm"
                              />
                            </label>
                            <label className="flex flex-col gap-1">
                              <span className="text-[11px] text-slate-500 uppercase tracking-wide">Jed. cijena</span>
                              <input
                                type="number"
                                value={row.unit_price ?? ''}
                                onChange={ev => { const v = ev.target.value === '' ? null : parseFloat(ev.target.value); updateRow(row.row_number, { unit_price: isNaN(v as number) ? null : v }) }}
                                disabled={!isIncluded || step === 'confirming'}
                                min="0" step="any"
                                className={cx(inp, 'text-right')}
                                placeholder="—"
                              />
                            </label>
                            <label className="flex flex-col gap-1">
                              <span className="text-[11px] text-slate-500 uppercase tracking-wide">Ukupna cijena</span>
                              <input
                                type="number"
                                value={row.total_price ?? ''}
                                onChange={ev => { const v = ev.target.value === '' ? null : parseFloat(ev.target.value); updateRow(row.row_number, { total_price: isNaN(v as number) ? null : v }) }}
                                disabled={!isIncluded || step === 'confirming'}
                                min="0" step="any"
                                className={cx(inp, 'text-right')}
                                placeholder="—"
                              />
                            </label>
                            <label className="flex flex-col gap-1">
                              <span className="text-[11px] text-slate-500 uppercase tracking-wide">Kategorija</span>
                              <input
                                value={row.category ?? ''}
                                onChange={ev => updateRow(row.row_number, { category: ev.target.value || null })}
                                disabled={!isIncluded || step === 'confirming'}
                                className={inp}
                                placeholder="—"
                              />
                            </label>
                            <label className="flex flex-col gap-1 col-span-2 sm:col-span-3 lg:col-span-4">
                              <span className="text-[11px] text-slate-500 uppercase tracking-wide">Napomena</span>
                              <input
                                value={row.note ?? ''}
                                onChange={ev => updateRow(row.row_number, { note: ev.target.value || null })}
                                disabled={!isIncluded || step === 'confirming'}
                                className={inp}
                                placeholder="—"
                              />
                            </label>
                          </div>

                          {/* Warnings */}
                          {warnings.length > 0 && (
                            <div className="space-y-0.5">
                              {warnings.map((w, i) => (
                                <p key={i} className="text-xs text-amber-400">⚠ {w}</p>
                              ))}
                            </div>
                          )}

                          {/* Live errors */}
                          {isIncluded && live.length > 0 && (
                            <div className="space-y-0.5">
                              {live.map((msg, i) => (
                                <p key={i} className="text-xs text-red-400">✕ {msg}</p>
                              ))}
                            </div>
                          )}

                          {/* Delete */}
                          <div className="flex justify-end border-t border-slate-800/50 pt-2">
                            <button
                              onClick={() => deleteRow(row.row_number)}
                              disabled={step === 'confirming'}
                              className="text-slate-600 hover:text-red-400 text-xs transition disabled:pointer-events-none"
                            >
                              Ukloni stavku
                            </button>
                          </div>
                        </div>
                      )}
                    </div>
                  )
                })}
              </div>
            )}

            {/* ── Table mode ───────────────────────────────────────────────── */}
            {viewMode === 'table' && (
              <div className="max-h-[65vh] overflow-auto rounded-xl border border-slate-800">
                <table
                  className="table-fixed text-sm border-collapse"
                  style={{ minWidth: hasPosition ? '1620px' : '1530px' }}
                >
                  <colgroup>
                    <col style={{ width: '56px' }} />
                    <col style={{ width: '70px' }} />
                    {hasPosition && <col style={{ width: '90px' }} />}
                    <col style={{ width: '360px' }} />
                    <col style={{ width: '100px' }} />
                    <col style={{ width: '90px' }} />
                    <col style={{ width: '130px' }} />
                    <col style={{ width: '130px' }} />
                    <col style={{ width: '180px' }} />
                    <col style={{ width: '304px' }} />
                    <col style={{ width: '70px' }} />
                  </colgroup>
                  <thead className="sticky top-0 z-10 bg-slate-950">
                    <tr className="border-b border-slate-800 text-slate-400 text-xs uppercase tracking-wide">
                      <th className="px-3 py-2.5 whitespace-nowrap"></th>
                      <th className="px-3 py-2.5 text-left whitespace-nowrap">#</th>
                      {hasPosition && <th className="px-3 py-2.5 text-left whitespace-nowrap">Poz.</th>}
                      <th className="px-3 py-2.5 text-left whitespace-nowrap">Naziv materijala *</th>
                      <th className="px-3 py-2.5 text-right whitespace-nowrap">Kol. *</th>
                      <th className="px-3 py-2.5 text-left whitespace-nowrap">JM *</th>
                      <th className="px-3 py-2.5 text-right whitespace-nowrap">Jed. cijena</th>
                      <th className="px-3 py-2.5 text-right whitespace-nowrap">Ukupno</th>
                      <th className="px-3 py-2.5 text-left whitespace-nowrap">Kategorija</th>
                      <th className="px-3 py-2.5 text-left whitespace-nowrap">Napomena</th>
                      <th className="px-3 py-2.5 whitespace-nowrap"></th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-800/50">
                    {filteredRows.length === 0 ? (
                      <tr>
                        <td colSpan={colSpanCount} className="px-4 py-6 text-slate-500 text-center">
                          {noResultsMsg}
                        </td>
                      </tr>
                    ) : filteredRows.map(row => {
                      const live = rowErrors.get(row.row_number) ?? []
                      const isIncluded = row.include
                      const hasLiveErr = isIncluded && live.length > 0
                      const warnings = row.warnings ?? []

                      return (
                        <tr
                          key={row.row_number}
                          className={cx(
                            'transition-colors',
                            !isIncluded && 'opacity-35',
                            hasLiveErr && 'bg-red-950/25',
                            isIncluded && !hasLiveErr && 'hover:bg-slate-900/30',
                          )}
                        >
                          <td className="px-3 py-2.5 text-center">
                            <input type="checkbox" checked={row.include}
                              onChange={ev => updateRow(row.row_number, { include: ev.target.checked })}
                              className="h-4 w-4 accent-green-500 cursor-pointer" />
                          </td>
                          <td className="px-3 py-2.5 text-slate-500 text-xs whitespace-nowrap font-mono">
                            {row.row_number > 0
                              ? (row.source_row_numbers?.filter(n => n > 0) ?? [row.row_number]).join('+')
                              : <span className="text-slate-600 italic text-[11px]">new</span>}
                          </td>
                          {hasPosition && (
                            <td className="px-2 py-2">
                              <input value={row.position ?? ''}
                                onChange={ev => updateRow(row.row_number, { position: ev.target.value || null })}
                                disabled={!isIncluded} title={row.position ?? ''} className={inp} placeholder="—" />
                            </td>
                          )}
                          <td className="px-2 py-2">
                            {row.category && (
                              <div className="text-[11px] text-slate-500 truncate mb-0.5 leading-tight" title={row.category}>
                                {row.category}
                              </div>
                            )}
                            <input value={row.material_name}
                              onChange={ev => updateRow(row.row_number, { material_name: ev.target.value })}
                              disabled={!isIncluded} title={row.material_name}
                              className={cx(inp, isIncluded && !row.material_name.trim() && inpErr)}
                              placeholder="Naziv materijala..." />
                            {warnings.length > 0 && (
                              <div className="text-[11px] text-amber-500 truncate mt-0.5 leading-tight" title={warnings.join('; ')}>
                                ⚠ {warnings[0]}
                              </div>
                            )}
                          </td>
                          <td className="px-2 py-2">
                            <input type="number" value={row.quantity === 0 ? '' : row.quantity}
                              onChange={ev => { const q = parseFloat(ev.target.value); updateRow(row.row_number, { quantity: isNaN(q) ? 0 : q }) }}
                              disabled={!isIncluded} min="0" step="any"
                              className={cx(inp, 'text-right', isIncluded && !(row.quantity > 0) && inpErr)}
                              placeholder="0" />
                          </td>
                          <td className="px-2 py-2">
                            <input value={row.unit}
                              onChange={ev => updateRow(row.row_number, { unit: ev.target.value })}
                              disabled={!isIncluded}
                              className={cx(inp, isIncluded && !row.unit.trim() && inpErr)}
                              placeholder="jm" />
                          </td>
                          <td className="px-2 py-2">
                            <input type="number" value={row.unit_price ?? ''}
                              onChange={ev => { const v = ev.target.value === '' ? null : parseFloat(ev.target.value); updateRow(row.row_number, { unit_price: isNaN(v as number) ? null : v }) }}
                              disabled={!isIncluded} min="0" step="any"
                              className={cx(inp, 'text-right')} placeholder="—" />
                          </td>
                          <td className="px-2 py-2">
                            <input type="number" value={row.total_price ?? ''}
                              onChange={ev => { const v = ev.target.value === '' ? null : parseFloat(ev.target.value); updateRow(row.row_number, { total_price: isNaN(v as number) ? null : v }) }}
                              disabled={!isIncluded} min="0" step="any"
                              className={cx(inp, 'text-right')} placeholder="—" />
                          </td>
                          <td className="px-2 py-2">
                            <input value={row.category ?? ''}
                              onChange={ev => updateRow(row.row_number, { category: ev.target.value || null })}
                              disabled={!isIncluded} title={row.category ?? ''} className={inp} placeholder="—" />
                          </td>
                          <td className="px-2 py-2">
                            <input value={row.note ?? ''}
                              onChange={ev => updateRow(row.row_number, { note: ev.target.value || null })}
                              disabled={!isIncluded} title={row.note ?? ''} className={inp} placeholder="—" />
                          </td>
                          <td className="px-2 py-2 text-center">
                            <button onClick={() => deleteRow(row.row_number)} disabled={step === 'confirming'}
                              className="text-slate-600 hover:text-red-400 transition text-base leading-none disabled:pointer-events-none"
                              title="Ukloni redak">×</button>
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>
            )}

            {/* Add row */}
            <button
              onClick={addRow}
              disabled={step === 'confirming'}
              className="text-slate-500 hover:text-slate-300 text-xs transition disabled:opacity-30 flex items-center gap-1"
            >
              + Dodaj stavku ručno
            </button>

            {/* Inline validation errors for included rows */}
            {invalidIncluded.length > 0 && (
              <div className="space-y-0.5">
                {invalidIncluded.slice(0, 4).map(r => (
                  <p key={r.row_number} className="text-red-400 text-xs">
                    {r.row_number > 0 ? `Redak ${r.row_number}` : 'Nova stavka'}: {(rowErrors.get(r.row_number) ?? []).join(' · ')}
                  </p>
                ))}
                {invalidIncluded.length > 4 && (
                  <p className="text-slate-500 text-xs">… i još {invalidIncluded.length - 4} stavki s greškama</p>
                )}
              </div>
            )}

            {/* Confirm button */}
            {includedRows.length > 0 ? (
              <button
                onClick={handleConfirm}
                disabled={!canConfirm}
                title={invalidIncluded.length > 0 ? 'Ispravi greške u označenim stavcima prije potvrde' : undefined}
                className="px-4 py-2 bg-white text-slate-900 font-semibold rounded-lg text-sm hover:bg-slate-100 transition disabled:opacity-40 disabled:cursor-not-allowed"
              >
                {step === 'confirming' ? 'Uvoženje...' : `Potvrdi uvoz (${includedRows.length} stavki)`}
              </button>
            ) : (
              <p className="text-slate-500 text-sm">Nema označenih stavki za uvoz.</p>
            )}
          </div>
        )
      })()}

      {/* Done */}
      {step === 'done' && confirmResult && (
        <div className="bg-green-950 border border-green-800 rounded-lg px-4 py-3">
          <p className="text-green-300 text-sm font-medium">
            {confirmResult.failed_count > 0 ? 'Uvoz završen' : 'Uvoz uspješan'}
          </p>
          <p className="text-green-400 text-xs mt-1">
            Uvezeno: {confirmResult.imported_count} stavki
            {confirmResult.skipped_count > 0 && ` · Preskočeno: ${confirmResult.skipped_count}`}
            {confirmResult.failed_count > 0 && (
              <span className="text-red-400"> · Odbijeno: {confirmResult.failed_count}</span>
            )}
          </p>
          {(confirmResult.errors ?? []).length > 0 && (
            <div className="mt-1 space-y-0.5">
              {confirmResult.errors.slice(0, 3).map(e => (
                <p key={e.row_number} className="text-red-400 text-xs">
                  Redak {e.row_number}: {e.errors.join(', ')}
                </p>
              ))}
            </div>
          )}
          <button onClick={handleReset} className="text-green-400 hover:text-green-300 text-xs mt-2 underline">
            Uvezi još jednu datoteku
          </button>
        </div>
      )}
    </div>
  )
}
