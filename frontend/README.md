# Pulse Frontend

Next.js (App Router) frontend for Pulse.

## Start

```bash
bun install
bun run dev
```

Set env vars:

```bash
NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY=pk_test_...
CLERK_SECRET_KEY=sk_test_...
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
# Optional: Clerk JWT template name for backend bearer token
CLERK_BACKEND_JWT_TEMPLATE=
```

## Build

```bash
bun run build
```

## API Layer

Runtime-validated API helpers live in `lib/api`:

- `lib/api/schemas.ts`: Zod schemas + TypeScript types
- `lib/api/client.ts`: fetch helpers for `/v1/ping` and `/api/v1/*`

Set backend URL with:

```bash
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
```
