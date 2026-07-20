'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import apiClient from '@/lib/api-client'
import {
  getFilesFromDataTransfer,
  getFilesFromFileList,
  type UploadEntry,
} from '@/lib/upload-path'

// ── Types ─────────────────────────────────────────────────────────────────────

interface ProjectDocument {
  id: string
  original_name: string
  content_type: string
  file_size: number
  uploaded_by_email: string
  created_at: string
  folder_id?: string | null
}

interface ProjectFolder {
  id: string
  name: string
  parent_id?: string | null
  created_at: string
}

interface BreadcrumbItem {
  id: string | null
  name: string
}

interface QueueItem {
  key: string
  relativePath: string
  // 'pending' = batch request in-flight (all items share one request, so all stay pending together)
  // Final statuses come from the server response after the single batch completes.
  status: 'pending' | 'uploaded' | 'skipped' | 'rejected' | 'failed'
  error?: string
}

interface BatchResult {
  relative_path: string
  status: string
  error?: string
}

interface Props {
  projectId: string
  canManage: boolean
}

// ── Component ─────────────────────────────────────────────────────────────────

export default function ProjectDocuments({ projectId, canManage }: Props) {
  const [folders, setFolders] = useState<ProjectFolder[]>([])
  const [docs, setDocs] = useState<ProjectDocument[]>([])
  const [breadcrumb, setBreadcrumb] = useState<BreadcrumbItem[]>([
    { id: null, name: 'Dokumenti' },
  ])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [isDragOver, setIsDragOver] = useState(false)
  const [queue, setQueue] = useState<QueueItem[]>([])
  const [isUploading, setIsUploading] = useState(false)
  const [downloading, setDownloading] = useState<string | null>(null)
  const [deleting, setDeleting] = useState<string | null>(null)
  const [deletingFolder, setDeletingFolder] = useState(false)

  const fileInputRef = useRef<HTMLInputElement>(null)
  const folderInputRef = useRef<HTMLInputElement>(null)
  const dragCounter = useRef(0)

  const currentFolderId = breadcrumb[breadcrumb.length - 1].id

  // ── Data fetching ─────────────────────────────────────────────────────────

  const fetchContent = useCallback(
    async (folderId: string | null) => {
      setLoading(true)
      setError(null)
      try {
        const folderParam = folderId ? `?parent_id=${folderId}` : ''
        const docParam = folderId ? `?folder_id=${folderId}` : ''
        const [fRes, dRes] = await Promise.all([
          apiClient.get(`/projects/${projectId}/folders${folderParam}`),
          apiClient.get(`/projects/${projectId}/documents${docParam}`),
        ])
        setFolders(fRes.data.folders ?? [])
        setDocs(dRes.data.documents ?? [])
      } catch {
        setError('Greška pri učitavanju sadržaja.')
      } finally {
        setLoading(false)
      }
    },
    [projectId],
  )

  useEffect(() => {
    fetchContent(currentFolderId)
  }, [fetchContent, currentFolderId])

  // ── Navigation ────────────────────────────────────────────────────────────

  function openFolder(folder: ProjectFolder) {
    setBreadcrumb(prev => [...prev, { id: folder.id, name: folder.name }])
  }

  function navigateTo(index: number) {
    setBreadcrumb(prev => prev.slice(0, index + 1))
  }

  // ── Batch upload ──────────────────────────────────────────────────────────

  async function startUpload(entries: UploadEntry[]) {
    if (entries.length === 0) return

    const items: QueueItem[] = entries.map((e, i) => ({
      key: `${Date.now()}-${i}`,
      relativePath: e.relativePath,
      status: 'pending',
    }))
    setQueue(items)
    setIsUploading(true)

    try {
      const form = new FormData()
      for (const e of entries) form.append('files', e.file)
      for (const e of entries) form.append('relative_paths', e.relativePath)
      if (currentFolderId) form.append('root_folder_id', currentFolderId)

      const res = await apiClient.post(
        `/projects/${projectId}/documents/folder-upload`,
        form,
        { headers: { 'Content-Type': 'multipart/form-data' } },
      )

      const results: BatchResult[] = res.data.results ?? []
      setQueue(
        items.map((item, i) => ({
          ...item,
          status: (results[i]?.status ?? 'failed') as QueueItem['status'],
          error: results[i]?.error,
        })),
      )

      await fetchContent(currentFolderId)
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      const msg = e?.response?.data?.error ?? 'Greška pri uploadu.'
      setQueue(items.map(item => ({ ...item, status: 'failed', error: msg })))
    } finally {
      setIsUploading(false)
    }
  }

  async function handleFileInput(e: React.ChangeEvent<HTMLInputElement>) {
    if (!e.target.files?.length) return
    const entries = getFilesFromFileList(e.target.files)
    e.target.value = ''
    await startUpload(entries)
  }

  // ── Drag and drop ─────────────────────────────────────────────────────────

  function onDragEnter(e: React.DragEvent) {
    e.preventDefault()
    dragCounter.current++
    setIsDragOver(true)
  }
  function onDragLeave(e: React.DragEvent) {
    e.preventDefault()
    dragCounter.current--
    if (dragCounter.current === 0) setIsDragOver(false)
  }
  function onDragOver(e: React.DragEvent) {
    e.preventDefault()
  }
  async function onDrop(e: React.DragEvent) {
    e.preventDefault()
    dragCounter.current = 0
    setIsDragOver(false)
    if (!canManage) return
    const entries = await getFilesFromDataTransfer(e.dataTransfer)
    await startUpload(entries)
  }

  // ── Folder deletion ───────────────────────────────────────────────────────

  async function handleDeleteFolder() {
    if (!currentFolderId) return
    const currentItem = breadcrumb[breadcrumb.length - 1]
    const confirmed = confirm(
      `Obrisati mapu "${currentItem.name}"?\n\nMapa mora biti prazna. Ova radnja se ne može poništiti.`,
    )
    if (!confirmed) return

    setDeletingFolder(true)
    try {
      await apiClient.delete(`/projects/${projectId}/folders/${currentFolderId}`)
      // Navigate to parent — useEffect will refresh parent content automatically.
      setBreadcrumb(prev => prev.slice(0, -1))
    } catch (err: unknown) {
      const e = err as { response?: { status?: number; data?: { error?: string } } }
      if (e?.response?.status === 409 && e?.response?.data?.error) {
        alert(e.response.data.error)
      } else {
        alert('Greška pri brisanju mape.')
      }
    } finally {
      setDeletingFolder(false)
    }
  }

  // ── Delete / Download ─────────────────────────────────────────────────────

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

  // ── Render ────────────────────────────────────────────────────────────────

  const hasContent = folders.length > 0 || docs.length > 0

  return (
    <div
      className={`relative bg-slate-900 border rounded-xl p-6 transition-colors ${
        isDragOver ? 'border-blue-500 bg-slate-800/80' : 'border-slate-800'
      }`}
      onDragEnter={onDragEnter}
      onDragLeave={onDragLeave}
      onDragOver={onDragOver}
      onDrop={onDrop}
    >
      {/* Header */}
      <div className="flex items-center justify-between gap-4 mb-4">
        <h2 className="text-white font-semibold">Dokumenti projekta</h2>
        <div className="flex items-center gap-2">
          {canManage && currentFolderId && (
            <button
              onClick={handleDeleteFolder}
              disabled={deletingFolder || isUploading}
              title="Obriši mapu"
              className="p-1.5 rounded text-slate-500 hover:text-red-400 hover:bg-slate-700 transition disabled:opacity-40"
            >
              <svg
                className="w-4 h-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                />
              </svg>
            </button>
          )}
          {canManage && (
            <>
              <label
                className={`cursor-pointer px-3 py-1.5 rounded-lg text-xs font-medium transition ${
                  isUploading
                    ? 'bg-slate-700 text-slate-400 cursor-not-allowed'
                    : 'bg-blue-600 hover:bg-blue-500 text-white'
                }`}
              >
                + Dodaj datoteke
                <input
                  ref={fileInputRef}
                  type="file"
                  multiple
                  className="hidden"
                  disabled={isUploading}
                  onChange={handleFileInput}
                  accept=".pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.jpg,.jpeg,.png,.webp,.gif,.tiff,.txt,.csv,.zip,.dwg"
                />
              </label>
              <label
                className={`cursor-pointer px-3 py-1.5 rounded-lg text-xs font-medium transition ${
                  isUploading
                    ? 'bg-slate-700 text-slate-400 cursor-not-allowed'
                    : 'bg-slate-700 hover:bg-slate-600 text-white'
                }`}
              >
                + Dodaj mapu
                <input
                  ref={folderInputRef}
                  type="file"
                  multiple
                  className="hidden"
                  disabled={isUploading}
                  onChange={handleFileInput}
                  {...({ webkitdirectory: '', directory: '' } as object)}
                />
              </label>
            </>
          )}
        </div>
      </div>

      {/* Breadcrumb */}
      {breadcrumb.length > 1 && (
        <nav className="flex items-center gap-1 mb-4 text-xs">
          {breadcrumb.map((crumb, i) => (
            <span key={i} className="flex items-center gap-1">
              {i > 0 && <span className="text-slate-600">/</span>}
              {i < breadcrumb.length - 1 ? (
                <button
                  onClick={() => navigateTo(i)}
                  className="text-blue-400 hover:text-blue-300 transition"
                >
                  {crumb.name}
                </button>
              ) : (
                <span className="text-white font-medium">{crumb.name}</span>
              )}
            </span>
          ))}
        </nav>
      )}

      {/* Drag overlay */}
      {isDragOver && canManage && (
        <div className="absolute inset-0 flex items-center justify-center pointer-events-none rounded-xl z-10">
          <p className="text-blue-300 text-sm font-medium bg-slate-900/90 border border-blue-500 px-4 py-2 rounded-lg">
            Ispusti datoteke ovdje
          </p>
        </div>
      )}

      {/* Upload queue */}
      {queue.length > 0 && (
        <div className="mb-4 bg-slate-800 border border-slate-700 rounded-lg p-3 space-y-1">
          <p className="text-xs text-slate-400 font-medium mb-2">Upload red</p>
          {queue.map(item => (
            <div key={item.key} className="flex items-center gap-2 text-xs">
              <QueueStatusIcon status={item.status} />
              <span className="flex-1 min-w-0 truncate text-slate-300">
                {item.relativePath}
              </span>
              {item.error && (
                <span className="text-red-400 truncate max-w-[180px]" title={item.error}>
                  {item.error}
                </span>
              )}
            </div>
          ))}
          {!isUploading && (
            <button
              className="text-xs text-slate-500 hover:text-slate-300 mt-1 transition"
              onClick={() => setQueue([])}
            >
              Zatvori
            </button>
          )}
        </div>
      )}

      {/* Content */}
      {loading ? (
        <p className="text-slate-500 text-sm">Učitavanje…</p>
      ) : error ? (
        <p className="text-red-400 text-sm">{error}</p>
      ) : !hasContent ? (
        <div className="text-center py-8">
          <p className="text-slate-500 text-sm italic">Nema dokumenata.</p>
          {canManage && (
            <p className="text-slate-600 text-xs mt-1">
              Povuci datoteke ovdje ili klikni &quot;Dodaj datoteke&quot;.
            </p>
          )}
        </div>
      ) : (
        <div className="space-y-1">
          {folders.map(folder => (
            <button
              key={folder.id}
              onClick={() => openFolder(folder)}
              className="w-full flex items-center gap-3 bg-slate-800/50 border border-slate-700 rounded-lg px-3 py-2.5 hover:bg-slate-800 transition text-left"
            >
              <span className="text-yellow-400 flex-shrink-0">
                <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M2 6a2 2 0 012-2h4l2 2h6a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
                </svg>
              </span>
              <span className="text-white text-sm font-medium flex-1 min-w-0 truncate">
                {folder.name}
              </span>
              <svg
                className="w-3.5 h-3.5 text-slate-500 flex-shrink-0"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M9 5l7 7-7 7"
                />
              </svg>
            </button>
          ))}

          {docs.map(doc => (
            <div
              key={doc.id}
              className="flex items-center gap-3 bg-slate-800/50 border border-slate-700 rounded-lg px-3 py-2.5"
            >
              <FileTypeBadge contentType={doc.content_type} />
              <div className="flex-1 min-w-0">
                <p className="text-white text-sm font-medium truncate">
                  {doc.original_name}
                </p>
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
                    <svg
                      className="w-4 h-4 animate-spin"
                      fill="none"
                      viewBox="0 0 24 24"
                    >
                      <circle
                        className="opacity-25"
                        cx="12"
                        cy="12"
                        r="10"
                        stroke="currentColor"
                        strokeWidth="4"
                      />
                      <path
                        className="opacity-75"
                        fill="currentColor"
                        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
                      />
                    </svg>
                  ) : (
                    <svg
                      className="w-4 h-4"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                      strokeWidth={2}
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
                      />
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
                    <svg
                      className="w-4 h-4"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                      strokeWidth={2}
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                      />
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

// ── Sub-components ────────────────────────────────────────────────────────────

function QueueStatusIcon({ status }: { status: QueueItem['status'] }) {
  if (status === 'pending') {
    return (
      <svg
        className="w-3.5 h-3.5 text-blue-400 animate-spin flex-shrink-0"
        fill="none"
        viewBox="0 0 24 24"
      >
        <circle
          className="opacity-25"
          cx="12"
          cy="12"
          r="10"
          stroke="currentColor"
          strokeWidth="4"
        />
        <path
          className="opacity-75"
          fill="currentColor"
          d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
        />
      </svg>
    )
  }
  if (status === 'uploaded') {
    return (
      <svg
        className="w-3.5 h-3.5 text-green-400 flex-shrink-0"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        strokeWidth={2}
      >
        <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
      </svg>
    )
  }
  if (status === 'skipped') {
    return (
      <svg
        className="w-3.5 h-3.5 text-slate-500 flex-shrink-0"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        strokeWidth={2}
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          d="M13 5l7 7-7 7M5 5l7 7-7 7"
        />
      </svg>
    )
  }
  return (
    <svg
      className="w-3.5 h-3.5 text-red-400 flex-shrink-0"
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
      strokeWidth={2}
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M6 18L18 6M6 6l12 12"
      />
    </svg>
  )
}

function FileTypeBadge({ contentType }: { contentType: string }) {
  const { label, className } = resolveType(contentType)
  return (
    <span
      className={`flex-shrink-0 text-[10px] font-bold px-1.5 py-0.5 rounded uppercase tracking-wide ${className}`}
    >
      {label}
    </span>
  )
}

function resolveType(ct: string): { label: string; className: string } {
  if (ct.includes('pdf'))
    return { label: 'PDF', className: 'bg-red-900/60 text-red-300' }
  if (ct.includes('word') || ct.includes('msword'))
    return { label: 'Word', className: 'bg-blue-900/60 text-blue-300' }
  if (ct.includes('excel') || ct.includes('spreadsheet'))
    return { label: 'Excel', className: 'bg-green-900/60 text-green-300' }
  if (ct.includes('powerpoint') || ct.includes('presentation'))
    return { label: 'PPT', className: 'bg-orange-900/60 text-orange-300' }
  if (ct.startsWith('image/'))
    return { label: 'Slika', className: 'bg-purple-900/60 text-purple-300' }
  if (ct.includes('csv') || ct.includes('text/plain'))
    return { label: 'TXT', className: 'bg-slate-700 text-slate-300' }
  if (ct.includes('zip'))
    return { label: 'ZIP', className: 'bg-yellow-900/60 text-yellow-300' }
  if (
    ct.includes('dwg') ||
    ct.includes('acad') ||
    ct.includes('autocad')
  )
    return { label: 'DWG', className: 'bg-cyan-900/60 text-cyan-300' }
  return { label: 'Dat.', className: 'bg-slate-700 text-slate-400' }
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('hr-HR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  })
}
