'use client'

import { useEffect, useState } from 'react'

export default function Home() {
  const [apiStatus, setApiStatus] = useState<string>('checking...')

  useEffect(() => {
    const checkAPI = async () => {
      try {
        const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/health`)
        if (response.ok) {
          setApiStatus('connected')
        } else {
          setApiStatus('disconnected')
        }
      } catch (error) {
        setApiStatus('error')
      }
    }

    checkAPI()
  }, [])

  return (
    <main className="min-h-screen bg-gradient-to-br from-slate-900 to-slate-800 text-white">
      <div className="max-w-4xl mx-auto px-4 py-16">
        <div className="text-center">
          <h1 className="text-5xl font-bold mb-4">Gradilište App</h1>
          <p className="text-xl text-slate-400 mb-12">Construction company management system</p>

          <div className="bg-slate-800 rounded-lg p-8 mb-8">
            <div className="text-sm text-slate-400 mb-2">Backend API Status</div>
            <div className={`text-2xl font-mono font-semibold ${
              apiStatus === 'connected' ? 'text-green-400' :
              apiStatus === 'checking...' ? 'text-yellow-400' :
              'text-red-400'
            }`}>
              {apiStatus}
            </div>
            <div className="text-xs text-slate-500 mt-2">
              {process.env.NEXT_PUBLIC_API_URL}
            </div>
          </div>

          <div className="space-y-4">
            <h2 className="text-2xl font-bold mb-6">Coming Soon</h2>
            <ul className="text-slate-300 space-y-2 text-left max-w-md mx-auto">
              <li>✓ Authentication & Login</li>
              <li>✓ Role-based access control</li>
              <li>✓ Employee management</li>
              <li>✓ Project management (Gradilište)</li>
              <li>✓ Daily reports & timesheets</li>
              <li>✓ Material tracking</li>
              <li>✓ Document management</li>
            </ul>
          </div>
        </div>
      </div>
    </main>
  )
}
