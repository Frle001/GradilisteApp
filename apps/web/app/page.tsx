import Link from 'next/link'

export default function Home() {
  return (
    <main className="min-h-screen bg-slate-950 flex items-center justify-center px-4">
      <div className="w-full max-w-sm text-center">
        <div className="mb-8">
          <h1 className="text-4xl font-bold text-white tracking-tight mb-2">Gradilište</h1>
          <p className="text-slate-400 text-sm leading-relaxed">
            Upravljanje projektima, zaposlenicima i materijalima za građevinska poduzeća.
          </p>
        </div>

        <Link
          href="/login"
          className="block w-full py-3 px-6 bg-white text-slate-900 font-semibold rounded-lg text-sm hover:bg-slate-100 active:bg-slate-200 transition"
        >
          Prijava
        </Link>
      </div>
    </main>
  )
}
