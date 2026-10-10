# v1 retrospective

The January 2026 deskdrop (Kotlin + Node/Express/Socket.IO/MongoDB/Redis + Electron) never worked end to end. v2 is a rebuild on this branch, not a port. Each cause below has a v2 control; later steps implement those controls — this file is the map.

## Phone and laptop were different users

**What happened.** Android authenticated with a compiled-in development bearer token that always mapped to a single hard-coded user. The laptop paired through WebAuthn (email / passkey) and received a pairing token under that other user. Shares succeeded on the phone and never arrived on the paired desktop.

**v2 control.** There is no email, password, WebAuthn, or development bearer. A user is the set of devices that claimed pairing codes in the same chain. Bootstrap `POST /v1/pairings` creates the user; later codes require an already-paired device. Phone, laptop, and dashboard all store a device token. Identity always comes from that token (or the dashboard cookie), never from a JSON `userId`.

## Env names did not match the code

**What happened.** `.env.example` documented `RP_ID` / `RP_NAME` / `RP_ORIGIN`. The WebAuthn code read `WEBAUTHN_RP_ID` / `WEBAUTHN_RP_NAME` / `WEBAUTHN_ORIGIN`. JWT secrets were documented in one file and omitted in another. Local and hosted setups drifted until pairing broke.

**v2 control.** No WebAuthn. `.env.example` lists the same names the process reads (`DEVICE_TOKEN_PEPPER`, `DATABASE_URL`, `HTTP_ADDR`, `DASHBOARD_ORIGIN`) with empty values. From B1, missing required env is a hard process exit. No fallback strings in code.

## Delivery used sockets on a host that slept

**What happened.** Production sat on a free host that idled out. The desktop used Socket.IO. After about a minute of disconnect, the agent stopped reconnecting. A laptop that was asleep or off at share time often never saw the link.

**v2 control.** No long-lived sockets in v2. The API stores pending rows in Postgres. The laptop agent pulls on start, on wake, and on a short interval, opens the browser, then acks. Links survive API restarts and laptop sleep. (SSE is Post-v2, only if measured lag says so.)

## Desktop did not start at login

**What happened.** The Electron tray app had to be launched by hand. After a reboot, nothing pulled the queue.

**v2 control.** The v2 agent is a Go macOS tray binary with a LaunchAgent (start at login), verified in C3.

## Queue rows disappeared without an ack

**What happened.** The server treated “emitted to a socket” or “written to Redis” as delivered and dropped or drained the queue without the laptop confirming it had opened the URL.

**v2 control.** A link stays pending until `POST /v1/links/{id}/ack`. Ack is idempotent. Pull does not delete. Delivered rows are purged only after 30 days.

## Insecure defaults

**What happened.**

- A default development bearer in API middleware and in the Android client.
- JWT signing secrets hard-coded as development strings.
- User id accepted from the request body.
- Wildcard CORS on HTTP and Socket.IO (any origin).
- Electron renderer had Node integration on and context isolation off.

**v2 control.** No default tokens or signing secrets. Tokens stored as `HMAC-SHA256(pepper, token)`, pepper from env. User id never from JSON. CORS allowlist is `DASHBOARD_ORIGIN` only. No Electron. Later CI greps the tree for those v1 footguns so they cannot return.

## Hosting and identity were implied, not declared

**What happened.** The Android client pointed at a Render URL. There was no Docker, Terraform, Kubernetes, or GitHub Actions in the repo. Secrets and hostnames lived in source.

**v2 control.** Local Compose first (A2), then a public Compose+Caddy deploy (D0), Terraform of the Oracle VM recorded in A0 (D3), optional migrate to k3s if Ampere (D4). A0 is still **pending** (Oracle signup). Images and deploy pipelines are later steps; they will not bake tokens into the app.

## What v2 will not repeat on purpose

v1 also mixed Redis queues, Mongo documents, and Socket.IO presence. v2 uses Postgres as the queue and a single Go API. That is a product choice, not only a bugfix: fewer moving parts on one Always Free VM, and a delivery story you can trace with SQL.
