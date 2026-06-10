/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  swcMinify: true,
  env: {
    NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api',
  },
  // Proxy /api/* to the local Go backend only in development.
  // Active when NEXT_PUBLIC_API_URL is unset or set to the relative '/api' path.
  // Disabled in production (Vercel) so Next.js doesn't try to proxy to localhost:8080.
  async rewrites() {
    const apiUrl = process.env.NEXT_PUBLIC_API_URL
    const isLocalDev = !apiUrl || apiUrl === '/api' || apiUrl.startsWith('http://localhost')
    if (!isLocalDev) return []
    return [
      {
        source: '/api/:path*',
        destination: 'http://localhost:8080/api/:path*',
      },
    ]
  },
};

module.exports = nextConfig;
