export default function OfflinePage() {
  return (
    <div className="min-h-screen bg-slate-950 flex items-center justify-center px-6">
      <div className="text-center max-w-xs">
        <div className="w-16 h-16 mx-auto mb-6 rounded-full bg-slate-800 border border-slate-700 flex items-center justify-center">
          <svg className="w-8 h-8 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M3 3l18 18M8.111 8.111A5.5 5.5 0 0115.5 15.5M6.343 6.343A8 8 0 0117.657 17.657M2 12C2 6.477 6.477 2 12 2" />
          </svg>
        </div>
        <h1 className="text-xl font-bold text-white mb-2">Nema veze</h1>
        <p className="text-slate-400 text-sm leading-relaxed mb-6">
          Provjerite internetsku vezu i pokušajte ponovo.
        </p>
        <button
          onClick={() => window.location.reload()}
          className="w-full py-3 px-6 bg-white text-slate-900 font-semibold rounded-lg text-sm hover:bg-slate-100 active:bg-slate-200 transition"
        >
          Pokušaj ponovo
        </button>
      </div>
    </div>
  )
}
