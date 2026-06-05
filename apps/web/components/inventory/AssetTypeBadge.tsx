import { ASSET_TYPE_LABELS } from '@/lib/types/inventory'

const colourMap: Record<string, string> = {
  car:       'bg-blue-900/50 text-blue-300 border-blue-700',
  tool:      'bg-amber-900/50 text-amber-300 border-amber-700',
  equipment: 'bg-violet-900/50 text-violet-300 border-violet-700',
  material:  'bg-emerald-900/50 text-emerald-300 border-emerald-700',
  other:     'bg-slate-800 text-slate-400 border-slate-600',
}

export default function AssetTypeBadge({ type }: { type: string }) {
  const cls = colourMap[type] ?? colourMap.other
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium border ${cls}`}>
      {ASSET_TYPE_LABELS[type] ?? type}
    </span>
  )
}
