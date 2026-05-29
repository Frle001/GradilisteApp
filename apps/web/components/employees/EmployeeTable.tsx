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
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-10 text-center">
        <p className="text-slate-400 text-sm">Nema zaposlenika za prikaz.</p>
      </div>
    )
  }

  return (
    <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
      {/* Desktop table */}
      <table className="hidden sm:table w-full text-sm">
        <thead>
          <tr className="border-b border-slate-800 text-slate-500 text-xs uppercase tracking-wider">
            <th className="text-left px-5 py-3 font-medium">Ime i prezime</th>
            <th className="text-left px-5 py-3 font-medium">Uloga</th>
            <th className="text-left px-5 py-3 font-medium">Kontakt</th>
            <th className="text-left px-5 py-3 font-medium">Status</th>
            <th className="text-right px-5 py-3 font-medium">Akcije</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-800">
          {employees.map((emp) => (
            <tr key={emp.id} className="hover:bg-slate-800/40 transition">
              <td className="px-5 py-3.5">
                <Link
                  href={`/dashboard/employees/${emp.id}`}
                  className="font-medium text-white hover:text-slate-200 transition"
                >
                  {fullName(emp)}
                </Link>
              </td>
              <td className="px-5 py-3.5">
                <RoleBadge role={emp.role} />
              </td>
              <td className="px-5 py-3.5 text-slate-400">
                {emp.email ?? emp.phone ?? '—'}
              </td>
              <td className="px-5 py-3.5">
                <StatusBadge active={emp.active} />
              </td>
              <td className="px-5 py-3.5">
                <div className="flex items-center justify-end gap-3">
                  <Link
                    href={`/dashboard/employees/${emp.id}`}
                    className="text-slate-400 hover:text-white transition text-xs"
                  >
                    Detalji
                  </Link>
                  {canManage && (
                    <>
                      <Link
                        href={`/dashboard/employees/${emp.id}/edit`}
                        className="text-slate-400 hover:text-white transition text-xs"
                      >
                        Uredi
                      </Link>
                      <button
                        onClick={() => onToggleActive(emp.id, emp.active)}
                        className="text-slate-500 hover:text-red-400 transition text-xs"
                      >
                        {emp.active ? 'Deaktiviraj' : 'Aktiviraj'}
                      </button>
                    </>
                  )}
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {/* Mobile cards */}
      <div className="sm:hidden divide-y divide-slate-800">
        {employees.map((emp) => (
          <div key={emp.id} className="p-4 space-y-2">
            <div className="flex items-start justify-between gap-2">
              <Link
                href={`/dashboard/employees/${emp.id}`}
                className="font-medium text-white hover:text-slate-200"
              >
                {fullName(emp)}
              </Link>
              <StatusBadge active={emp.active} />
            </div>
            <div className="flex items-center gap-2 flex-wrap">
              <RoleBadge role={emp.role} />
              {emp.email && <span className="text-slate-400 text-xs">{emp.email}</span>}
            </div>
            <div className="flex items-center gap-3 pt-1">
              <Link
                href={`/dashboard/employees/${emp.id}`}
                className="text-xs text-slate-400 hover:text-white transition"
              >
                Detalji
              </Link>
              {canManage && (
                <>
                  <Link
                    href={`/dashboard/employees/${emp.id}/edit`}
                    className="text-xs text-slate-400 hover:text-white transition"
                  >
                    Uredi
                  </Link>
                  <button
                    onClick={() => onToggleActive(emp.id, emp.active)}
                    className="text-xs text-slate-500 hover:text-red-400 transition"
                  >
                    {emp.active ? 'Deaktiviraj' : 'Aktiviraj'}
                  </button>
                </>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
