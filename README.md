# Pacer Monorepo

Pacer is an AI-assisted marathon coaching platform with:

- `pacer-api`: Go + Fiber backend (auth, activities, planning, chat SSE, race strategy, stats, notifications SSE)
- `pacer-web`: Next.js frontend (dashboard, plan, chat, race, stats)

## Repository Layout

- `pacer-api/` backend service
- `pacer-web/` frontend app
- `pacer-api/db/migrations/` SQL schema migrations

## Prerequisites

- Go `1.25+`
- Node `20.9+` (required by Next 16)
- npm `10+`
- Docker + Docker Compose (recommended for backend dependencies)
- `just` (optional, but recommended)

## Quick Start

### 1) Start backend infra + API

```bash
cd pacer-api
just up
```

### 2) Start frontend

```bash
cd pacer-web
just dev
```

Frontend: `http://localhost:3000`  
API health: `http://localhost:3001/health`

## Environment Files

- Backend example: `pacer-api/.env.example`
- Frontend example: `pacer-web/.env.local.example`

Keep environment files service-scoped:

- Backend: `pacer-api/.env`
- Frontend: `pacer-web/.env.local`

Do not use a repo-root `.env`.

Copy examples locally and adjust values before running outside Docker.

## Developer Commands

Commands are intentionally split per service:

- Backend commands: `pacer-api/Justfile`
- Frontend commands: `pacer-web/Justfile`

Run `just --list` inside each folder to see available tasks.

## Git Hygiene

- Session progress/status markdown files were removed.
- Legacy root-level orchestration docs and duplicate root docker compose were removed.
- Service-specific docs and task runners now live in each service directory.
