import { Layers } from 'lucide-react'
import { BRAND } from '@/lib/platform-brand'

interface BrandMarkProps {
  variant?: 'welcome' | 'login'
}

export default function BrandMark({ variant = 'welcome' }: BrandMarkProps) {
  if (variant === 'login') {
    return (
      <div className="mb-10 flex items-center gap-3">
        <div className="w-10 h-10 rounded-xl bg-gi-yellow flex items-center justify-center shadow-md shadow-gi-yellow/25 flex-shrink-0">
          <Layers className="w-5 h-5 text-gi-navy" strokeWidth={1.75} />
        </div>
        <div>
          <p className="text-gi-text font-bold text-base leading-none">{BRAND.name}</p>
          <p className="text-gi-text-subtle text-[11px] mt-0.5 tracking-wide uppercase">{BRAND.tagline}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col items-center mb-8">
      <div className="w-12 h-12 rounded-xl bg-gi-yellow flex items-center justify-center shadow-lg shadow-gi-yellow/30 mb-5">
        <Layers className="w-6 h-6 text-gi-navy" strokeWidth={1.75} />
      </div>
      <p className="text-gi-yellow text-[11px] font-semibold tracking-[0.2em] uppercase">
        {BRAND.name} · {BRAND.tagline}
      </p>
    </div>
  )
}
