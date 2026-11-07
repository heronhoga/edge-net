# Edgenet

A self-hosted, multi-user portal for monitoring Arduino and ESP32 devices and coordinating firmware updates. People sign in with Google and manage their own private device workspace. Built with **Vue 3 + TypeScript + Vite**, **Bun**, and **Go Fiber v3**.

Supported device families are limited to **Arduino and ESP32**.

## What works

- Sign in with Google; each account has its own devices, firmware, deployments, and activity.
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
cp .env.example .env
# Configure APP_URL, GOOGLE_CLIENT_ID, and GOOGLE_CLIENT_SECRET in .env (see below).
bun run api
```

The backend entry point is `main.go` in the project root. You can also start it directly with `go run main.go`.

In a second terminal:

```sh
bun run dev
```

Open **http://localhost:5173** and choose **Continue with Google**. Vite proxies `/api`, including OAuth callbacks, to `127.0.0.1:8080`. Set `APP_URL=http://localhost:5173` and use that exact origin in your browser.

For local development without Google credentials, set an optional `ADMIN_TOKEN` of at least 16 characters and expand **Local administrator** on the sign-in page. This opens a separate local workspace.

The API uses `godotenv` to load `.env` from the current working directory at startup. Existing environment variables take precedence over values in the file. The file is optional when configuration is supplied through the environment; unreadable or malformed files stop startup. `.env` is gitignored.

| Variable | Default | Purpose |
| --- | --- | --- |
| `APP_URL` | `http://localhost:5173` | Public browser origin; HTTPS required outside localhost |
| `GOOGLE_CLIENT_ID` | unset | Google OAuth web application's client ID |
| `GOOGLE_CLIENT_SECRET` | unset | Google OAuth client secret, used only by the backend |
| `ADMIN_TOKEN` | unset | Optional local-workspace access token, at least 16 characters |
| `LISTEN_ADDR` | `:8080` | API listen address |
| `DATA_DIR` | `data` | Database and firmware artifact directory |

Configure both Google credentials, or set `ADMIN_TOKEN` for local-only access. Leave `ADMIN_TOKEN` blank for a Google-only deployment. Never put the Google client secret in a `VITE_` variable or frontend code.

### Google sign-in setup

1. Create or select a project in the [Google Cloud Console](https://console.cloud.google.com/).
2. Configure the Google Auth Platform consent screen/branding and audience. Choose an external audience if people outside your organization should use the app. While the app is in testing, add the Google accounts that should be allowed to sign in as test users.
3. Create an OAuth client with application type **Web application**.
4. Add an **Authorized redirect URI** that exactly matches `APP_URL` plus `/api/auth/google/callback`:
   - Vite development: `http://localhost:5173/api/auth/google/callback`
   - Local single-server build: `http://localhost:8080/api/auth/google/callback`
   - Production example: `https://edgenet.example.com/api/auth/google/callback`
5. Copy the client ID and secret into `.env` and restart the backend:

```dotenv
APP_URL=http://localhost:5173
GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-client-secret
ADMIN_TOKEN=
```

The flow requests only `openid`, `email`, and `profile`. The backend exchanges the authorization code, verifies Google's signed ID token (issuer, audience, expiry, verified email, and nonce), and creates a session. OAuth state is bound to the initiating browser and can be used once; PKCE protects the code exchange. Google access/refresh tokens are not retained.

Google users receive an HttpOnly, SameSite=Lax session cookie lasting seven days; it is Secure on HTTPS. Session IDs are hashed in the database and logout revokes the current session. Browser mutations require a session-specific CSRF token. Identity is keyed by Google's stable subject identifier, so changing an email address does not create a new workspace. There is no shared fleet or organization/team access in this version.

### Existing installations

On startup, the old single-workspace database is migrated atomically to the **local-admin workspace**. Its devices, tokens, firmware, deployment history, and activity are preserved. Keep your existing `ADMIN_TOKEN` to access that workspace. Google accounts start with independent empty workspaces; existing data is not assigned to the first person who signs in. The local administrator token cannot access Google users' workspaces.

### Single-server build

```sh
bun run build
# Set APP_URL=http://localhost:8080 in .env and register the matching Google callback.
bun run api
```

Open **http://localhost:8080**. Fiber serves the production frontend from `dist/`. Run from the project root so the server can locate it. For a public deployment, set `APP_URL` to the HTTPS origin, register that exact Google callback, and put the server behind an HTTPS reverse proxy that forwards both the frontend and `/api` paths to the same backend. Redirects and cookie security come from `APP_URL`, not forwarded request headers.

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

## Authentication and workspace API

The browser uses `GET /api/auth/google` to start sign-in and returns through `GET /api/auth/google/callback`. `GET /api/auth/session` returns the current user, CSRF token, and enabled login methods (or a null user when signed out). `POST /api/auth/logout` revokes the current session.

The `/api/admin` route prefix is retained for compatibility; every operation is scoped to the authenticated user's workspace. Google users authenticate with their session cookie and send `X-CSRF-Token` on POST/DELETE requests. Local-workspace clients may use `Authorization: Bearer ADMIN_TOKEN`. A device token grants access only to that device's agent endpoints.

Devices, artifacts, deployments, and activity are isolated by workspace. Another user's resource IDs return `404`; a deployment must use firmware and devices from the same workspace. Firmware target/version uniqueness is also scoped per workspace.

| Method | Endpoint | Body |
| --- | --- | --- |
| GET | `/api/admin/state` | Fleet, artifacts, jobs, and activity snapshot |
| POST | `/api/admin/devices` | `{ "name", "kind", "target", "location" }` |
| POST | `/api/admin/devices/:id/rotate-token` | Returns new plaintext token once |
| DELETE | `/api/admin/devices/:id` | Blocked while an active job exists |
| POST | `/api/admin/firmware` | Multipart `file`, `version`, `target` |
| POST | `/api/admin/deployments` | `{ "firmwareId": "...", "deviceIds": ["..."] }` |
| POST | `/api/admin/deployments/:id/cancel` | Cancels only a queued job |

Accepted device kinds: `esp32` and `arduino` only. Enrollment fields and versions have bounded lengths; artifact target/version pairs are unique. Multi-device deployment validation and creation are atomic. Errors use `{ "error": "message" }`.

## Structure and scope

```text
main.go               Go entry point and graceful shutdown
internal/portal/      Fiber routes, models, bbolt transactions, integration tests
src/app/              Workspace orchestration, navigation, and snapshot types
src/domains/          Auth, devices, firmware, deployments, overview, connection guide
src/components/       Shared layout and UI components
src/shared/           HTTP client, formatting helpers, and notification composable
src/styles/           Global theme and responsive styles
data/                 Runtime database + artifacts (gitignored)
```

This is a **multi-user MVP** with one private workspace per Google account, hosted on one server. bbolt serializes writes; each transaction reads/writes one workspace's metadata snapshot. User records and hashed sessions are persisted alongside workspace data. Heartbeat history is bounded, but deployment records and artifacts are retained. Stop the server before copying the entire data directory for a consistent backup. Run only one API process per database.

Next steps for a larger deployment are device-specific OTA adapters, signed firmware and secure-boot integration, a relational/time-series database, per-user storage quotas, artifact retention, rollout batches and rollback policies, and team workspaces/RBAC. SHA-256 detects corruption; it is not a firmware signature. The current UI polls the API every five seconds rather than using MQTT or WebSockets.
