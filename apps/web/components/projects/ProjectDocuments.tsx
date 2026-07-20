'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import apiClient from '@/lib/api-client'

interface ProjectDocument {
  id: string
  original_name: string
  content_type: string
  file_size: number
  uploaded_by_email: string
  created_at: string
}

interface Props {
  projectId: string
  canManage: boolean
}

export default function ProjectDocuments({ projectId, canManage }: Props) {
  const [docs, setDocs] = useState<ProjectDocument[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)
  const [uploadError, setUploadError] = useState<string | null>(null)
  const [downloading, setDownloading] = useState<string | null>(null)
  const [deleting, setDeleting] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const fetchDocs = useCallback(async () => {
    try {
      const res = await apiClient.get(`/projects/${projectId}/documents`)
      setDocs(res.data.documents ?? [])
      setError(null)
    } catch {
      setError('Greška pri učitavanju dokumenata.')
    } finally {
      setLoading(false)
    }
  }, [projectId])

  useEffect(() => { fetchDocs() }, [fetchDocs])

  async function handleUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    setUploadError(null)
    setUploading(true)

    const form = new FormData()
    form.append('file', file)
    try {
      await apiClient.post(`/projects/${projectId}/documents`, form, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      await fetchDocs()
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setUploadError(e?.response?.data?.error ?? 'Greška pri uploadu datoteke.')
    } finally {
      setUploading(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  async function handleDownload(doc: ProjectDocument) {
    setDownloading(doc.id)
    try {
      const res = await apiClient.get(
        `/projects/${projectId}/documents/${doc.id}/download`,
        { responseType: 'blob' },
      )
      const blob = new Blob([res.data as BlobPart], { type: doc.content_type })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = doc.original_name
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
    } catch {
      alert('Greška pri preuzimanju datoteke.')
    } finally {
      setDownloading(null)
    }
  }

  async function handleDelete(doc: ProjectDocument) {
    if (!confirm(`Obrisati dokument "${doc.original_name}"?`)) return
    setDeleting(doc.id)
    try {
      await apiClient.delete(`/projects/${projectId}/documents/${doc.id}`)
      setDocs(prev => prev.filter(d => d.id !== doc.id))
    } catch {
      alert('Greška pri brisanju dokumenta.')
    } finally {
      setDeleting(null)
    }
  }

  return (
    <div className="bg-slate-900 border border-slate-800 rounded-xl p-6">
      <div className="flex items-center justify-between gap-4 mb-4">
        <h2 className="text-white font-semibold">Dokumenti projekta</h2>
        {canManage && (
          <label className={`cursor-pointer px-3 py-1.5 rounded-lg text-xs font-medium transition ${
            uploading
              ? 'bg-slate-700 text-slate-400 cursor-not-allowed'
              : 'bg-blue-600 hover:bg-blue-500 text-white'
          }`}>
            {uploading ? 'Uploading…' : '+ Dodaj datoteku'}
            <input
              ref={fileInputRef}
              type="file"
              className="hidden"
              disabled={uploading}
              onChange={handleUpload}
              accept=".pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.jpg,.jpeg,.png,.webp,.gif,.tiff,.txt,.csv,.zip,.dwg"
            />
          </label>
        )}
      </div>

      {uploadError && (
        <p className="text-red-400 text-xs mb-3">{uploadError}</p>
      )}

      {loading ? (
        <p className="text-slate-500 text-sm">Učitavanje…</p>
      ) : error ? (
        <p className="text-red-400 text-sm">{error}</p>
      ) : docs.length === 0 ? (
        <p className="text-slate-500 text-sm italic">Nema dokumenata za ovaj projekt.</p>
      ) : (
        <div className="space-y-2">
          {docs.map(doc => (
            <div
              key={doc.id}
              className="flex items-center gap-3 bg-slate-800/50 border border-slate-700 rounded-lg px-3 py-2.5"
            >
              <FileTypeBadge contentType={doc.content_type} />

              <div className="flex-1 min-w-0">
                <p className="text-white text-sm font-medium truncate">{doc.original_name}</p>
                <p className="text-slate-500 text-xs mt-0.5">
                  {formatBytes(doc.file_size)} · {formatDate(doc.created_at)}
                  {doc.uploaded_by_email ? ` · ${doc.uploaded_by_email}` : ''}
                </p>
              </div>

              <div className="flex items-center gap-1 flex-shrink-0">
                <button
                  onClick={() => handleDownload(doc)}
                  disabled={downloading === doc.id}
                  title="Preuzmi"
                  className="p-1.5 rounded text-slate-400 hover:text-white hover:bg-slate-700 transition disabled:opacity-40"
                >
                  {downloading === doc.id ? (
                    <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                    </svg>
                  ) : (
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                    </svg>
                  )}
                </button>

                {canManage && (
                  <button
                    onClick={() => handleDelete(doc)}
                    disabled={deleting === doc.id}
                    title="Obriši"
                    className="p-1.5 rounded text-slate-500 hover:text-red-400 hover:bg-slate-700 transition disabled:opacity-40"
                  >
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function FileTypeBadge({ contentType }: { contentType: string }) {
  const { label, className } = resolveType(contentType)
  return (
    <span className={`flex-shrink-0 text-[10px] font-bold px-1.5 py-0.5 rounded uppercase tracking-wide ${className}`}>
      {label}
    </span>
  )
}

function resolveType(ct: string): { label: string; className: string } {
  if (ct.includes('pdf'))          return { label: 'PDF',   className: 'bg-red-900/60 text-red-300' }
  if (ct.includes('word') || ct.includes('msword')) return { label: 'Word', className: 'bg-blue-900/60 text-blue-300' }
  if (ct.includes('excel') || ct.includes('spreadsheet')) return { label: 'Excel', className: 'bg-green-900/60 text-green-300' }
  if (ct.includes('powerpoint') || ct.includes('presentation')) return { label: 'PPT', className: 'bg-orange-900/60 text-orange-300' }
  if (ct.startsWith('image/'))     return { label: 'Slika', className: 'bg-purple-900/60 text-purple-300' }
  if (ct.includes('csv') || ct.includes('text/plain')) return { label: 'TXT', className: 'bg-slate-700 text-slate-300' }
  if (ct.includes('zip'))          return { label: 'ZIP',   className: 'bg-yellow-900/60 text-yellow-300' }
  if (ct.includes('dwg') || ct.includes('acad') || ct.includes('autocad')) return { label: 'DWG', className: 'bg-cyan-900/60 text-cyan-300' }
  return { label: 'Dat.', className: 'bg-slate-700 text-slate-400' }
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('hr-HR', { day: '2-digit', month: '2-digit', year: 'numeric' })
}
