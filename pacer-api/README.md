# pacer-api

Go/Fiber backend for Pacer.

## Features

- Auth + OAuth callback routes
- Activity ingestion and analytics
- Training plan generation + adjustments
- Chat endpoint with SSE streaming + history
- Race strategy endpoints
- Stats and readiness endpoints
- User event SSE stream (`/api/v1/events/stream`)

## Prerequisites

- Go `1.25+`
- Docker + Docker Compose
- `just` (optional)

## Environment

Use `.env.example` as a template:

```bash
cp .env.example .env
```

Do not keep project-level `.env` files in repo root. Keep backend secrets only in `pacer-api/.env` (local, gitignored).

Required minimum vars:

- `DATABASE_URL`
- `JWT_SECRET`
- `ENCRYPTION_KEY_HEX`

Optional provider vars:

- `ANTHROPIC_API_KEY`
- `OPENAI_API_KEY`
- `GEMINI_API_KEY`
- `STRAVA_CLIENT_ID` / `STRAVA_CLIENT_SECRET`
- `OPENWEATHERMAP_API_KEY`

`OPENWEATHERMAP_API_KEY` belongs in `pacer-api/.env` (or container env), not in a root `.env`.

## Run with Just

```bash
just up         # start postgres + redis + api in docker
just logs       # follow all service logs
just down       # stop services
```

## Run locally

```bash
just deps-up    # start postgres + redis only
just dev        # run go app locally
```

## Quality

```bash
just test
just fmt
```

## API Base

Base path: `/api/v1`

Examples:

- `POST /api/v1/chat`
- `GET /api/v1/chat/history`
- `POST /api/v1/race/strategy`
- `GET /api/v1/events/stream`
- `GET /api/v1/stats/overview`
- `GET /api/v1/readiness/today`
