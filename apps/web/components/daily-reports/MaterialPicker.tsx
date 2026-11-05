'use client'

import { useState, useRef, useEffect, Fragment } from 'react'
import { type FormDataMaterial } from '@/lib/types/daily-reports'

const MAX_VISIBLE = 20

interface Props {
  materials: FormDataMaterial[]
  loading?: boolean
  /** Currently selected project_material_id, or '' when nothing selected. */
  value: string
  /** Called with (id, unit) on selection; called with ('', '') on clear. */
  onChange: (id: string, unit: string) => void
  disabled?: boolean
  placeholder?: string
  /** Message shown when the materials list is empty. */
  emptyMessage?: string
  /** When provided, items matching this predicate are shown but not selectable. */
  isItemDisabled?: (m: FormDataMaterial) => boolean
  /** When provided, a "Dodaj novi materijal" option is shown for unmatched queries. */
  onCreateNew?: (name: string) => void
}

export default function MaterialPicker({
  materials,
  loading = false,
  value,
  onChange,
  disabled = false,
  placeholder = 'Odaberi materijal…',
  emptyMessage = 'Nema dostupnog materijala na projektu.',
  isItemDisabled,
  onCreateNew,
}: Props) {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [highlighted, setHighlighted] = useState(0)

  const containerRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLUListElement>(null)

  const selected = materials.find(m => m.id === value) ?? null

  const filtered = (() => {
    const q = query.trim().toLowerCase()
    if (!q) return materials.slice(0, MAX_VISIBLE)
    return materials
      .filter(m =>
        m.material_name.toLowerCase().includes(q) ||
        m.unit.toLowerCase().includes(q) ||
        (m.material_code?.toLowerCase().includes(q) ?? false)
      )
      .slice(0, MAX_VISIBLE)
  })()

  const trimmedQuery = query.trim()
  const normalizedQuery = trimmedQuery.toLocaleLowerCase('hr-HR')
  const exactMatch = materials.some(
    m => m.material_name.trim().toLocaleLowerCase('hr-HR') === normalizedQuery
  )
  const showCreate =
    Boolean(onCreateNew) &&
    !disabled &&
    normalizedQuery.length > 0 &&
    !exactMatch

  // Close when user clicks outside the component
  useEffect(() => {
    function onOutsideMouseDown(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', onOutsideMouseDown)
    return () => document.removeEventListener('mousedown', onOutsideMouseDown)
  }, [])

  // Reset highlighted row whenever the filtered list or open state changes
  useEffect(() => { setHighlighted(0) }, [query, open])

  // Scroll highlighted row into view during keyboard navigation
  useEffect(() => {
    const li = listRef.current?.children[highlighted] as HTMLElement | undefined
    li?.scrollIntoView({ block: 'nearest' })
  }, [highlighted])

  function openPicker() {
    if (disabled || loading) return
    setQuery('')
    setOpen(true)
    setTimeout(() => inputRef.current?.focus(), 0)
  }

  function select(mat: FormDataMaterial) {
    onChange(mat.id, mat.unit)
    setOpen(false)
    setQuery('')
  }

  function clear(e: React.MouseEvent) {
    e.stopPropagation()
    onChange('', '')
    setOpen(false)
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    const maxIdx = filtered.length - 1 + (showCreate ? 1 : 0)
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault()
        setHighlighted(i => Math.min(i + 1, maxIdx))
        break
      case 'ArrowUp':
        e.preventDefault()
        setHighlighted(i => Math.max(i - 1, 0))
        break
      case 'Enter':
        e.preventDefault()
        if (highlighted < filtered.length && filtered[highlighted]) {
          const item = filtered[highlighted]
          if (!isItemDisabled?.(item)) select(item)
        } else if (showCreate && highlighted === filtered.length) {
          onCreateNew!(trimmedQuery)
          setOpen(false)
          setQuery('')
        }
        break
      case 'Escape':
        e.preventDefault()
        setOpen(false)
        break
    }
  }

  // Index of the first disabled item in filtered (used to render the "Nema na stanju" separator).
  const firstDisabledIdx = isItemDisabled
    ? filtered.findIndex(m => isItemDisabled(m))
    : -1

  return (
    <div ref={containerRef} className="relative w-full">

      {/* ── Closed trigger ───────────────────────────────────────────────── */}
      {!open && (
        <div
          role="combobox"
          aria-haspopup="listbox"
          aria-expanded={false}
          tabIndex={disabled ? -1 : 0}
          onClick={openPicker}
          onKeyDown={e => {
            if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); openPicker() }
          }}
          className={[
            'w-full bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2 text-sm',
            'flex items-center justify-between gap-2 select-none transition-colors',
            disabled
              ? 'opacity-50 cursor-not-allowed'
              : 'cursor-pointer hover:border-gi-yellow/60 focus:outline-none focus:border-gi-yellow',
          ].join(' ')}
        >
          {selected ? (
            <>
              <span className="text-gi-text truncate min-w-0">{selected.material_name}</span>
              <button
                type="button"
                onClick={clear}
                disabled={disabled}
                aria-label="Ukloni odabir"
                className="shrink-0 text-gi-text-subtle hover:text-gi-text px-0.5 text-lg leading-none focus:outline-none"
              >
                ×
              </button>
            </>
          ) : (
            <span className="text-gi-text-subtle truncate">
              {loading ? 'Učitavanje materijala…' : placeholder}
            </span>
          )}
        </div>
      )}

      {/* ── Search input (shown when open) ───────────────────────────────── */}
      {open && (
        <input
          ref={inputRef}
          type="text"
          value={query}
          onChange={e => setQuery(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Pretraži po nazivu, šifri ili jedinici…"
          autoComplete="off"
          className="w-full bg-gi-card-elevated border border-gi-yellow ring-1 ring-gi-yellow/40 rounded-lg px-3 py-2 text-sm text-gi-text placeholder:text-gi-text-subtle focus:outline-none"
        />
      )}

      {/* ── Dropdown panel ───────────────────────────────────────────────── */}
      {open && (
        <div className="absolute z-50 left-0 right-0 mt-1 bg-gi-workspace border border-gi-border rounded-xl shadow-2xl overflow-hidden">
          {materials.length === 0 && !showCreate ? (
            <p className="px-3 py-2.5 text-sm text-gi-text-subtle">
              {emptyMessage}
            </p>
          ) : filtered.length === 0 && !showCreate ? (
            <p className="px-3 py-2.5 text-sm text-gi-text-subtle">
              Nema pronađenih materijala.
            </p>
          ) : (
            <ul
              ref={listRef}
              role="listbox"
              aria-label="Popis materijala"
              className="max-h-56 overflow-y-auto divide-y divide-gi-border/60"
            >
              {filtered.map((m, idx) => {
                const itemDisabled = isItemDisabled ? isItemDisabled(m) : false
                const showSeparator = idx === firstDisabledIdx && firstDisabledIdx > 0
                return (
                  <Fragment key={m.id}>
                    {showSeparator && (
                      <li
                        role="separator"
                        className="px-3 py-1.5 text-xs font-medium text-gi-text-subtle uppercase tracking-wider bg-gi-card-elevated border-t border-gi-border select-none pointer-events-none"
                      >
                        Nema na stanju
                      </li>
                    )}
                    <li
                      role="option"
                      aria-selected={m.id === value}
                      aria-disabled={itemDisabled}
                      onClick={() => { if (!itemDisabled) select(m) }}
                      onMouseEnter={() => setHighlighted(idx)}
                      className={[
                        'px-3 py-2.5 transition-colors',
                        itemDisabled
                          ? 'opacity-40 cursor-not-allowed'
                          : ['cursor-pointer', idx === highlighted ? 'bg-gi-yellow/10' : 'hover:bg-gi-card-elevated'].join(' '),
                        m.id === value ? 'bg-gi-yellow/5' : '',
                      ].join(' ')}
                    >
                      <p className="text-sm text-gi-text leading-snug break-words line-clamp-2">
                        {m.material_name}
                      </p>
                      <p className="text-xs text-gi-text-subtle mt-0.5 truncate">
                        {m.unit}
                        {m.material_code ? ` · ${m.material_code}` : ''}
                        {m.tracking_type === 'work'
                          ? <span className="text-emerald-500"> · radna aktivnost</span>
                          : ` · kol: ${m.available_quantity}`}
                      </p>
                    </li>
                  </Fragment>
                )
              })}
              {showCreate && (
                <li
                  role="option"
                  aria-selected={false}
                  onClick={() => { onCreateNew!(trimmedQuery); setOpen(false); setQuery('') }}
                  onMouseEnter={() => setHighlighted(filtered.length)}
                  className={[
                    'px-3 py-2.5 cursor-pointer transition-colors',
                    filtered.length > 0 ? 'border-t border-gi-border' : '',
                    highlighted === filtered.length ? 'bg-gi-yellow/10' : 'hover:bg-gi-card-elevated',
                  ].join(' ')}
                >
                  <p className="text-sm text-gi-yellow leading-snug">
                    + Dodaj novi materijal: &ldquo;{trimmedQuery}&rdquo;
                  </p>
                </li>
              )}
            </ul>
          )}
        </div>
      )}
    </div>
  )
}
