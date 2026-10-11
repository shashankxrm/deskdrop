# Privacy

deskdrop has no email or password. A user is the set of devices that claimed pairing codes on the same chain.

- **Tokens.** Device tokens are HMAC-SHA256 at rest. The Android app uses EncryptedSharedPreferences. The macOS agent uses a 0600 file. The dashboard uses an httpOnly `deskdrop_device` cookie and does not put the raw token in `localStorage`. The dashboard proxy strips `token` from claim JSON.
- **Links.** Shared `http`/`https` URLs (optional title) sit in Postgres until ack, then are purged after 30 days if delivered.
- **Logs.** JSON access logs include `request_id`, method, path, status. No advertising or analytics SDKs in v2.
- **Delete.** `DELETE /v1/account` removes the user and cascaded devices, pairing codes, links, and deliveries.
- **Unpair.** `DELETE /v1/devices/{id}` sets `revoked_at`. The next request with that token is 401 (laptop tray poll included).

Dashboard page: `/privacy`. Linked from the app and this file.
