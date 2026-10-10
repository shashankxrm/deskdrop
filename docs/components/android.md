# Android (pairing)

C1 is pairing only. Share target is C2. The phone is a `phone` device: paste or scan a pairing code, `POST /v1/pairings/claim`, store the token in **EncryptedSharedPreferences**. There is no compiled-in token or pepper.

## Why this way

- The API is HTTP on the LAN for Compose. The user types `http://<computer-lan-ip>:8080`. Emulator: `http://10.0.2.2:8080`.
- Token at rest is AndroidX Security crypto (AES-256). The UI shows device id, never the token.
- Claim retries use **full jitter** backoff on 429/5xx/IO. 404/409/410/4xx do not retry.
- QR payload is the pairing code (`ABCD-EFGH`), optional `deskdrop:` prefix.

## Failure modes

- Empty or non-http(s) API URL: in-app error, no request.
- Incomplete code: in-app error.
- 404/409/410: shown from JSON `error`; no retry.
- Camera missing: paste still works (`camera` is not required).
- Cleartext HTTP: allowed until D0 HTTPS. Do not ship a default token to “make LAN work.”

## Verify or demo

JDK 17+ (`JAVA_HOME`). Android SDK with `compileSdk` 34 (`sdk.dir` in `android/local.properties`, not committed).

```bash
# From repo root
echo "sdk.dir=$HOME/Library/Android/sdk" > android/local.properties

export JAVA_HOME="$(/usr/libexec/java_home -v 21 2>/dev/null || echo "$JAVA_HOME")"
cd android
./gradlew test
# BackoffTest, BackoffRetryTest, PairingCodeTest, RetryingClaimTest green

./gradlew assembleDebug
# APK: app/build/outputs/apk/debug/app-debug.apk

# No compiled-in secret (pepper / fake bearer).
unzip -p app/build/outputs/apk/debug/app-debug.apk classes*.dex 2>/dev/null | strings | grep -Ei 'compose-local-pepper|dev-token' && echo 'FAIL: secret in apk' || echo 'ok: no known secret strings'
```

LAN pair (phone and computer on the same Wi-Fi; Compose up):

```bash
# Computer:
docker compose up --build -d
ipconfig getifaddr en0
# e.g. 192.168.1.20

curl -sS http://127.0.0.1:8080/v1/pairings -H 'Content-Type: application/json' -d '{}'
# JSON code ABCD-EFGH — type it, paste, or put it in a QR

# Phone: API URL http://192.168.1.20:8080 — Pair as phone.
# Status shows device id. Token is not on screen.
```

`./gradlew test` is the C1 suite. GitHub Actions Gradle is D1.
