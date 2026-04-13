# Planning Poker

Real-time planning poker app for agile teams. Create a room, invite your team, and estimate story points together using Fibonacci cards.

## Stack

- **Backend** — Go 1.22, WebSocket (Gorilla)
- **Frontend** — Vue 3, Vite, Tailwind CSS
- **Infra** — Docker Compose, Nginx

## Getting started

```bash
cp .env.example .env
# Edit .env with your domain
docker compose up -d
```

The app is available on `http://localhost:3000` (or the port set in `FRONTEND_PORT`).

## Configuration

| Variable | Description | Default |
|---|---|---|
| `ALLOWED_ORIGINS` | Domain(s) allowed for CORS and WebSocket — must match the browser Origin header exactly | — |
| `BACKEND_PORT` | Internal port for the Go backend | `8080` |
| `FRONTEND_PORT` | Host port exposed by the frontend container | `3000` |

Multiple origins can be comma-separated: `https://poker.example.com,https://www.poker.example.com`

## Production

The app is designed to run behind a reverse proxy that handles TLS termination (e.g. Pangolin). Point your proxy to `localhost:FRONTEND_PORT`. Nginx inside the frontend container handles routing: static files are served directly, `/api/` and `/ws` are proxied to the backend.

Make sure `ALLOWED_ORIGINS` is set to your public HTTPS domain, otherwise WebSocket connections will be rejected.

## Features

- Create a room and share the URL with your team
- Vote with Fibonacci cards: 1, 2, 3, 5, 8, 13, 21, or Pass
- Votes are hidden until the manager reveals them
- Average score displayed on reveal
- Reconnection with exponential backoff
- Up to 50 participants per room
