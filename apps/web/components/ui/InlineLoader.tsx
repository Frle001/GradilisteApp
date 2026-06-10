interface InlineLoaderProps {
  text?: string
}

// Tiny inline spinner. Use inside buttons, table cells, or short status lines.
export default function InlineLoader({ text }: InlineLoaderProps) {
  return (
    <span className="inline-flex items-center gap-1.5 text-slate-400 text-sm">
      <span
        className="inline-block w-3.5 h-3.5 rounded-full border-2 border-slate-600 border-t-slate-300 animate-spin"
        aria-hidden="true"
      />
      {text}
    </span>
  )
}
