'use client'

export default function ReloadButton() {
  return (
    <button
      onClick={() => window.location.reload()}
      className="w-full py-3 px-6 bg-white text-slate-900 font-semibold rounded-lg text-sm hover:bg-slate-100 active:bg-slate-200 transition"
    >
      Pokušaj ponovo
    </button>
  )
}
