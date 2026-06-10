import type { Metadata, Viewport } from 'next'
import './globals.css'
import ServiceWorkerProvider from '@/components/layout/ServiceWorkerProvider'
import OfflineIndicator from '@/components/layout/OfflineIndicator'

export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
  minimumScale: 1,
  maximumScale: 5,
  viewportFit: 'cover',
  themeColor: '#0f172a',
}

export const metadata: Metadata = {
  title: 'Gradilište',
  description: 'Upravljanje projektima, zaposlenicima i materijalima za građevinska poduzeća.',
  manifest: '/manifest.json',
  appleWebApp: {
    capable: true,
    statusBarStyle: 'black-translucent',
    title: 'Gradilište',
  },
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="hr">
      <body>
        <ServiceWorkerProvider />
        <OfflineIndicator />
        {children}
      </body>
    </html>
  )
}
