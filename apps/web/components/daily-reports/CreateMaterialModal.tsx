'use client'

import { useState } from 'react'
import apiClient from '@/lib/api-client'
import type { FormDataMaterial } from '@/lib/types/daily-reports'

interface Props {
  projectId: string
  initialName: string
  onCreated: (material: FormDataMaterial) => void
  onClose: () => void
}

export default function CreateMaterialModal({ projectId, initialName, onCreated, onClose }: Props) {
  const [name, setName] = useState(initialName)
  const [unit, setUnit] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const inputCls = 'w-full bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2 text-sm text-gi-text placeholder:text-gi-text-subtle focus:outline-none focus:border-gi-yellow transition-colors disabled:opacity-50'

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const trimmedName = name.trim()
    const trimmedUnit = unit.trim()
    if (!trimmedName || !trimmedUnit) {
      setError('Naziv i jedinica mjere su obavezni.')
      return
    }
    setSaving(true)
    setError(null)
    try {
      const res = await apiClient.post(`/projects/${projectId}/materials/resolve-or-create`, {
        material_name: trimmedName,
        unit: trimmedUnit,
      })
      onCreated(res.data.material as FormDataMaterial)
    } catch (e: unknown) {
      const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error
      setError(msg ?? 'Greška pri kreiranju materijala.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4"
      onClick={onClose}
    >
      <div
        className="bg-gi-card border border-gi-border rounded-xl p-6 w-full max-w-sm space-y-4 shadow-2xl"
        onClick={e => e.stopPropagation()}
      >
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold text-gi-text">Dodaj novi materijal</h3>
          <button onClick={onClose} className="text-gi-text-subtle hover:text-gi-text transition-colors p-0.5">
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-3">
          <div>
            <label className="block text-xs font-medium text-gi-text-muted uppercase tracking-wider mb-1.5">
              Naziv materijala *
            </label>
            <input
              type="text"
              value={name}
              onChange={e => setName(e.target.value)}
              disabled={saving}
              autoFocus
              className={inputCls}
            />
          </div>

          <div>
            <label className="block text-xs font-medium text-gi-text-muted uppercase tracking-wider mb-1.5">
              Jedinica mjere *
            </label>
            <input
              type="text"
              value={unit}
              onChange={e => setUnit(e.target.value)}
              disabled={saving}
              placeholder="kom, m, m², kg…"
              className={inputCls}
            />
          </div>

          {error && <p className="text-xs text-red-400">{error}</p>}

          <div className="flex gap-2 pt-1">
            <button
              type="button"
              onClick={onClose}
              disabled={saving}
              className="flex-1 px-3 py-2 text-sm text-gi-text-muted hover:text-gi-text border border-gi-border hover:bg-gi-card-elevated rounded-lg transition disabled:opacity-50"
            >
              Odustani
            </button>
            <button
              type="submit"
              disabled={saving || !name.trim() || !unit.trim()}
              className="flex-1 px-3 py-2 bg-gi-yellow hover:bg-gi-yellow-hover disabled:opacity-50 text-gi-navy text-sm font-semibold rounded-lg transition"
            >
              {saving ? 'Dodavanje…' : 'Dodaj'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
