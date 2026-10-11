#!/usr/bin/env bash
# Build unsigned macOS agent (+ optional Android APK) and SHA-256 sums for a GitHub Release.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
mkdir -p dist
rm -f dist/SHA256SUMS

echo "building deskdrop-agent (darwin/arm64)"
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o dist/deskdrop-agent-darwin-arm64 ./cmd/agent

if [[ -x android/gradlew ]]; then
  echo "building Android debug APK"
  (
    cd android
    ./gradlew assembleDebug
  )
  cp android/app/build/outputs/apk/debug/app-debug.apk dist/deskdrop-android-debug.apk
fi

(
  cd dist
  shasum -a 256 deskdrop-agent-darwin-arm64 > SHA256SUMS
  if [[ -f deskdrop-android-debug.apk ]]; then
    shasum -a 256 deskdrop-android-debug.apk >> SHA256SUMS
  fi
  cat SHA256SUMS
)

echo "verify: cd dist && shasum -a 256 -c SHA256SUMS"
echo "publish: gh release create v2-d0 dist/* --title 'v2 closed beta' --notes 'Unsigned agent: Open Anyway. APK: debug.'"
