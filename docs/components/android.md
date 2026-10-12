# Android (pairing + share)

The phone is a `phone` device. C1 claims a pairing code and stores the token in **EncryptedSharedPreferences**. C2 is an `ACTION_SEND` target: `text/plain` URL (optional title) → `POST /v1/links` with a new `Idempotency-Key` per tap. There is no compiled-in token or pepper.

## Why this way

- Production API URL is `https://shashank-dropdesk.duckdns.org` (pre-filled). Local Compose: `http://<lan-ip>:8080`. Emulator: `http://10.0.2.2:8080`. See [connect.md](../connect.md).
- Token at rest is AndroidX Security crypto (AES-256). The UI shows device id, never the token.
- Claim and share retries use **full jitter** backoff on 429/5xx/IO. 404/409/410/4xx do not retry.
- QR payload is `https://host/connect?code=ABCD-EFGH` (fills API URL + code). Older `deskdrop:https://host|CODE` still parses. Scan auto-claims. Pair stays enabled so you can switch chains without Forget.
- YouTube and similar apps put a URL (sometimes with a title) in `EXTRA_TEXT` / `EXTRA_SUBJECT`. Non-http(s) is rejected in-app.
- One UUID `Idempotency-Key` per share tap (kept across rotation). HTTP retries reuse that key so a retried intent does not insert a second `links` row.

## Failure modes

- Empty or non-http(s) API URL: in-app error, no request.
- Incomplete code: in-app error.
- 404/409/410: shown from JSON `error`; no retry.
- Camera missing: paste still works (`camera` is not required).
- Share while unpaired: opens pairing, nothing posted.
- Share of plain text / `ftp:` / `javascript:`: toast, nothing posted.
- Cleartext HTTP: allowed until D0 HTTPS. Do not ship a default token to “make LAN work.”

## Verify or demo

JDK 17+ (`JAVA_HOME`). Android SDK with `compileSdk` 34 (`sdk.dir` in `android/local.properties`, not committed).

```bash
# From repo root
echo "sdk.dir=$HOME/Library/Android/sdk" > android/local.properties

export JAVA_HOME="$(/usr/libexec/java_home -v 21 2>/dev/null || echo "$JAVA_HOME")"
cd android
./gradlew test
# includes ShareUrlParseTest (table) and RetryingShareTest

./gradlew assembleDebug
# APK: app/build/outputs/apk/debug/app-debug.apk

./scripts/check-secrets.sh
unzip -p app/build/outputs/apk/debug/app-debug.apk classes*.dex 2>/dev/null | strings | grep -Ei 'compose-local-pepper|DEV_TOKEN' && echo 'FAIL: secret in apk' || echo 'ok: no known secret strings'
```

LAN pair (phone and computer on the same Wi-Fi; Compose up):

```bash
docker compose up --build -d
ipconfig getifaddr en0
curl -sS http://127.0.0.1:8080/v1/pairings -H 'Content-Type: application/json' -d '{}'
# Pair in the app with http://<lan-ip>:8080
```

YouTube’s first share row is often only WhatsApp / Copy link. Open **More** / **See all apps** (or share from Chrome). deskdrop should be in that full list.

If the sheet still hides it: open deskdrop while paired, paste the URL into **Send a URL**, tap **Send to queue**. Same `POST /v1/links`.

YouTube share (phone already paired; a **laptop** must already be on the same pairing chain or the share has nowhere to fan out):

1. YouTube → Share → **More** → deskdrop.
2. Toast **Queued for your laptop.**
3. Computer:

```bash
curl -sS http://127.0.0.1:8080/v1/links/pending -H "Authorization: Bearer $LAPTOP"
# one link. Share the same video again from the sheet = new tap = new key = second row.
# Android retrying the *same* send (rotation / 429 retry) reuses the key → 200, same id.

curl -sS http://127.0.0.1:8080/metrics | grep deskdrop_shares_total
```

CI job **android** runs `./gradlew :app:testDebugUnitTest`. See [ci-cd.md](ci-cd.md).
