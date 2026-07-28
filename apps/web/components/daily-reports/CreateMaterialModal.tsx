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
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4"
      onClick={onClose}
    >
      <div
        className="bg-slate-900 border border-slate-700 rounded-xl p-6 w-full max-w-sm space-y-4"
        onClick={e => e.stopPropagation()}
      >
        <h3 className="text-sm font-semibold text-slate-200">Dodaj novi materijal</h3>

        <form onSubmit={handleSubmit} className="space-y-3">
          <div>
            <label className="block text-xs text-slate-400 mb-1">Naziv materijala *</label>
            <input
              type="text"
              value={name}
              onChange={e => setName(e.target.value)}
              disabled={saving}
              autoFocus
              className="w-full bg-slate-800 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:opacity-50"
            />
          </div>

          <div>
            <label className="block text-xs text-slate-400 mb-1">Jedinica mjere *</label>
            <input
              type="text"
              value={unit}
              onChange={e => setUnit(e.target.value)}
              disabled={saving}
              placeholder="kom, m, m², kg…"
              className="w-full bg-slate-800 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:opacity-50"
            />
          </div>

          {error && <p className="text-xs text-red-400">{error}</p>}

          <div className="flex gap-2 pt-1">
            <button
              type="button"
              onClick={onClose}
              disabled={saving}
              className="flex-1 px-3 py-2 text-sm text-slate-400 hover:text-slate-200 border border-slate-700 rounded-lg transition disabled:opacity-50"
            >
              Odustani
            </button>
            <button
              type="submit"
              disabled={saving || !name.trim() || !unit.trim()}
              className="flex-1 px-3 py-2 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white text-sm font-medium rounded-lg transition"
            >
              {saving ? 'Dodavanje…' : 'Dodaj'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
