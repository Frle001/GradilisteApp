'use client'

import { useState } from 'react'
import Image from 'next/image'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import {
  Home,
  FolderOpen,
  FileText,
  Users,
  Clock,
  CalendarDays,
  ClipboardList,
  Receipt,
  BarChart2,
  Archive,
  BookOpen,
  Briefcase,
  ShoppingCart,
  Package,
  Wrench,
  Menu,
  X,
  ChevronLeft,
  LogOut,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { ROLE_LABELS } from '@/lib/types/employees'
import MobileBottomNav from '@/components/layout/MobileBottomNav'
import SyncStatusBanner from '@/components/ui/SyncStatusBanner'
import { useServerReachable } from '@/lib/connection-state'
import type { MeUser, MeEmployee } from '@/hooks/useAuth'

interface NavItem {
  href: string
  label: string
  Icon: LucideIcon
  exact?: boolean
}

interface NavGroup {
  label: string
  items: NavItem[]
}

const navGroups: Record<string, NavGroup[]> = {
  direktor: [
    {
      label: 'Pregled',
      items: [{ href: '/dashboard', label: 'Početna', Icon: Home, exact: true }],
    },
    {
      label: 'Operativa',
      items: [
        { href: '/dashboard/projects', label: 'Projekti', Icon: FolderOpen },
        { href: '/dashboard/schedule', label: 'Raspored smjena', Icon: CalendarDays },
        { href: '/dashboard/daily-reports', label: 'Dnevni izvještaji', Icon: FileText },
        { href: '/dashboard/worker-hours', label: 'Sati zaposlenika', Icon: Clock },
        { href: '/dashboard/management-hours', label: 'Moji sati', Icon: Briefcase },
      ],
    },
    {
      label: 'Materijali',
      items: [
        { href: '/dashboard/material-purchases', label: 'Upisi materijala', Icon: ShoppingCart },
        { href: '/dashboard/inventory', label: 'Stanje robe', Icon: Package },
      ],
    },
    {
      label: 'Upravljanje',
      items: [
        { href: '/dashboard/employees', label: 'Zaposlenici', Icon: Users },
        { href: '/dashboard/analytics', label: 'Analitika', Icon: BarChart2 },
        { href: '/dashboard/zaduzenja', label: 'Zaduženja', Icon: Wrench },
        { href: '/dashboard/dokumentacija', label: 'Dokumentacija', Icon: ClipboardList },
        { href: '/dashboard/racuni', label: 'Računi', Icon: Receipt },
        { href: '/dashboard/r1', label: 'R1 računi', Icon: BookOpen },
        { href: '/dashboard/reports/gradevinski-dnevnik', label: 'Građevinski dnevnik', Icon: ClipboardList },
        { href: '/dashboard/reports/gradevinska-knjiga', label: 'Građevinska knjiga', Icon: Archive },
      ],
    },
  ],
  inzenjer: [
    {
      label: 'Pregled',
      items: [{ href: '/dashboard', label: 'Početna', Icon: Home, exact: true }],
    },
    {
      label: 'Operativa',
      items: [
        { href: '/dashboard/projects', label: 'Projekti', Icon: FolderOpen },
        { href: '/dashboard/schedule', label: 'Raspored smjena', Icon: CalendarDays },
        { href: '/dashboard/daily-reports', label: 'Dnevni izvještaji', Icon: FileText },
        { href: '/dashboard/worker-hours', label: 'Sati zaposlenika', Icon: Clock },
        { href: '/dashboard/management-hours', label: 'Moji sati', Icon: Briefcase },
      ],
    },
    {
      label: 'Materijali',
      items: [
        { href: '/dashboard/material-purchases', label: 'Upisi materijala', Icon: ShoppingCart },
        { href: '/dashboard/inventory', label: 'Stanje robe', Icon: Package },
      ],
    },
    {
      label: 'Upravljanje',
      items: [
        { href: '/dashboard/employees', label: 'Zaposlenici', Icon: Users },
        { href: '/dashboard/analytics', label: 'Analitika', Icon: BarChart2 },
        { href: '/dashboard/r1', label: 'Moji R1', Icon: BookOpen },
        { href: '/dashboard/reports/gradevinski-dnevnik', label: 'Građevinski dnevnik', Icon: ClipboardList },
        { href: '/dashboard/reports/gradevinska-knjiga', label: 'Građevinska knjiga', Icon: Archive },
      ],
    },
  ],
  administracija: [
    {
      label: 'Pregled',
      items: [{ href: '/dashboard', label: 'Početna', Icon: Home, exact: true }],
    },
    {
      label: 'Upravljanje',
      items: [
        { href: '/dashboard/employees', label: 'Zaposlenici', Icon: Users },
        { href: '/dashboard/management-hours', label: 'Moji sati', Icon: Clock },
        { href: '/dashboard/inventory', label: 'Stanje robe', Icon: Package },
        { href: '/dashboard/zaduzenja', label: 'Zaduženja', Icon: Wrench },
        { href: '/dashboard/dokumentacija', label: 'Dokumentacija', Icon: ClipboardList },
        { href: '/dashboard/racuni', label: 'Računi', Icon: Receipt },
      ],
    },
  ],
  poslovoda: [
    {
      label: 'Pregled',
      items: [{ href: '/dashboard', label: 'Početna', Icon: Home, exact: true }],
    },
    {
      label: 'Operativa',
      items: [
        { href: '/dashboard/projects', label: 'Moja gradilišta', Icon: FolderOpen },
        { href: '/dashboard/schedule', label: 'Raspored smjena', Icon: CalendarDays },
        { href: '/dashboard/daily-reports', label: 'Dnevni izvještaj', Icon: FileText },
        { href: '/dashboard/my-hours', label: 'Moji sati', Icon: Clock },
      ],
    },
    {
      label: 'Materijali',
      items: [
        { href: '/dashboard/material-purchases', label: 'Upisi materijala', Icon: ShoppingCart },
        { href: '/dashboard/inventory', label: 'Stanje robe', Icon: Package },
      ],
    },
    {
      label: 'Izvještaji',
      items: [
        { href: '/dashboard/r1', label: 'Moji R1', Icon: Receipt },
        { href: '/dashboard/reports/gradevinski-dnevnik', label: 'Građevinski dnevnik', Icon: ClipboardList },
        { href: '/dashboard/reports/gradevinska-knjiga', label: 'Građevinska knjiga', Icon: Archive },
      ],
    },
  ],
  radnik: [
    {
      label: 'Pregled',
      items: [{ href: '/dashboard', label: 'Početna', Icon: Home, exact: true }],
    },
    {
      label: 'Operativa',
      items: [
        { href: '/dashboard/schedule', label: 'Raspored smjena', Icon: CalendarDays },
        { href: '/dashboard/my-hours', label: 'Moji sati', Icon: Clock },
        { href: '/dashboard/projects', label: 'Projekti', Icon: FolderOpen },
        { href: '/dashboard/r1', label: 'Moji R1', Icon: Receipt },
      ],
    },
  ],
}

export interface AppShellProps {
  user: MeUser
  employee: MeEmployee | null
  title: string
  children: React.ReactNode
  onLogout: () => void
  backHref?: string
  action?: React.ReactNode
}


export default function AppShell({
  user,
  employee,
  title,
  children,
  onLogout,
  backHref,
  action,
}: AppShellProps) {
  const pathname = usePathname()
  const serverReachable = useServerReachable()
  const [mobileOpen, setMobileOpen] = useState(false)

  const groups = navGroups[user.role] ?? []
  const displayName = employee
    ? `${employee.first_name} ${employee.last_name}`
    : user.email
  const roleLabel = ROLE_LABELS[user.role] ?? user.role
  const initial = displayName.charAt(0).toUpperCase()

  function isActive(item: NavItem): boolean {
    if (item.exact) return pathname === item.href
    return pathname === item.href || pathname.startsWith(item.href + '/')
  }

  const sidebarInner = (
    <div className="flex flex-col h-full overflow-hidden">
      {/* ── Logo ── */}
      <div className="border-b border-white/[0.07] flex-shrink-0">
        <Link href="/dashboard" className="block">
          <Image
            src="/branding/get-install-original.jpeg"
            alt="Get Install Electrical Solutions"
            width={1456}
            height={816}
            className="w-full h-auto block"
            priority
          />
        </Link>
      </div>

      {/* ── Navigation groups ── */}
      <nav className="flex-1 overflow-y-auto py-3 px-3 space-y-4 gi-scroll">
        {groups.map((group) => (
          <div key={group.label}>
            <p className="text-gi-text-subtle text-[10px] font-semibold uppercase tracking-[0.12em] px-2 mb-1">
              {group.label}
            </p>
            <ul className="space-y-0.5">
              {group.items.map((item) => {
                const active = isActive(item)
                return (
                  <li key={item.href}>
                    <Link
                      href={item.href}
                      onClick={() => setMobileOpen(false)}
                      className={`flex items-center gap-2.5 px-2.5 py-[7px] rounded-lg text-[13px] transition-colors duration-150 ${
                        active
                          ? 'bg-gi-yellow text-gi-navy font-semibold'
                          : 'text-gi-text-muted hover:text-gi-text hover:bg-gi-navy-hover'
                      }`}
                    >
                      <item.Icon
                        className="w-[15px] h-[15px] flex-shrink-0"
                        strokeWidth={active ? 2.5 : 1.8}
                      />
                      {item.label}
                    </Link>
                  </li>
                )
              })}
            </ul>
          </div>
        ))}
      </nav>

      {/* ── User footer ── */}
      <div className="flex-shrink-0 border-t border-white/[0.07] px-3 py-3">
        <div className="flex items-center gap-2.5">
          <div className="w-7 h-7 rounded-full bg-gi-yellow/[0.15] flex items-center justify-center flex-shrink-0">
            <span className="text-gi-yellow text-[11px] font-bold">{initial}</span>
          </div>
          <div className="min-w-0 flex-1">
            <p className="text-gi-text text-[12px] font-medium truncate leading-none">
              {displayName}
            </p>
            <p className="text-gi-text-subtle text-[11px] truncate leading-none mt-[3px]">
              {roleLabel}
            </p>
          </div>
          <button
            onClick={onLogout}
            className="text-gi-text-subtle hover:text-gi-text p-1.5 rounded-lg hover:bg-gi-navy-hover transition-colors flex-shrink-0"
            title="Odjava"
          >
            <LogOut className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>
    </div>
  )

  return (
    <div className="flex h-screen overflow-hidden bg-gi-workspace">
      {/* ── Desktop sidebar ── */}
      <aside className="hidden lg:flex flex-col w-[240px] flex-shrink-0 bg-gi-navy">
        {sidebarInner}
      </aside>

      {/* ── Mobile drawer ── */}
      {mobileOpen && (
        <div className="lg:hidden fixed inset-0 z-50 flex">
          <div
            className="absolute inset-0 bg-black/70"
            onClick={() => setMobileOpen(false)}
          />
          <aside className="relative w-[260px] bg-gi-navy flex flex-col z-10 shadow-2xl">
            <button
              onClick={() => setMobileOpen(false)}
              className="absolute top-4 right-3 text-gi-text-subtle hover:text-gi-text p-1.5 rounded-lg hover:bg-gi-navy-hover transition-colors"
              aria-label="Zatvori izbornik"
            >
              <X className="w-4 h-4" />
            </button>
            {sidebarInner}
          </aside>
        </div>
      )}

      {/* ── Main area ── */}
      <div className="flex flex-col flex-1 min-w-0 overflow-hidden">
        {/* ── Top header ── */}
        <header className="h-13 bg-gi-header border-b border-gi-border/60 flex items-center px-4 sm:px-5 gap-3 flex-shrink-0" style={{ height: '52px' }}>
          {/* Mobile hamburger */}
          <button
            className="lg:hidden p-1.5 rounded-lg text-gi-text-subtle hover:text-gi-text hover:bg-gi-navy-hover transition-colors"
            onClick={() => setMobileOpen(true)}
            aria-label="Otvori izbornik"
          >
            <Menu className="w-5 h-5" />
          </button>

          {/* Mobile wordmark */}
          <Link href="/dashboard" className="lg:hidden mr-auto">
            <span className="text-[15px] font-semibold leading-none">
              <span className="text-gi-text">Get </span>
              <span className="text-gi-yellow">Install</span>
            </span>
          </Link>

          {/* Desktop: back arrow + page title */}
          <div className="hidden lg:flex items-center gap-2 flex-1 min-w-0">
            {backHref && (
              <Link
                href={backHref}
                className="p-1.5 rounded-lg text-gi-text-subtle hover:text-gi-text hover:bg-gi-navy-hover transition-colors flex-shrink-0"
              >
                <ChevronLeft className="w-4 h-4" />
              </Link>
            )}
            <h1 className="text-gi-text font-semibold text-[14px] truncate">{title}</h1>
          </div>

          {/* Desktop: action slot + user info + logout */}
          <div className="hidden lg:flex items-center gap-3 ml-auto flex-shrink-0">
            {action}
            <div className="text-right">
              <p className="text-gi-text text-[13px] font-medium leading-none">{displayName}</p>
              <p className="text-gi-text-subtle text-[11px] leading-none mt-0.5">{roleLabel}</p>
            </div>
            <button
              onClick={onLogout}
              className="text-gi-text-muted hover:text-gi-text border border-gi-border hover:border-gi-border/80 px-3 py-[5px] rounded-lg text-[12px] transition-colors"
            >
              Odjava
            </button>
          </div>
        </header>

        {/* Offline banner */}
        {!serverReachable && (
          <div className="bg-amber-900/40 border-b border-amber-700/50 px-4 py-2 text-center flex-shrink-0">
            <p className="text-amber-300 text-[13px]">
              Veza s poslužiteljem nije dostupna. Radite u izvanmrežnom načinu.
            </p>
          </div>
        )}

        {/* Sync banner */}
        <SyncStatusBanner userId={user.id} companyId={user.company_id} />

        {/* ── Scrollable content ── */}
        <main className="flex-1 overflow-y-auto gi-scroll">
          <div className="p-4 sm:p-6 pb-nav lg:pb-6">{children}</div>
        </main>
      </div>

      <MobileBottomNav role={user.role} />
    </div>
  )
}
