# ADR 0003: Cookie vs Bearer

## Status

Accepted (B3)

## Context

Agents (Android, macOS tray) and the dashboard must authenticate as devices. Browsers should not keep a raw token in `localStorage` (XSS). Agents have no cookie jar that we want to depend on.

## Decision

- **Bearer** `Authorization: Bearer <token>` for Android and the laptop agent. Token is stored in EncryptedSharedPreferences / a 0600 config file (C1/C3).
- **Cookie** `deskdrop_device` (httpOnly, SameSite=Lax, Path=/) set when a **web** device claims a pairing code. The C4 dashboard proxies `/v1` so the browser is same-origin; D0 Caddy does that in production. Claim JSON `token` is stripped by the proxy.
- If both are present, Bearer wins.
- Lookup is HMAC of the token against `devices.token_hash` where `revoked_at IS NULL`. User id comes from that row.

## Alternatives

- Cookie-only: awkward for a tray app and a share-target APK.
- localStorage Bearer in the dashboard: simpler CORS, worse XSS.
- JWT: extra secret and expiry story; the device token already is the credential.

## Consequences

- Local Compose uses `Secure=false` (HTTP). D0 must serve HTTPS and can set Secure.
- Revoke sets `revoked_at`; the next request with that token is 401. No JWT clock skew.
