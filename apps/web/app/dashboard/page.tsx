'use client'

import Link from 'next/link'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
import {
  FolderOpen,
  FileText,
  Users,
  Clock,
  CalendarDays,
  ClipboardList,
  Receipt,
  BarChart2,
  Archive,
  ShoppingCart,
  Package,
  Wrench,
  ArrowRight,
  PlusCircle,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

interface DashboardCard {
  title: string
  description: string
  Icon: LucideIcon
  href: string
  highlight?: boolean
}

interface CardSection {
  label: string
  cards: DashboardCard[]
}

const cardSections: Record<string, CardSection[]> = {
  direktor: [
    {
      label: 'Operativa',
      cards: [
        { title: 'Projekti / Gradilišta', description: 'Pregled i upravljanje projektima', Icon: FolderOpen, href: '/dashboard/projects' },
        { title: 'Novi projekt', description: 'Pokretanje novog gradilišta', Icon: PlusCircle, href: '/dashboard/projects/new', highlight: true },
        { title: 'Raspored smjena', description: 'Tjedni raspored radnih smjena', Icon: CalendarDays, href: '/dashboard/schedule' },
        { title: 'Dnevni izvještaji', description: 'Pregled svih dnevnih izvještaja', Icon: FileText, href: '/dashboard/daily-reports' },
      ],
    },
    {
      label: 'Radni sati',
      cards: [
        { title: 'Radni sati', description: 'Unos i pregled radnih sati', Icon: Clock, href: '/dashboard/radni-sati' },
      ],
    },
    {
      label: 'Materijali i oprema',
      cards: [
        { title: 'Upisi materijala', description: 'Evidencija kupljenog materijala', Icon: ShoppingCart, href: '/dashboard/material-purchases' },
        { title: 'Stanje robe', description: 'Osobna zadužnica i prijenosi', Icon: Package, href: '/dashboard/inventory' },
      ],
    },
    {
      label: 'Financije i upravljanje',
      cards: [
        { title: 'Zaposlenici', description: 'Upravljanje osobljem', Icon: Users, href: '/dashboard/employees' },
        { title: 'Analitika', description: 'Plaće, hladni pogon i troškovi poslovanja', Icon: BarChart2, href: '/dashboard/analytics' },
        { title: 'Računi i troškovi', description: 'R1, gorivo i ulazni računi tvrtke', Icon: Receipt, href: '/dashboard/racuni-i-troskovi' },
        { title: 'Građevinska evidencija', description: 'Dnevnik i knjiga radova po gradilištu', Icon: Archive, href: '/dashboard/gradevinska-evidencija' },
      ],
    },
  ],
  inzenjer: [
    {
      label: 'Operativa',
      cards: [
        { title: 'Projekti / Gradilišta', description: 'Pregled i upravljanje projektima', Icon: FolderOpen, href: '/dashboard/projects' },
        { title: 'Novi projekt', description: 'Pokretanje novog gradilišta', Icon: PlusCircle, href: '/dashboard/projects/new', highlight: true },
        { title: 'Raspored smjena', description: 'Tjedni raspored radnih smjena', Icon: CalendarDays, href: '/dashboard/schedule' },
        { title: 'Dnevni izvještaji', description: 'Pregled svih dnevnih izvještaja', Icon: FileText, href: '/dashboard/daily-reports' },
      ],
    },
    {
      label: 'Radni sati',
      cards: [
        { title: 'Radni sati', description: 'Unos i pregled radnih sati', Icon: Clock, href: '/dashboard/radni-sati' },
      ],
    },
    {
      label: 'Materijali i oprema',
      cards: [
        { title: 'Upisi materijala', description: 'Evidencija kupljenog materijala', Icon: ShoppingCart, href: '/dashboard/material-purchases' },
        { title: 'Stanje robe', description: 'Osobna zadužnica i prijenosi', Icon: Package, href: '/dashboard/inventory' },
      ],
    },
    {
      label: 'Financije i upravljanje',
      cards: [
        { title: 'Zaposlenici', description: 'Pregled osoblja', Icon: Users, href: '/dashboard/employees' },
        { title: 'Analitika', description: 'Plaće, hladni pogon i troškovi poslovanja', Icon: BarChart2, href: '/dashboard/analytics' },
        { title: 'Računi i troškovi', description: 'R1 i gorivo računi', Icon: Receipt, href: '/dashboard/racuni-i-troskovi' },
        { title: 'Građevinska evidencija', description: 'Dnevnik i knjiga radova po gradilištu', Icon: Archive, href: '/dashboard/gradevinska-evidencija' },
      ],
    },
  ],
  administracija: [
    {
      label: 'Radni sati',
      cards: [
        { title: 'Radni sati', description: 'Evidencija vlastitih radnih sati', Icon: Clock, href: '/dashboard/radni-sati' },
      ],
    },
    {
      label: 'Materijali i oprema',
      cards: [
        { title: 'Stanje robe', description: 'Osobna zadužnica i prijenosi', Icon: Package, href: '/dashboard/inventory' },
        { title: 'Zaduženja', description: 'Evidencija alata, opreme i vozila', Icon: Wrench, href: '/dashboard/zaduzenja' },
      ],
    },
    {
      label: 'Financije i upravljanje',
      cards: [
        { title: 'Zaposlenici', description: 'Upravljanje zaposlenicima', Icon: Users, href: '/dashboard/employees' },
        { title: 'Dokumentacija', description: 'HR dokumenti i compliance zaposlenika', Icon: ClipboardList, href: '/dashboard/dokumentacija' },
        { title: 'Računi i troškovi', description: 'Gorivo i ulazni računi tvrtke', Icon: Receipt, href: '/dashboard/racuni-i-troskovi' },
      ],
    },
  ],
  poslovoda: [
    {
      label: 'Operativa',
      cards: [
        { title: 'Moja gradilišta', description: 'Aktivni projekti dodijeljeni vama', Icon: FolderOpen, href: '/dashboard/projects' },
        { title: 'Raspored smjena', description: 'Tjedni raspored radnih smjena', Icon: CalendarDays, href: '/dashboard/schedule' },
        { title: 'Dnevni izvještaj', description: 'Unos dnevnog izvještaja', Icon: FileText, href: '/dashboard/daily-reports', highlight: true },
      ],
    },
    {
      label: 'Radni sati',
      cards: [
        { title: 'Radni sati', description: 'Unos vlastitih dnevnih radnih sati', Icon: Clock, href: '/dashboard/radni-sati' },
      ],
    },
    {
      label: 'Materijali i oprema',
      cards: [
        { title: 'Upis materijala', description: 'Evidentiranje kupljenog materijala', Icon: ShoppingCart, href: '/dashboard/material-purchases' },
        { title: 'Osobno stanje robe', description: 'Zadužnica, oprema i prijenosi', Icon: Package, href: '/dashboard/inventory' },
      ],
    },
    {
      label: 'Financije i upravljanje',
      cards: [
        { title: 'Računi i troškovi', description: 'R1 i gorivo računi', Icon: Receipt, href: '/dashboard/racuni-i-troskovi' },
        { title: 'Građevinska evidencija', description: 'Dnevnik i knjiga radova po gradilištu', Icon: Archive, href: '/dashboard/gradevinska-evidencija' },
      ],
    },
  ],
  radnik: [
    {
      label: 'Operativa',
      cards: [
        { title: 'Raspored smjena', description: 'Moje smjene i raspored', Icon: CalendarDays, href: '/dashboard/schedule' },
        { title: 'Projekti / Gradilišta', description: 'Pregled projekata tvrtke', Icon: FolderOpen, href: '/dashboard/projects' },
      ],
    },
    {
      label: 'Radni sati',
      cards: [
        { title: 'Radni sati', description: 'Unos vlastitih dnevnih radnih sati', Icon: Clock, href: '/dashboard/radni-sati', highlight: true },
      ],
    },
    {
      label: 'Financije i upravljanje',
      cards: [
        { title: 'Računi i troškovi', description: 'R1 i gorivo računi', Icon: Receipt, href: '/dashboard/racuni-i-troskovi' },
      ],
    },
  ],
}

function getGreeting(): string {
  const h = new Date().getHours()
  if (h < 12) return 'Dobro jutro'
  if (h < 18) return 'Dobar dan'
  return 'Dobra večer'
}

function getTodayLabel(): string {
  return new Date().toLocaleDateString('hr-HR', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  })
}

export default function DashboardPage() {
  const { user, employee, isLoading, logout } = useAuth()

  if (isLoading) return <LoadingScreen />
  if (!user) return null

  const firstName = employee ? employee.first_name : user.email.split('@')[0]
  const sections = cardSections[user.role] ?? []

  return (
    <AppShell user={user} employee={employee} title="Početna" onLogout={logout}>
      {/* ── Welcome ── */}
      <div className="mb-5 sm:mb-7">
        <p className="text-gi-text-subtle text-[11px] font-medium uppercase tracking-widest mb-1 capitalize">
          {getTodayLabel()}
        </p>
        <h1 className="text-gi-text text-[26px] sm:text-[30px] font-bold leading-tight">
          {getGreeting()}, {firstName}
        </h1>
      </div>

      {/* ── Card sections ── */}
      {sections.length > 0 ? (
        <div className="space-y-5 sm:space-y-7">
          {sections.map((section) => (
            <div key={section.label}>
              <h2 className="text-gi-text-subtle text-[10px] font-semibold uppercase tracking-widest mb-2.5">
                {section.label}
              </h2>
              <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-2 sm:gap-2.5">
                {section.cards.map((card) => (
                  <Link
                    key={card.href}
                    href={card.href}
                    className={`group flex items-start gap-3.5 p-4 rounded-xl border transition-all duration-150 hover:bg-gi-card-elevated ${
                      card.highlight
                        ? 'bg-gi-card border-gi-yellow/30 hover:border-gi-yellow/60'
                        : 'bg-gi-card border-gi-border hover:border-gi-border/80'
                    }`}
                  >
                    {/* Icon */}
                    <div
                      className={`w-9 h-9 rounded-lg flex items-center justify-center flex-shrink-0 mt-0.5 transition-colors duration-150 ${
                        card.highlight
                          ? 'bg-gi-yellow/15 text-gi-yellow'
                          : 'bg-white/[0.06] text-gi-text-muted group-hover:bg-gi-yellow/10 group-hover:text-gi-yellow'
                      }`}
                    >
                      <card.Icon className="w-[17px] h-[17px]" strokeWidth={1.8} />
                    </div>

                    {/* Text */}
                    <div className="flex-1 min-w-0">
                      <h3 className="text-gi-text text-[13px] font-semibold leading-snug">
                        {card.title}
                      </h3>
                      <p className="text-gi-text-subtle text-[11px] mt-0.5 leading-snug">
                        {card.description}
                      </p>
                    </div>

                    {/* Arrow */}
                    <ArrowRight
                      className={`w-3.5 h-3.5 flex-shrink-0 mt-1 transition-colors duration-150 ${
                        card.highlight
                          ? 'text-gi-yellow/60 group-hover:text-gi-yellow'
                          : 'text-gi-text-subtle/50 group-hover:text-gi-yellow'
                      }`}
                      strokeWidth={2}
                    />
                  </Link>
                ))}
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="bg-gi-card border border-gi-border rounded-xl p-8 text-center">
          <p className="text-gi-text-muted text-sm">Nema dostupnih modula za ovu ulogu.</p>
        </div>
      )}
    </AppShell>
  )
}
