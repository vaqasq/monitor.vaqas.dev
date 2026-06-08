# Container Health Monitor

A long-running Go daemon that monitors Docker containers through the Docker Engine API over a Unix socket. Metrics are stored in SQLite and displayed on a live dashboard at [monitor.vaqas.dev](https://monitor.vaqas.dev).

I built this instead of using the Docker SDK so I could work directly with the underlying HTTP API. This meant writing the transport layer myself, manually parsing JSON responses, and understanding exactly what the SDK normally hides from you.

![Go](https://img.shields.io/badge/Go-00ADD8?style=flat&logo=go&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-003B57?style=flat&logo=sqlite&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-2496ED?style=flat&logo=docker&logoColor=white)
![Linux](https://img.shields.io/badge/Linux-FCC624?style=flat&logo=linux&logoColor=black)
![systemd](https://img.shields.io/badge/systemd-000000?style=flat&logoColor=white)

## What it does

- Polls all running containers every 30 seconds via the Docker Engine API
- Calculates CPU% and memory% from raw kernel counters returned by the API
- Persists metrics to a SQLite database using Go's `database/sql`
- Serves a server-side rendered dashboard using `html/template`
- Runs as a systemd service on a self-managed Ubuntu VPS, routed through Caddy

## Architecture

The daemon runs as a systemd service directly on the host. Every 30 seconds it polls the Docker Engine through `/var/run/docker.sock` and writes the results to SQLite. When someone visits the dashboard, the Go HTTP server queries the most recent row per container and renders the page server-side using `html/template`. Caddy handles TLS and routes public traffic to the server.

## Technical decisions

**Unix socket over TCP** - the daemon runs on the same host as Docker, so there is no need to expose a network port. Early in the project I made the mistake of exposing Docker's TCP port publicly, which is a serious security risk. The Unix socket keeps everything local.

**Raw HTTP over the Docker SDK** - I used `http.Transport` with a custom `DialContext` to talk to the Docker Engine directly. This was harder than using the SDK but taught me how the API actually works under the hood.

**modernc.org/sqlite over PostgreSQL** - SQLite is more than enough for two containers at 30-second intervals. No separate process to manage, no configuration, just a single file on disk.

**Server-side rendering over a JS frontend** - Go's `html/template` renders the dashboard on the server. No JavaScript framework, no build step. The page refreshes every 30 seconds using an HTML meta tag.

**systemd over Docker** - the daemon needs direct access to `/var/run/docker.sock` on the host. Running it as a systemd service is simpler and more appropriate than mounting the socket into a container.

## What I learned

- How to make HTTP requests over a Unix socket in Go using a custom transport
- How the Docker Engine API works at the HTTP level, including how CPU usage is calculated from raw kernel deltas
- How to use Go's `database/sql` with SQLite for persistent storage
- How to deploy a Go binary as a systemd service and set up a reverse proxy with Caddy
- Why exposing Docker's TCP port without TLS is dangerous

## Stack

- Go (`net/http`, `database/sql`, `html/template`)
- SQLite via `modernc.org/sqlite`
- Docker Engine API (v1.43)
- systemd
- Caddy

## Running locally

Requires access to `/var/run/docker.sock` and a host with Docker installed.

```bash
git clone https://github.com/vaqasq/monitor.vaqas.dev
cd monitor.vaqas.dev
go run main.go
```

Dashboard available at `http://localhost:8081`.