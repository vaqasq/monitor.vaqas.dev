# Container Health Monitor

A long-running Go daemon that monitors Docker containers by interfacing directly with the Docker Engine API over a Unix socket. Metrics are persisted to SQLite and served on a live dashboard at [monitor.vaqas.dev](https://monitor.vaqas.dev).

Built as an alternative to using the Docker SDK — all communication with the Docker Engine is handled via raw HTTP over `/var/run/docker.sock`, giving full visibility into what the SDK abstracts away.

## What it does

- Polls all running containers every 30 seconds via the Docker Engine API
- Calculates CPU% and memory% from raw kernel counters returned by the API
- Persists metrics to a SQLite database using Go's `database/sql`
- Serves a server-side rendered dashboard using `html/template`
- Runs as a systemd service on a self-managed Ubuntu VPS, routed through Caddy

## Architecture

The daemon runs as a systemd service directly on the host, polling the Docker Engine via `/var/run/docker.sock` every 30 seconds. Metrics are written to a SQLite database after each poll. On each HTTP request, the dashboard queries the most recent row per container from SQLite and renders the result server-side using `html/template`. Caddy handles TLS and proxies public traffic to the Go HTTP server.

## Technical decisions

**Unix socket over TCP** — the daemon runs on the host alongside the Docker Engine, so network exposure is unnecessary and a security liability. All API communication stays local via the Unix socket.

**Raw HTTP over the Docker SDK** — using `http.Transport` with a custom `DialContext` rather than the official SDK makes the underlying API calls explicit. Responses are unmarshaled manually into typed structs.

**modernc.org/sqlite over PostgreSQL** — SQLite is sufficient for two containers polled at 30-second intervals. No separate database process, no operational overhead, single file on disk.

**Server-side rendering over a JS frontend** — `html/template` renders the dashboard on the server. No JavaScript framework, no build step. The page auto-refreshes every 30 seconds via an HTML meta tag.

**systemd over Docker** — the daemon needs access to `/var/run/docker.sock`. Running it directly on the host as a systemd service is simpler and more appropriate than mounting the socket into a container.

## Stack

- Go (`net/http`, `database/sql`, `html/template`)
- SQLite via `modernc.org/sqlite`
- Docker Engine API (v1.43)
- systemd
- Caddy

## Running locally

The daemon requires access to `/var/run/docker.sock` and must run on a host with Docker installed.

```bash
git clone https://github.com/vaqasq/monitor.vaqas.dev
cd monitor.vaqas.dev
go run main.go
```

Dashboard available at `http://localhost:8080`.