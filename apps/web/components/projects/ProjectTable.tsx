'use client'

import Link from 'next/link'
import { type ProjectListItem } from '@/lib/types/projects'
import ProjectStatusBadge from './ProjectStatusBadge'

interface Props {
  projects: ProjectListItem[]
  canManage: boolean
  onClose: (id: string) => void
  onArchive: (id: string) => void
  onReactivate: (id: string) => void
}

export default function ProjectTable({ projects, canManage, onClose, onArchive, onReactivate }: Props) {
  if (projects.length === 0) {
    return (
      <div className="text-center py-14 text-gi-text-subtle text-sm">
        Nema projekata za prikaz.
      </div>
    )
  }

  return (
    <>
      {/* Desktop table */}
      <div className="hidden md:block overflow-x-auto rounded-xl border border-gi-border">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gi-border bg-gi-card">
              <th className="text-left px-4 py-3 text-gi-text-subtle font-medium text-xs uppercase tracking-wider">Projekt</th>
              <th className="text-left px-4 py-3 text-gi-text-subtle font-medium text-xs uppercase tracking-wider">Status</th>
              <th className="text-left px-4 py-3 text-gi-text-subtle font-medium text-xs uppercase tracking-wider">Poslovođa</th>
              <th className="text-left px-4 py-3 text-gi-text-subtle font-medium text-xs uppercase tracking-wider">Početak</th>
              {canManage && <th className="text-right px-4 py-3 text-gi-text-subtle font-medium text-xs uppercase tracking-wider">Akcije</th>}
            </tr>
          </thead>
          <tbody className="divide-y divide-gi-border/60">
            {projects.map((p) => (
              <tr key={p.id} className="bg-gi-card hover:bg-gi-card-elevated transition-colors">
                <td className="px-4 py-3">
                  <Link href={`/dashboard/projects/${p.id}`} className="text-gi-text font-medium hover:text-gi-yellow transition-colors">
                    {p.name}
                  </Link>
                  {p.address && <p className="text-gi-text-subtle text-xs mt-0.5">{p.address}</p>}
                </td>
                <td className="px-4 py-3">
                  <ProjectStatusBadge status={p.status} />
                </td>
                <td className="px-4 py-3 text-gi-text-muted text-sm">
                  {p.primary_poslovoda_name ?? <span className="text-gi-text-subtle italic">Nije dodijeljen</span>}
                </td>
                <td className="px-4 py-3 text-gi-text-subtle text-sm">
                  {p.start_date ?? '—'}
                </td>
                {canManage && (
                  <td className="px-4 py-3 text-right">
                    <div className="flex items-center justify-end gap-3">
                      <Link
                        href={`/dashboard/projects/${p.id}/edit`}
                        className="text-xs text-gi-text-muted hover:text-gi-text transition-colors"
                      >
                        Uredi
                      </Link>
                      {p.status === 'active' && (
                        <button
                          onClick={() => onClose(p.id)}
                          className="text-xs text-amber-400 hover:text-amber-300 transition-colors"
                        >
                          Zatvori
                        </button>
                      )}
                      {p.status !== 'archived' && (
                        <button
                          onClick={() => onArchive(p.id)}
                          className="text-xs text-gi-text-subtle hover:text-gi-text-muted transition-colors"
                        >
                          Arhiviraj
                        </button>
                      )}
                      {p.status !== 'active' && (
                        <button
                          onClick={() => onReactivate(p.id)}
                          className="text-xs text-green-400 hover:text-green-300 transition-colors"
                        >
                          Reaktiviraj
                        </button>
                      )}
                    </div>
                  </td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Mobile cards */}
      <div className="md:hidden space-y-2.5">
        {projects.map((p) => (
          <div key={p.id} className="bg-gi-card border border-gi-border rounded-xl p-4">
            <div className="flex items-start justify-between gap-2 mb-2">
              <Link href={`/dashboard/projects/${p.id}`} className="text-gi-text font-medium hover:text-gi-yellow transition-colors flex-1 min-w-0 truncate">
                {p.name}
              </Link>
              <ProjectStatusBadge status={p.status} />
            </div>
            {p.address && <p className="text-gi-text-subtle text-xs mb-1">{p.address}</p>}
            <p className="text-gi-text-subtle text-xs">
              Poslovođa: {p.primary_poslovoda_name ?? <span className="italic">Nije dodijeljen</span>}
            </p>
            {p.start_date && <p className="text-gi-text-subtle text-xs mt-0.5">Početak: {p.start_date}</p>}
            {canManage && (
              <div className="flex flex-wrap gap-3 mt-3 pt-3 border-t border-gi-border">
                <Link href={`/dashboard/projects/${p.id}/edit`} className="text-xs text-gi-text-muted hover:text-gi-text transition-colors">
                  Uredi
                </Link>
                {p.status === 'active' && (
                  <button onClick={() => onClose(p.id)} className="text-xs text-amber-400 hover:text-amber-300 transition-colors">
                    Zatvori
                  </button>
                )}
                {p.status !== 'archived' && (
                  <button onClick={() => onArchive(p.id)} className="text-xs text-gi-text-subtle hover:text-gi-text-muted transition-colors">
                    Arhiviraj
                  </button>
                )}
                {p.status !== 'active' && (
                  <button onClick={() => onReactivate(p.id)} className="text-xs text-green-400 hover:text-green-300 transition-colors">
                    Reaktiviraj
                  </button>
                )}
              </div>
            )}
          </div>
        ))}
      </div>
    </>
  )
}
