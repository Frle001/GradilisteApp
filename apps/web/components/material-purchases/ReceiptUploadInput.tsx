'use client'

import { useRef } from 'react'

interface Props {
  file: File | null
  onChange: (file: File | null) => void
  disabled?: boolean
}

const ALLOWED_TYPES = ['image/jpeg', 'image/png', 'image/webp', 'application/pdf']
const MAX_SIZE_MB = 10

export default function ReceiptUploadInput({ file, onChange, disabled }: Props) {
  const inputRef = useRef<HTMLInputElement>(null)
  const cameraRef = useRef<HTMLInputElement>(null)

  function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    const selected = e.target.files?.[0] ?? null
    if (!selected) {
      onChange(null)
      return
    }
    if (!ALLOWED_TYPES.includes(selected.type)) {
      alert('Nepodržani format datoteke. Dopušteni: JPEG, PNG, WEBP, PDF.')
      e.target.value = ''
      return
    }
    if (selected.size > MAX_SIZE_MB * 1024 * 1024) {
      alert(`Datoteka je prevelika (max ${MAX_SIZE_MB} MB).`)
      e.target.value = ''
      return
    }
    onChange(selected)
  }

  function handleRemove() {
    onChange(null)
    if (inputRef.current) inputRef.current.value = ''
    if (cameraRef.current) cameraRef.current.value = ''
  }

  return (
    <div className="flex flex-col gap-2">
      {!file ? (
        <>
          {/* Main upload area */}
          <label
            className={`flex flex-col items-center gap-2 border-2 border-dashed border-slate-600 rounded-xl px-4 py-6 cursor-pointer hover:border-slate-400 hover:bg-slate-800/40 transition-colors ${disabled ? 'opacity-40 cursor-not-allowed' : ''}`}
          >
            <svg className="w-8 h-8 text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5" />
            </svg>
            <span className="text-sm text-slate-300 font-medium">Odaberi datoteku</span>
            <span className="text-xs text-slate-500">JPEG, PNG, WEBP, PDF · max 10 MB</span>
            <input
              ref={inputRef}
              type="file"
              accept="image/*,.pdf"
              className="sr-only"
              disabled={disabled}
              onChange={handleChange}
            />
          </label>

          {/* Camera shortcut — visible on touch devices */}
          <button
            type="button"
            disabled={disabled}
            onClick={() => cameraRef.current?.click()}
            className={`sm:hidden flex items-center justify-center gap-2 border border-slate-700 rounded-xl px-4 py-3 text-sm text-slate-300 hover:border-slate-500 hover:bg-slate-800/40 transition-colors disabled:opacity-40`}
          >
            <svg className="w-5 h-5 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6.827 6.175A2.31 2.31 0 015.186 7.23c-.38.054-.757.112-1.134.175C2.999 7.58 2.25 8.507 2.25 9.574V18a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9.574c0-1.067-.75-1.994-1.802-2.169a47.865 47.865 0 00-1.134-.175 2.31 2.31 0 01-1.64-1.055l-.822-1.316a2.192 2.192 0 00-1.736-1.039 48.776 48.776 0 00-5.232 0 2.192 2.192 0 00-1.736 1.039l-.821 1.316z" />
              <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 12.75a4.5 4.5 0 11-9 0 4.5 4.5 0 019 0zM18.75 10.5h.008v.008h-.008V10.5z" />
            </svg>
            Fotografiraj račun
          </button>

          {/* Hidden camera input (mobile only — capture="environment") */}
          <input
            ref={cameraRef}
            type="file"
            accept="image/*"
            capture="environment"
            className="sr-only"
            disabled={disabled}
            onChange={handleChange}
          />
        </>
      ) : (
        <div className="flex items-center gap-3 bg-slate-800 border border-slate-700 rounded-xl px-4 py-3">
          <div className="flex-shrink-0 text-emerald-400">
            {file.type === 'application/pdf' ? (
              <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" />
              </svg>
            ) : (
              <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909m-18 3.75h16.5a1.5 1.5 0 001.5-1.5V6a1.5 1.5 0 00-1.5-1.5H3.75A1.5 1.5 0 002.25 6v12a1.5 1.5 0 001.5 1.5zm10.5-11.25h.008v.008h-.008V8.25zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0z" />
              </svg>
            )}
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-sm text-slate-200 font-medium truncate">{file.name}</p>
            <p className="text-xs text-slate-500">{(file.size / 1024).toFixed(0)} KB</p>
          </div>
          <button
            type="button"
            onClick={handleRemove}
            disabled={disabled}
            className="flex-shrink-0 text-slate-400 hover:text-red-400 transition-colors disabled:opacity-40"
            aria-label="Ukloni datoteku"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      )}
    </div>
  )
}
