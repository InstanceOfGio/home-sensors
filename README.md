# home-sensors

A small Go server that receives BLE sensor events from a Minew gateway,
stores them in SQLite, and displays them on a real-time dashboard with
temperature/humidity charts (indoor vs. outdoor).

Current scope: webhook receiver, SQLite history, dashboard with live
per-sensor state, historical charts (24h/7d/30d), real-time updates over
WebSocket. Door/window sensors, a comfort index, and notifications are
not implemented yet.

## Project layout

```
cmd/server/         entrypoint
internal/config/    configuration from environment variables
internal/models/    shared types (Device, CurrentState, Minew payload)
internal/storage/   SQLite setup, migrations, queries
internal/webhook/   Minew webhook handler
internal/websocket/ real-time broadcast hub (package "hub")
internal/api/       chi router + REST handlers
migrations/         SQL schema, embedded into the binary
frontend/           static dashboard (HTML/CSS/JS + Chart.js via CDN), embedded into the binary
```

The binary is single-file: the frontend and migrations are embedded via
`go:embed`, so no extra files need to be shipped alongside it at runtime.

## Requirements

- Go 1.25+ (not needed if you only deploy via Docker/Fly)
- No C compiler required: uses `modernc.org/sqlite`, a pure-Go SQLite driver

## Running locally

```bash
go mod tidy      # resolves dependencies and generates go.sum
go run ./cmd/server
```

The server starts on `:8080`, with the database at `./data/sensors.db`
(created on first run). Configurable via the `PORT` and `DB_PATH`
environment variables (see `.env.example`).

Open `http://localhost:8080`.

## Authentication

The whole app (dashboard, REST API, webhook, WebSocket) is protected with
HTTP Basic Auth when `BASIC_AUTH_USER` and `BASIC_AUTH_PASS` are both set.
If either is left unset, the server starts without authentication and logs
a warning — handy for local development, but make sure both are set in
production.

Point the Minew gateway's webhook URL at
`https://user:pass@your-app.fly.dev/webhook/minew` (or configure an
`Authorization: Basic <base64(user:pass)>` header directly, if the gateway
supports custom headers). The browser dashboard just prompts for
credentials natively on first load.

## Endpoints

- `POST /webhook/minew` — receives the gateway payload (see below)
- `GET /api/devices` — list known sensors
- `PATCH /api/devices/{id}` — assign a `name`/`room`/`type` to a sensor
  (also available from the dashboard, by clicking a sensor's name on its card)
- `GET /api/current` — latest state of every sensor
- `GET /api/history?range=24h|7d|30d` — historical readings
- `GET /ws` — WebSocket stream of real-time updates

### Expected payload from the Minew gateway

```json
{
  "tm": "2026-07-19T15:24:23.975Z",
  "gw": "ac233fc27092",
  "seq": 1,
  "adv": [
    {
      "type": "ht",
      "temperature": 27.15,
      "humidity": 71.1,
      "battery": 100,
      "rssi": -44,
      "tm": "2026-07-19T15:24:28.928Z",
      "mac": "c3000071fb47"
    }
  ]
}
```

Only entries with `type: "ht"` are stored; other types (door/window,
beacon, button) are ignored for now and don't cause the request to fail.

The gateway often sends several closely-spaced readings for the same mac
(BLE radio noise): **throttling is expected to happen upstream, on the
gateway** (roughly one data point every ~10 minutes per sensor) — the
backend stores whatever it receives as-is, without deduplicating.

### First run: labeling sensors

When an event arrives for an unknown mac address for the first time, the
backend automatically creates a device with `name` set to the mac address
and an empty `room`. Open the dashboard, click a sensor card's name, and
give it a real name/room (e.g. "Outdoor" / "outdoor", "Living room" /
"living-room"). No restart needed.

## Deploying to Fly.io (free tier)

SQLite works well on Fly because persistent volumes survive VM restarts
(unlike some other free tiers with an ephemeral filesystem). Double-check
Fly's current free-tier limits, since they change over time.

```bash
fly launch --no-deploy        # generates/updates the app, uses the existing fly.toml
fly volumes create sensors_data --size 1 --region ams
fly secrets set BASIC_AUTH_USER=changeme BASIC_AUTH_PASS=changeme
fly deploy
```

`BASIC_AUTH_USER`/`BASIC_AUTH_PASS` are set via `fly secrets set` rather
than `[env]` in `fly.toml`, since that file is committed to the repo.

Then point the Minew gateway at `https://<your-app>.fly.dev/webhook/minew`.

## Technical notes

- Timestamps are stored as INTEGER unix milliseconds, not TEXT/DATETIME:
  this avoids conversion ambiguity between the SQLite driver and
  `time.Time`.
- A single open DB connection (`SetMaxOpenConns(1)`) plus
  `journal_mode=WAL`: sufficient for the write volume of a couple of home
  sensors, and avoids "database is locked" errors.
- Charts use a fixed categorical palette (color tied to the sensor, never
  to its rank) and a single Y axis per chart — no dual axes.

## Roadmap

- `services/`, `scheduler/`: comfort analysis, "should I open the
  windows", heat loss estimation — to be added when needed, not present yet
- BLE door/window sensors
- Notifications (Telegram/email)
