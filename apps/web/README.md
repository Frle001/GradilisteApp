# Gradilište App - Frontend

Next.js frontend for the construction company management application.

## Stack

- **Next.js 14** — React framework with App Router
- **TypeScript** — Static type checking
- **Tailwind CSS** — Utility-first styling
- **shadcn/ui** — High-quality UI components (ready to add)
- **Axios** — HTTP client for API calls
- **React Hook Form** — Form management

## Setup

```bash
npm install
npm run dev
```

Open [http://localhost:3000](http://localhost:3000)

## Folder Structure

```
app/                    # Next.js app router
├── layout.tsx         # Root layout
├── page.tsx           # Home page
├── globals.css        # Global styles
└── [future modules]/

components/            # Reusable React components
├── ui/               # shadcn/ui components (auto-installed)
└── [custom]/

lib/                   # Utilities and helpers
├── api-client.ts     # Axios instance with interceptors
└── [utilities]/

public/                # Static assets
```

## Environment Variables

```bash
NEXT_PUBLIC_API_URL=http://localhost:8080/api
```

## API Communication

All API calls go through `lib/api-client.ts`, which is pre-configured with:
- Base URL from env var
- Automatic error handling
- Token injection (prepared for auth)

Example:
```typescript
import apiClient from '@/lib/api-client'

const response = await apiClient.get('/health')
```

## Adding shadcn/ui Components

```bash
npx shadcn-ui@latest add button
```

## Future

- Auth pages (login, register)
- Dashboard layouts
- Role-based UI
- Module-specific pages
