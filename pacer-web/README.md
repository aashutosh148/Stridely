# pacer-web

Next.js frontend for Pacer.

## Features

- Auth/login and app shell
- Dashboard with readiness + fitness views
- Training plan calendar
- Chat UI with SSE stream rendering and tool activity indicator
- Race strategy planner with GPX upload
- Stats dashboard with charts and semantic facts
- Real-time notifications via SSE

## Prerequisites

- Node `20.9+` (required by Next 16)
- npm `10+`
- Backend running at `http://localhost:3001`
- `just` (optional)

## Environment

```bash
cp .env.local.example .env.local
```

Set:

- `NEXT_PUBLIC_API_URL=http://localhost:3001/api/v1`

## Run

```bash
just dev
```

Open `http://localhost:3000`.

## Quality

```bash
just lint
```

```bash
just build
```

If build fails due to Node version, upgrade Node first.
