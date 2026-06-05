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
      <div className="text-center py-12 text-slate-500 text-sm">
        Nema projekata za prikaz.
      </div>
    )
  }

  return (
    <>
      {/* Desktop table */}
      <div className="hidden md:block overflow-x-auto rounded-xl border border-slate-800">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-800 bg-slate-900">
              <th className="text-left px-4 py-3 text-slate-400 font-medium">Projekt</th>
              <th className="text-left px-4 py-3 text-slate-400 font-medium">Status</th>
              <th className="text-left px-4 py-3 text-slate-400 font-medium">Poslovođa</th>
              <th className="text-left px-4 py-3 text-slate-400 font-medium">Datum početka</th>
              {canManage && <th className="text-right px-4 py-3 text-slate-400 font-medium">Akcije</th>}
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-800">
            {projects.map((p) => (
              <tr key={p.id} className="bg-slate-900 hover:bg-slate-800/60 transition">
                <td className="px-4 py-3">
                  <Link href={`/dashboard/projects/${p.id}`} className="text-white font-medium hover:text-slate-300 transition">
                    {p.name}
                  </Link>
                  {p.address && <p className="text-slate-500 text-xs mt-0.5">{p.address}</p>}
                </td>
                <td className="px-4 py-3">
                  <ProjectStatusBadge status={p.status} />
                </td>
                <td className="px-4 py-3 text-slate-300">
                  {p.primary_poslovoda_name ?? <span className="text-slate-600 italic">Nije dodijeljen</span>}
                </td>
                <td className="px-4 py-3 text-slate-400">
                  {p.start_date ?? '—'}
                </td>
                {canManage && (
                  <td className="px-4 py-3 text-right">
                    <div className="flex items-center justify-end gap-2">
                      <Link
                        href={`/dashboard/projects/${p.id}/edit`}
                        className="text-xs text-slate-400 hover:text-white transition"
                      >
                        Uredi
                      </Link>
                      {p.status === 'active' && (
                        <button
                          onClick={() => onClose(p.id)}
                          className="text-xs text-amber-400 hover:text-amber-300 transition"
                        >
                          Zatvori
                        </button>
                      )}
                      {p.status !== 'archived' && (
                        <button
                          onClick={() => onArchive(p.id)}
                          className="text-xs text-slate-500 hover:text-slate-300 transition"
                        >
                          Arhiviraj
                        </button>
                      )}
                      {p.status !== 'active' && (
                        <button
                          onClick={() => onReactivate(p.id)}
                          className="text-xs text-green-400 hover:text-green-300 transition"
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
      <div className="md:hidden space-y-3">
        {projects.map((p) => (
          <div key={p.id} className="bg-slate-900 border border-slate-800 rounded-xl p-4">
            <div className="flex items-start justify-between gap-2 mb-2">
              <Link href={`/dashboard/projects/${p.id}`} className="text-white font-medium hover:text-slate-300 transition">
                {p.name}
              </Link>
              <ProjectStatusBadge status={p.status} />
            </div>
            {p.address && <p className="text-slate-500 text-xs mb-1">{p.address}</p>}
            <p className="text-slate-400 text-xs">
              Poslovođa: {p.primary_poslovoda_name ?? <span className="italic text-slate-600">Nije dodijeljen</span>}
            </p>
            {p.start_date && <p className="text-slate-500 text-xs mt-0.5">Početak: {p.start_date}</p>}
            {canManage && (
              <div className="flex flex-wrap gap-3 mt-3 pt-3 border-t border-slate-800">
                <Link href={`/dashboard/projects/${p.id}/edit`} className="text-xs text-slate-400 hover:text-white transition">
                  Uredi
                </Link>
                {p.status === 'active' && (
                  <button onClick={() => onClose(p.id)} className="text-xs text-amber-400 hover:text-amber-300 transition">
                    Zatvori
                  </button>
                )}
                {p.status !== 'archived' && (
                  <button onClick={() => onArchive(p.id)} className="text-xs text-slate-500 hover:text-slate-300 transition">
                    Arhiviraj
                  </button>
                )}
                {p.status !== 'active' && (
                  <button onClick={() => onReactivate(p.id)} className="text-xs text-green-400 hover:text-green-300 transition">
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
