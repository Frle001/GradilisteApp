interface LoadingScreenProps {
  text?: string
}

export default function LoadingScreen({ text = 'Učitavanje...' }: LoadingScreenProps) {
  return (
    <div className="min-h-screen bg-slate-950 flex flex-col items-center justify-center gap-3">
      <div
        className="w-8 h-8 rounded-full border-2 border-slate-700 border-t-slate-300 animate-spin"
        aria-hidden="true"
      />
      <p className="text-slate-400 text-sm">{text}</p>
    </div>
  )
}
