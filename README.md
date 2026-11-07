# Edgenet

A self-hosted portal for monitoring Arduino and ESP32 devices and coordinating firmware updates. Built with **Vue 3 + TypeScript + Vite**, **Bun**, and **Go Fiber v3**.

## What works

- Enroll ESP32 and Arduino devices with a unique access token.
- Monitor online/offline status, CPU, memory, temperature, uptime, and firmware version.
- View the latest 60 telemetry samples per device and the latest 100 activity events.
- Upload artifacts up to 64 MiB, with server-generated SHA-256 checksums.
- Deploy to one or more devices with an **exact matching hardware target**.
- Track queued → downloading → installing → succeeded / failed jobs; cancel queued jobs.
- Rotate device tokens and remove devices; credentials are separate from admin access.
- Persist metadata in a transactional bbolt database and store artifacts on disk.

The dashboard starts empty and displays real API data. Devices connect outward over HTTP(S), so they can live behind NAT; the server does not discover or directly dial hardware.

## Quick start

Requires **Go 1.26+** and **Bun 1.3+**. Node/npm are not required.

```sh
bun install
export ADMIN_TOKEN='replace-with-a-long-random-secret'
bun run api
```

The backend entry point is `main.go` in the project root. You can also start it directly with `go run main.go`.

In a second terminal:

```sh
bun run dev
```

Open **http://localhost:5173** and enter the same `ADMIN_TOKEN`. Vite proxies `/api` to `127.0.0.1:8080`. The browser stores the admin token in session storage for the current tab.

The API reads environment variables directly; `.env.example` is a reference, not an automatically loaded config file.

| Variable | Default | Purpose |
| --- | --- | --- |
| `ADMIN_TOKEN` | required | Admin API secret, at least 16 characters |
| `LISTEN_ADDR` | `:8080` | API listen address |
| `DATA_DIR` | `data` | Database and firmware artifact directory |

### Single-server build

```sh
bun run build
export ADMIN_TOKEN='replace-with-a-long-random-secret'
bun run api
```

Open **http://localhost:8080**. Fiber serves the production frontend from `dist/`. Run from the project root so the server can locate it. Place the server behind an HTTPS reverse proxy for remote devices.

### Checks

```sh
bun run build
bun run test:api
go test -race ./...
go vet ./...
```

Optional browser integration test (uses its own database under `data/e2e`):

```sh
bunx --bun playwright install chromium
bun run test:e2e
```

## Connecting an Arduino or ESP32

1. Click **Add device**. Select **ESP32** or **Arduino** and an exact board target, such as `esp32-devkit-v1` or `arduino-uno-r4-wifi`.
2. Copy the device ID and token. Only the token hash is persisted by the API; the plaintext token is shown once.
3. Configure your board firmware’s HTTP client with the server URL and the displayed authentication headers.
4. Implement the device protocol below: send a heartbeat every 30 seconds, poll for jobs, and report update progress. A device is offline after 90 seconds without a heartbeat.

**Trying a dashboard update:** upload firmware built for the same hardware target, open **Deployments → New deployment**, select the artifact and device, then watch the status. Your board firmware must implement downloading, verification, and board-specific flashing. Report success only after validating the updated firmware.

## ESP32 and Arduino integration

The portal supplies the control plane and transport; your firmware supplies hardware-specific telemetry and flashing:

| Hardware | Integration |
| --- | --- |
| ESP32 | HTTP client for heartbeats/job polling; an OTA-capable partition layout and ESP32 `Update`/ESP-IDF OTA APIs for flashing. Verify the artifact before activating it. Persist the job ID across reboot and report success after boot validation. |
| Network-capable Arduino | Implement the HTTP protocol and the OTA mechanism supported by that board/bootloader. Exact targets should distinguish board variants and flash layouts. |
| USB-only Arduino (e.g. classic Uno) | A networked gateway must own the device identity, collect serial telemetry, and run a board-specific flashing tool. A classic Uno cannot receive network OTA directly. |

There is no universal binary or flashing routine shared by these boards. Firmware built for one board or ESP32 partition layout is not interchangeable with another target. Arduino/ESP32 sketches and serial gateway implementations are not included yet; the protocol below is the integration contract.

## Device protocol

Every agent endpoint requires:

```http
Authorization: Bearer DEVICE_TOKEN
X-Device-ID: DEVICE_ID
```

### Heartbeat

```sh
curl -X POST "$EDGENET_URL/api/agent/heartbeat" \
  -H "Authorization: Bearer $EDGENET_DEVICE_TOKEN" \
  -H "X-Device-ID: $EDGENET_DEVICE_ID" \
  -H 'Content-Type: application/json' \
  -d '{"version":"1.0.0","metrics":{"cpu":12.5,"memory":38,"temperature":42,"uptime":3600}}'
```

`version` is required. `cpu` and `memory` are optional percentages (0–100); `temperature` is optional in °C; `uptime` is seconds. Omit unsupported metrics rather than inventing values.

### Poll jobs

`GET /api/agent/jobs` returns an array (empty when no work is pending):

```json
[
  {
    "deployment": {
      "id": "job-id",
      "deviceId": "device-id",
      "firmwareId": "artifact-id",
      "version": "1.1.0",
      "status": "queued"
    },
    "firmware": {
      "id": "artifact-id",
      "name": "firmware.bin",
      "version": "1.1.0",
      "target": "esp32-devkit-v1",
      "sha256": "64-character-hex-digest",
      "size": 1048576
    },
    "downloadPath": "/api/agent/jobs/job-id/artifact"
  }
]
```

Responses also include timestamps and display metadata. Polling does not claim a job. Report `downloading` **before** downloading; this atomically claims the queued job and prevents cancellation. Only one active deployment is allowed per device. Offline devices may have queued jobs.

Download from `downloadPath` with the same credentials. Downloads are authorized only for the assigned device while its job is active. Verify the **target, size, and SHA-256** on the device before installing.

### Report status

```http
POST /api/agent/jobs/JOB_ID/status
Content-Type: application/json

{"status":"downloading","message":"Optional details, up to 500 characters"}
```

Valid transitions:

```text
queued ──→ downloading ──→ installing ──→ succeeded
  │             │              │
  └─────────────┴──────────────┴──→ failed
  └──→ cancelled (admin only, while still queued)
```

Repeating the current status is idempotent. Skipping a state or changing a terminal state returns `409`. Report `succeeded` only after installation/boot verification. The server does not infer success from an upload or a download. Jobs have no automatic timeout: interrupted devices retain their work until their agent reconnects and resolves the outcome.

## Admin API

Use `Authorization: Bearer ADMIN_TOKEN`.

| Method | Endpoint | Body |
| --- | --- | --- |
| GET | `/api/admin/state` | Fleet, artifacts, jobs, and activity snapshot |
| POST | `/api/admin/devices` | `{ "name", "kind", "target", "location" }` |
| POST | `/api/admin/devices/:id/rotate-token` | Returns new plaintext token once |
| DELETE | `/api/admin/devices/:id` | Blocked while an active job exists |
| POST | `/api/admin/firmware` | Multipart `file`, `version`, `target` |
| POST | `/api/admin/deployments` | `{ "firmwareId": "...", "deviceIds": ["..."] }` |
| POST | `/api/admin/deployments/:id/cancel` | Cancels only a queued job |

Device kinds: `esp32`, `arduino`. Enrollment fields and versions have bounded lengths; artifact target/version pairs are unique. Multi-device deployment validation and creation are atomic. Errors use `{ "error": "message" }`.

## Structure and scope

```text
main.go               Go entry point and graceful shutdown
internal/portal/      Fiber routes, models, bbolt transactions, integration tests
src/                  Vue dashboard, typed API data, responsive styles
data/                 Runtime database + artifacts (gitignored)
```

This is a **single-workspace MVP**, designed for a small fleet on one server. bbolt serializes writes; each transaction reads/writes the metadata snapshot. Heartbeat history is bounded, but deployment records and artifacts are retained. Stop the server before copying the entire data directory for a consistent backup. Run only one API process per database.

Next steps for a larger deployment are device-specific OTA adapters, signed firmware and secure-boot integration, a relational/time-series database, artifact retention, rollout batches and rollback policies, and user accounts/RBAC. SHA-256 detects corruption; it is not a firmware signature. The current UI polls the API every five seconds rather than using MQTT or WebSockets.
