# macOS agent

The laptop is a `laptop` device. C3 is a Go binary (`cmd/agent`): claim a pairing code, store the Bearer token in a **0600** config file, poll `GET /v1/links/pending` every 10s, open the default browser, then `POST /v1/links/{id}/ack`. A LaunchAgent starts it at login. The poll interval is the sleep/wake guarantee (no Socket.IO). Windows is Post-v2.

## Why this way

- Pull+ack already lives in the API ([ADR 0007](../adr/0007-pull-vs-push.md), [ADR 0004](../adr/0004-at-least-once-ack.md)). The agent does not hold a socket.
- Config dir `0700`, file `0600`, under the OS user config dir (`~/Library/Application Support/deskdrop/config.json` on macOS). Token is never compiled in.
- In one process, a link is opened **once**; if ack fails, the next cycle retries ack only. If the process dies before ack, the next launch opens again (at-least-once).
- Unsigned builds hit Gatekeeper. Notarization is paid; v2 documents **Open Anyway**.
- `-once` and `-open-file` exist so compose E2E can drive the same binary without Safari.

## Commands

```text
deskdrop-agent pair [-api URL] [-code CODE] [-name NAME] [-config PATH]
deskdrop-agent run  [-config PATH] [-once] [-poll 10s] [-open-file PATH]
deskdrop-agent install [-config PATH]
deskdrop-agent uninstall
```

Default API for local Compose: `http://127.0.0.1:8080`. Tray title `DD` (darwin). Recent URLs are in the config file and the menu.

## Failure modes

- Not paired: `run` exits 1 and prints `deskdrop-agent pair`.
- 401 (revoked): poll stops; tray shows the error (dashboard unpair is C4).
- 429/5xx/network: full-jitter retry, same as Android.
- Non-http(s) pending URL: refused, not opened.
- Sleep 2+ minutes with a pending share: the ticker fires on wake; one browser open, then ack.
- Relaunch after ack: pending is empty, browser stays closed.
- Gatekeeper: macOS blocks the unsigned binary until Open Anyway.

## Verify or demo

```bash
go test ./internal/agent/
go build -o /tmp/deskdrop-agent ./cmd/agent

docker compose up --build -d
tests/e2e/compose_agent.sh
# e2e agent ok … opened=1

# Manual (this Mac, Compose up):
# 1. Phone already paired (C1). Mint a laptop code from the phone token:
curl -sS http://127.0.0.1:8080/v1/pairings -H "Authorization: Bearer $PHONE"
/tmp/deskdrop-agent pair -api http://127.0.0.1:8080 -code ABCD-EFGH
/tmp/deskdrop-agent run
# Share from Android. Browser should open within ~10s.
# Sleep the Mac 2+ minutes with a share in flight; on wake, one tab.

# Start at login:
/tmp/deskdrop-agent install
launchctl list | grep deskdrop
# ls ~/Library/LaunchAgents/com.deskdrop.agent.plist

# Gatekeeper (unsigned):
# System Settings → Privacy & Security → Open Anyway
# or: xattr -d com.apple.quarantine /tmp/deskdrop-agent
```

GitHub Actions agent compile is D1.
