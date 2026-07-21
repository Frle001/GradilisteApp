'use client'

import Link from 'next/link'
import { useState, useEffect } from 'react'

type Platform = 'ios' | 'android' | 'desktop' | 'unknown'

function detectPlatform(): Platform {
  if (typeof navigator === 'undefined') return 'unknown'
  const ua = navigator.userAgent
  if (/iPad|iPhone|iPod/.test(ua) && !(window as Window & { MSStream?: unknown }).MSStream) return 'ios'
  if (/Android/.test(ua)) return 'android'
  if (/Macintosh|Windows|Linux/.test(ua)) return 'desktop'
  return 'unknown'
}

function isStandalone(): boolean {
  if (typeof window === 'undefined') return false
  return (
    window.matchMedia('(display-mode: standalone)').matches ||
    ('standalone' in navigator && (navigator as Navigator & { standalone?: boolean }).standalone === true)
  )
}

export default function InstallPage() {
  const [platform, setPlatform] = useState<Platform>('unknown')
  const [installed, setInstalled] = useState(false)

  useEffect(() => {
    setPlatform(detectPlatform())
    setInstalled(isStandalone())
  }, [])

  if (installed) {
    return (
      <div className="min-h-screen bg-slate-950 flex items-center justify-center px-6">
        <div className="text-center max-w-xs">
          <div className="text-4xl mb-4">✓</div>
          <h1 className="text-xl font-bold text-white mb-2">Već instalirano</h1>
          <p className="text-slate-400 text-sm mb-6">Aplikacija radi kao samostalna PWA.</p>
          <Link href="/dashboard" className="block w-full py-3 px-6 bg-white text-slate-900 font-semibold rounded-lg text-sm hover:bg-slate-100 transition text-center">
            Na početnu
          </Link>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-slate-950 text-white px-6 py-12">
      <div className="max-w-sm mx-auto">
        <div className="mb-8 text-center">
          <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-slate-800 border border-slate-700 flex items-center justify-center">
            <span className="text-3xl font-bold">G</span>
          </div>
          <h1 className="text-2xl font-bold mb-1">Instaliraj Gradilište</h1>
          <p className="text-slate-400 text-sm">Dodaj aplikaciju na početni zaslon za brži pristup.</p>
        </div>

        {platform === 'ios' && (
          <div className="space-y-4">
            <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 flex items-start gap-3">
              <span className="text-xl shrink-0">1</span>
              <p className="text-sm text-slate-300">
                Dodirnite ikonu <strong className="text-white">Dijeli</strong> (kvadrat sa strelicom gore) u alatnoj traci Safarija.
              </p>
            </div>
            <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 flex items-start gap-3">
              <span className="text-xl shrink-0">2</span>
              <p className="text-sm text-slate-300">
                Pomičite popis prema dolje i odaberite <strong className="text-white">„Dodaj na početni zaslon"</strong>.
              </p>
            </div>
            <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 flex items-start gap-3">
              <span className="text-xl shrink-0">3</span>
              <p className="text-sm text-slate-300">
                Potvrdite dodirom na <strong className="text-white">Dodaj</strong> u gornjem desnom kutu.
              </p>
            </div>
            <p className="text-xs text-slate-500 text-center mt-2">Funkcionira samo u Safariju na iOS-u.</p>
          </div>
        )}

        {platform === 'android' && (
          <div className="space-y-4">
            <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 flex items-start gap-3">
              <span className="text-xl shrink-0">1</span>
              <p className="text-sm text-slate-300">
                Dodirnite izbornik <strong className="text-white">⋮</strong> u gornjem desnom kutu preglednika Chrome.
              </p>
            </div>
            <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 flex items-start gap-3">
              <span className="text-xl shrink-0">2</span>
              <p className="text-sm text-slate-300">
                Odaberite <strong className="text-white">Dodaj na početni zaslon</strong> ili <strong className="text-white">„Instaliraj aplikaciju"</strong>.
              </p>
            </div>
            <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 flex items-start gap-3">
              <span className="text-xl shrink-0">3</span>
              <p className="text-sm text-slate-300">
                Potvrdite dodirom na <strong className="text-white">Instaliraj</strong>.
              </p>
            </div>
          </div>
        )}

        {platform === 'desktop' && (
          <div className="space-y-4">
            <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 flex items-start gap-3">
              <span className="text-xl shrink-0">1</span>
              <p className="text-sm text-slate-300">
                U pregledavatelju Chrome potražite ikonu instalacije <strong className="text-white">⊕</strong> u adresnoj traci desno.
              </p>
            </div>
            <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 flex items-start gap-3">
              <span className="text-xl shrink-0">2</span>
              <p className="text-sm text-slate-300">
                Kliknite na nju i potvrdite s <strong className="text-white">Instaliraj</strong>.
              </p>
            </div>
            <p className="text-xs text-slate-500 text-center mt-2">
              Podržano u Chromeu i Edgeu. Firefox ne podržava PWA instalaciju.
            </p>
          </div>
        )}

        {platform === 'unknown' && (
          <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 text-sm text-slate-400 text-center">
            Koristite Safari (iOS) ili Chrome (Android/Desktop) za instalaciju.
          </div>
        )}

        <div className="mt-8">
          <Link
            href="/dashboard"
            className="block w-full py-3 px-6 text-center text-sm text-slate-400 hover:text-white border border-slate-800 rounded-lg transition"
          >
            Nastavi u pregledavatelju
          </Link>
        </div>
      </div>
    </div>
  )
}
