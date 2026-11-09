import Link from 'next/link'
import BrandMark from '@/components/brand/BrandMark'
import { BRAND } from '@/lib/platform-brand'

export default function Home() {
  return (
    <main className="min-h-screen bg-gi-navy flex flex-col items-center justify-center relative overflow-hidden px-6 py-12">

      {/* ── Ambient glow orbs ── */}
      <div
        className="absolute top-0 right-0 w-[600px] h-[600px] rounded-full pointer-events-none select-none"
        style={{ background: 'radial-gradient(circle, rgba(244,184,47,0.07) 0%, transparent 70%)', transform: 'translate(30%, -30%)' }}
        aria-hidden
      />
      <div
        className="absolute bottom-0 left-0 w-[400px] h-[400px] rounded-full pointer-events-none select-none"
        style={{ background: 'radial-gradient(circle, rgba(244,184,47,0.04) 0%, transparent 70%)', transform: 'translate(-30%, 30%)' }}
        aria-hidden
      />

      {/* ── Circuit trace decoration ── */}
      <svg
        className="absolute inset-0 w-full h-full pointer-events-none select-none opacity-40"
        xmlns="http://www.w3.org/2000/svg"
        aria-hidden
      >
        <path d="M 0 80 H 60 V 40 H 120" stroke="#32445E" strokeWidth="1" fill="none" />
        <circle cx="60" cy="80" r="3" fill="#32445E" />
        <circle cx="120" cy="40" r="2.5" fill="#F4B82F" fillOpacity="0.5" />

        <path d="M calc(100% - 0px) 120 H calc(100% - 80px) V 60 H calc(100% - 160px)" stroke="#32445E" strokeWidth="1" fill="none" />
        <circle cx="calc(100% - 80px)" cy="120" r="3" fill="#32445E" />
        <circle cx="calc(100% - 160px)" cy="60" r="2.5" fill="#F4B82F" fillOpacity="0.5" />

        <path d="M 0 calc(100% - 100px) H 100 V calc(100% - 40px) H 180" stroke="#32445E" strokeWidth="1" fill="none" />
        <circle cx="100" cy="calc(100% - 100px)" r="3" fill="#32445E" />
        <circle cx="180" cy="calc(100% - 40px)" r="2.5" fill="#F4B82F" fillOpacity="0.4" />

        <path d="M calc(100% - 0px) calc(100% - 80px) H calc(100% - 100px) V calc(100% - 20px)" stroke="#32445E" strokeWidth="1" fill="none" />
        <circle cx="calc(100% - 100px)" cy="calc(100% - 80px)" r="3" fill="#32445E" />

        <path d="M 0 50% H 60 V calc(50% - 30px)" stroke="#32445E" strokeWidth="1" fill="none" />
        <circle cx="60" cy="50%" r="2.5" fill="#F4B82F" fillOpacity="0.3" />

        <path d="M calc(100%) 50% H calc(100% - 60px) V calc(50% + 30px)" stroke="#32445E" strokeWidth="1" fill="none" />
        <circle cx="calc(100% - 60px)" cy="50%" r="2.5" fill="#F4B82F" fillOpacity="0.3" />
      </svg>

      {/* ── Main content ── */}
      <div className="relative z-10 flex flex-col items-center text-center max-w-md w-full">

        <BrandMark />

        {/* Headline */}
        <h1 className="text-[26px] sm:text-[32px] font-bold text-gi-text leading-tight tracking-tight">
          Digitalno upravljanje gradilištima.
        </h1>

        {/* Supporting text */}
        <p className="text-gi-text-muted text-sm sm:text-[15px] mt-4 leading-relaxed max-w-sm">
          Projekti, zaposlenici, materijali i analitika — sve na jednom mjestu.
        </p>

        {/* Feature grid */}
        <div className="mt-6 grid grid-cols-2 gap-x-6 gap-y-2.5 w-full max-w-xs text-left">
          {[
            'Projekti i gradilišta',
            'Zaposlenici i plaće',
            'Materijali i nabava',
            'Analitika poslovanja',
          ].map(f => (
            <div key={f} className="flex items-center gap-2 text-xs text-gi-text-subtle">
              <span className="w-1.5 h-1.5 rounded-full bg-gi-yellow/60 flex-shrink-0" />
              {f}
            </div>
          ))}
        </div>

        {/* CTA */}
        <Link
          href="/login"
          className="mt-9 w-full max-w-xs block py-3.5 px-8 bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy font-bold rounded-xl text-sm transition-colors text-center shadow-lg shadow-gi-yellow/20"
        >
          Prijava
        </Link>

        {/* Subtle footer */}
        <p className="mt-10 text-gi-text-subtle text-xs">
          {BRAND.footer}
        </p>
      </div>
    </main>
  )
}
