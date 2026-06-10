interface LoadingOverlayProps {
  text?: string
  minHeight?: string // e.g. 'min-h-[120px]'
}

// Renders a centered spinner inside whatever container wraps it.
// Use this instead of a bare "Učitavanje..." string while data is fetching.
export default function LoadingOverlay({ text = 'Učitavanje...', minHeight = 'min-h-[120px]' }: LoadingOverlayProps) {
  return (
    <div className={`flex flex-col items-center justify-center gap-2 py-12 ${minHeight}`}>
      <div
        className="w-6 h-6 rounded-full border-2 border-slate-700 border-t-slate-400 animate-spin"
        aria-hidden="true"
      />
      <p className="text-slate-500 text-sm">{text}</p>
    </div>
  )
}
