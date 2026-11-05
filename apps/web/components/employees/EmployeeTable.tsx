'use client'

import Link from 'next/link'
import { type Employee, fullName } from '@/lib/types/employees'
import { RoleBadge, StatusBadge } from './EmployeeBadge'

interface EmployeeTableProps {
  employees: Employee[]
  canManage: boolean
  onToggleActive: (id: string, active: boolean) => void
}

export default function EmployeeTable({ employees, canManage, onToggleActive }: EmployeeTableProps) {
  if (employees.length === 0) {
    return (
      <div className="bg-gi-card border border-gi-border rounded-xl px-4 py-10 text-center">
        <p className="text-gi-text-subtle text-sm">Nema zaposlenika za prikaz.</p>
      </div>
    )
  }

  return (
    <div className="bg-gi-card border border-gi-border rounded-xl overflow-hidden">
      {/* Desktop table */}
      <table className="hidden sm:table w-full text-sm">
        <thead>
          <tr className="border-b border-gi-border">
            <th className="text-left px-5 py-3 text-xs font-medium text-gi-text-subtle uppercase tracking-wider">Ime i prezime</th>
            <th className="text-left px-5 py-3 text-xs font-medium text-gi-text-subtle uppercase tracking-wider">Uloga</th>
            <th className="text-left px-5 py-3 text-xs font-medium text-gi-text-subtle uppercase tracking-wider">Kontakt</th>
            <th className="text-left px-5 py-3 text-xs font-medium text-gi-text-subtle uppercase tracking-wider">Status</th>
            <th className="text-right px-5 py-3 text-xs font-medium text-gi-text-subtle uppercase tracking-wider">Akcije</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gi-border/60">
          {employees.map((emp) => (
            <tr key={emp.id} className="hover:bg-gi-card-elevated/40 transition-colors">
              <td className="px-5 py-3.5">
                <Link
                  href={`/dashboard/employees/${emp.id}`}
                  className="font-medium text-gi-text hover:text-gi-yellow transition-colors"
                >
                  {fullName(emp)}
                </Link>
              </td>
              <td className="px-5 py-3.5">
                <RoleBadge role={emp.role} />
              </td>
              <td className="px-5 py-3.5 text-gi-text-muted text-xs">
                {emp.email ?? emp.phone ?? '—'}
              </td>
              <td className="px-5 py-3.5">
                <StatusBadge active={emp.active} />
              </td>
              <td className="px-5 py-3.5">
                <div className="flex items-center justify-end gap-2">
                  {canManage ? (
                    <>
                      <Link
                        href={`/dashboard/employees/${emp.id}/edit`}
                        className="text-xs text-gi-text-muted hover:text-gi-text px-2.5 py-1 rounded border border-gi-border hover:border-gi-border/80 bg-gi-card-elevated hover:bg-gi-border transition-colors"
                      >
                        Uredi
                      </Link>
                      <button
                        onClick={() => onToggleActive(emp.id, emp.active)}
                        className={`text-xs px-2.5 py-1 rounded border transition-colors ${
                          emp.active
                            ? 'border-red-900/50 text-red-400/80 hover:bg-red-950/30 hover:text-red-400'
                            : 'border-green-900/50 text-green-400/80 hover:bg-green-950/30 hover:text-green-400'
                        }`}
                      >
                        {emp.active ? 'Deaktiviraj' : 'Aktiviraj'}
                      </button>
                    </>
                  ) : (
                    <Link
                      href={`/dashboard/employees/${emp.id}`}
                      className="text-xs text-gi-text-muted hover:text-gi-yellow transition-colors"
                    >
                      Detalji →
                    </Link>
                  )}
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {/* Mobile cards */}
      <div className="sm:hidden divide-y divide-gi-border/60">
        {employees.map((emp) => (
          <Link
            key={emp.id}
            href={`/dashboard/employees/${emp.id}`}
            className="block p-4 hover:bg-gi-card-elevated/40 transition-colors"
          >
            <div className="flex items-start justify-between gap-2 mb-2">
              <span className="font-medium text-gi-text">{fullName(emp)}</span>
              <StatusBadge active={emp.active} />
            </div>
            <div className="flex items-center gap-2 flex-wrap mb-2">
              <RoleBadge role={emp.role} />
              {emp.email && <span className="text-gi-text-subtle text-xs">{emp.email}</span>}
            </div>
            {canManage && (
              <div className="flex items-center gap-2 pt-1" onClick={(e) => e.preventDefault()}>
                <Link
                  href={`/dashboard/employees/${emp.id}/edit`}
                  onClick={(e) => e.stopPropagation()}
                  className="text-xs text-gi-text-muted hover:text-gi-text px-2.5 py-1 rounded border border-gi-border bg-gi-card-elevated transition-colors"
                >
                  Uredi
                </Link>
                <button
                  onClick={(e) => { e.preventDefault(); e.stopPropagation(); onToggleActive(emp.id, emp.active) }}
                  className={`text-xs px-2.5 py-1 rounded border transition-colors ${
                    emp.active
                      ? 'border-red-900/50 text-red-400/80 hover:bg-red-950/30'
                      : 'border-green-900/50 text-green-400/80 hover:bg-green-950/30'
                  }`}
                >
                  {emp.active ? 'Deaktiviraj' : 'Aktiviraj'}
                </button>
              </div>
            )}
          </Link>
        ))}
      </div>
    </div>
  )
}
