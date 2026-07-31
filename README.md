# Pulse99

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-00ADD8?style=flat&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-green?style=flat)](LICENSE)

A lightweight, zero-dependency-required uptime monitoring daemon written in Go. Pulse99 periodically pings your HTTP/HTTPS endpoints, tracks failure thresholds to suppress transient noise, and dispatches alerts across Discord, Telegram, Email, and Twilio SMS. Ships with a real-time dark-mode web dashboard and SQLite persistence.

---

## Features

- **Concurrent health checks** -- all targets probed simultaneously via goroutines.
- **Failure thresholds** -- only fires alerts after N consecutive failures, suppressing transient flapping.
- **Multi-channel alerting** -- Discord (rich embeds), Telegram, Email (SMTP/STARTTLS), Twilio SMS.
- **Alert cooldowns** -- per-target, per-channel rate limiting prevents notification storms.
- **Retry with exponential backoff** -- failed webhook calls are retried automatically.
- **Per-target configuration** -- custom HTTP headers, accepted status code lists, timeouts.
- **SQLite storage** -- check history, latency metrics, and incident logs persist across restarts.
- **Web dashboard** -- real-time dark-mode UI with live status cards, latency trend charts (Chart.js), incident log, all updated via WebSocket.
- **REST API** -- programmatic access to status, history, and statistics.
- **Graceful shutdown** -- SIGINT / SIGTERM handled cleanly.

---

## Quickstart

### Prerequisites

- Go 1.21 or later

### Clone, Build, Run

```bash
git clone https://github.com/yaojiayi2020/Pulse99.git
cd Pulse99
go mod tidy
go run main.go
```

The daemon starts immediately with the included `config.yaml` (3 test endpoints). The dashboard is at:

```
http://localhost:8080
```

### Build a Binary

```bash
go build -o pulse99 main.go
./pulse99 -config config.yaml
```

To use a custom config path:

```bash
./pulse99 -config /etc/pulse99/config.yaml
```

---

## Configuration

Pulse99 reads a single YAML file. Every section is optional and ships with sensible defaults.

### Full Reference (`config.yaml`)

```yaml
# -- Scan Settings ------------------------------------------
interval_seconds: 15        # seconds between full scan sweeps
failure_threshold: 3        # consecutive failures before triggering a CRITICAL alert

# -- Webhooks (simple mode) ---------------------------------
webhooks:
  discord: ""               # Discord webhook URL
  telegram:
    bot_token: ""           # token from @BotFather
    chat_id: ""             # target chat or channel ID

# -- Extended Alerts ----------------------------------------
alerts:
  email:
    enabled: false
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    smtp_username: ""
    smtp_password: ""
    from_address: "pulse99@example.com"
    to_addresses:
      - "admin@example.com"
  twilio:
    enabled: false
    account_sid: ""
    auth_token: ""
    from_phone: "+1234567890"
    to_phones:
      - "+0987654321"

# -- Dashboard ----------------------------------------------
dashboard:
  enabled: true             # serve the web UI and REST API
  port: 8080                # HTTP listen port

# -- Storage ------------------------------------------------
storage:
  db_path: "pulse99.db"     # SQLite database file path

# -- Targets ------------------------------------------------
targets:
  - name: "production-api"
    url: "https://api.example.com/health"
    method: "GET"
    expected_status: 200
    timeout_seconds: 10
    headers: {}

  - name: "staging-dashboard"
    url: "https://staging.example.com"
    method: "GET"
    allowed_statuses: [200, 201, 204]
    timeout_seconds: 15
    headers:
      Authorization: "Bearer eyJ..."
      X-Environment: "staging"
```

### Target Fields

| Field | Type | Default | Description |
|---|---|---|---|
| `name` | `string` | required | Display name for the target |
| `url` | `string` | required | Full URL to probe |
| `method` | `string` | `GET` | HTTP method |
| `expected_status` | `int` | `200` | Exact status code for healthy (ignored if `allowed_statuses` is set) |
| `allowed_statuses` | `[]int` | `[]` | Accept any status in this list as healthy; overrides `expected_status` |
| `headers` | `map[string]string` | `{}` | Custom HTTP headers sent with each request |
| `timeout_seconds` | `int` | `10` | Per-request timeout |

### Notification Channels

| Channel | Config Key | Transport | Notes |
|---|---|---|---|
| Discord | `webhooks.discord` | HTTP POST (webhook) | Rich embeds with color-coded fields |
| Telegram | `webhooks.telegram` | HTTP POST (Bot API) | Plain text, no Markdown parsing |
| Email | `alerts.email` | SMTP + STARTTLS | Works with Gmail, SendGrid, any SMTP server |
| Twilio SMS | `alerts.twilio` | REST API | Sends to all `to_phones` |

Empty or omitted configuration values are silently skipped at runtime. No channel is required to be configured for Pulse99 to function.

---

## Dashboard

The built-in web dashboard provides real-time monitoring without any build step or external dependencies.

### Metrics

| Card | Description |
|---|---|
| **Uptime** | Overall percentage of checks with UP status |
| **Avg Latency** | Mean response time across all currently UP targets |
| **Targets** | Total count + number currently healthy |
| **Alerts** | Current count of DOWN targets |

### Charts

A **response time trend** line chart tracks latency per target over the last 20 sweeps. The chart updates in real time via WebSocket.

### Incident Log

A scrollable table shows the last 50 check results with color-coded event tags:
- **Recovered** -- green, target came back online
- **Down** -- red, threshold breached

Data is fetched from the REST API every 15 seconds and supplemented by WebSocket pushes.

---

## REST API

All endpoints return JSON. The API is served on the same port as the dashboard.

### `GET /api/status`

Current state of every target.

```json
[
  {
    "name": "production-api",
    "url": "https://api.example.com/health",
    "status": "UP",
    "status_code": 200,
    "latency_ms": 42,
    "failure_count": 0,
    "threshold": 3,
    "is_down": false
  }
]
```

### `GET /api/history?target=&limit=50`

Recent check records. Filter by target name; default limit is 100.

```json
[
  {
    "id": 1423,
    "sweep_id": 47,
    "timestamp": "2026-07-31T12:00:00Z",
    "target": "production-api",
    "status": "UP",
    "status_code": 200,
    "latency_ms": 42,
    "error": ""
  }
]
```

### `GET /api/stats`

Aggregate uptime statistics per target.

```json
[
  {
    "target": "production-api",
    "total_checks": 940,
    "up_checks": 935,
    "down_checks": 5,
    "uptime_pct": 99.47,
    "avg_latency_ms": 48.2
  }
]
```

### `GET /ws`

WebSocket endpoint. On each completed scan sweep the server pushes the full `GET /api/status` payload to all connected clients.

### `GET /`

Serves the web dashboard HTML.

---

## Command-Line Interface

```
pulse99 uptime daemon v2.0.0 [starting]

config: 3 targets | scan interval 15s | failure threshold 3
dashboard: http://0.0.0.0:8080
[10:05:36] check sweep #1 (3 targets)
[10:05:36] OK    production-api      42ms 200
[10:05:36] OK    staging-dashboard   67ms 200
[10:05:36] WARN  legacy-endpoint     1/3 status 500 (expected 200)
[10:05:36] sweep done | 2 up  0 down  1 degraded
```

### Log Levels

| Prefix | Meaning |
|---|---|
| `OK` | Target responded with expected status |
| `WARN` | Failure count below threshold; monitoring |
| `DOWN` | Threshold reached; alert dispatched |
| `UP` | Previously down target has recovered |

---

## Architecture

```
pulse99/
|-- main.go                          daemon entry point, wiring, signal handling
|-- config.yaml                      runtime configuration
|-- go.mod / go.sum
|-- internal/
    |-- config/config.go             YAML parsing, validation, defaults
    |-- checker/checker.go           concurrent sweep engine, state tracking, interfaces
    |-- alerter/alerter.go           alert orchestrator: cooldowns, retries, dispatch routing
    |-- notify/
    |   |-- notifier.go              Notifier interface + Payload type
    |   |-- discord.go               Discord rich embed webhook
    |   |-- telegram.go              Telegram Bot API
    |   |-- email.go                 SMTP with STARTTLS
    |   |-- twilio.go                Twilio SMS REST API
    |-- store/store.go               SQLite persistence (via modernc.org/sqlite)
    |-- api/
    |   |-- server.go                HTTP server, route registration, REST handlers
    |   |-- hub.go                   WebSocket hub: client registry, broadcast fan-out
    |   |-- index.html               embedded dashboard UI (Tailwind + Chart.js CDN)
    |-- logger/logger.go             structured ANSI terminal log output
```

### Data Flow

1. `main.go` loads config, initializes notifiers, alerter, SQLite store, checker engine, and HTTP server.
2. A `time.Ticker` triggers a scan sweep at `interval_seconds`.
3. `SweepEngine.RunSweep()` spawns one goroutine per target.
4. Each goroutine executes an HTTP request, measures latency, updates thread-safe state (`sync.Mutex`), and writes a record to SQLite.
5. If failures reach `failure_threshold`, the `Alerter` dispatches a CRITICAL alert to all configured notifiers (with cooldown and retry).
6. On recovery, a RECOVERY alert is dispatched.
7. After each sweep, the WebSocket hub broadcasts the full status snapshot to all connected dashboard clients.

---

## Dependencies

Go standard library is preferred wherever possible.

| Package | Purpose |
|---|---|
| `gopkg.in/yaml.v3` | YAML configuration parsing |
| `modernc.org/sqlite` | Embedded SQLite (pure Go, no CGO required, cross-compile safe) |
| `github.com/gorilla/websocket` | WebSocket server |
| Chart.js + Tailwind CSS | Dashboard UI (loaded via CDN in the browser; no build step) |

---

## License

MIT
