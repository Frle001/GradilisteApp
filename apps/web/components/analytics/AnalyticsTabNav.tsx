'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'

const TABS = [
  { href: '/dashboard/analytics/pregled', label: 'Pregled' },
  { href: '/dashboard/analytics/place', label: 'Plaće' },
  { href: '/dashboard/analytics/hladni-pogon', label: 'Hladni pogon' },
]

export default function AnalyticsTabNav() {
  const pathname = usePathname()

  return (
    <div className="flex gap-1 bg-gi-card border border-gi-border rounded-xl p-1 mb-5 w-fit">
      {TABS.map((tab) => {
        const active = pathname === tab.href || pathname.startsWith(tab.href + '/')
        return (
          <Link
            key={tab.href}
            href={tab.href}
            className={[
              'px-4 py-1.5 rounded-lg text-sm font-medium transition-colors whitespace-nowrap',
              active
                ? 'bg-gi-yellow text-gi-navy'
                : 'text-gi-text-muted hover:text-gi-text hover:bg-gi-navy-hover',
            ].join(' ')}
          >
            {tab.label}
          </Link>
        )
      })}
    </div>
  )
}
