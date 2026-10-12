#!/usr/bin/env bash
# D5: v1 default secrets and Electron/CORS footguns must not return in product code.
# Docs may name the audit; this scan is tracked files excluding docs/.
set -euo pipefail
cd "$(dirname "$0")/.."

fail=0
# git grep exits 1 when there are no matches.
hits="$(git grep -n -I -E 'dev-token|change-in-production|cors\.AllowAll|nodeIntegration' \
  -- ':!docs/' ':!scripts/check-secrets.sh' || true)"
if [ -n "$hits" ]; then
  echo "$hits" >&2
  echo "FAIL: v1 default/secret strings in product files" >&2
  fail=1
else
  echo "ok: no v1 default/secret strings in product files"
fi

prod="$(git grep -n -I -F 'compose-local-pepper' -- deploy/prod deploy/k8s || true)"
if [ -n "$prod" ]; then
  echo "$prod" >&2
  echo "FAIL: compose-local-pepper in production deploy" >&2
  fail=1
else
  echo "ok: compose-local-pepper not in deploy/prod or deploy/k8s"
fi

exit "$fail"
