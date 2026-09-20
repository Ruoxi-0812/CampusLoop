#!/bin/bash
set -Eeuo pipefail
: "${DATABASE_URL:?Set DATABASE_URL to the Neon connection string}"
backend_pid=''
frontend_pid=''
cleanup() {
  trap - EXIT INT TERM
  [[ -z "$frontend_pid" ]] || kill "$frontend_pid" 2>/dev/null || true
  [[ -z "$backend_pid" ]] || kill "$backend_pid" 2>/dev/null || true
  wait 2>/dev/null || true
}
trap cleanup EXIT INT TERM
LISTEN_ADDR=127.0.0.1:8090 /app/marketplace &
backend_pid=$!
ready=false
for ((attempt=0; attempt<60; attempt++)); do
  if wget -q -O /dev/null http://127.0.0.1:8090/healthz; then ready=true; break; fi
  kill -0 "$backend_pid" 2>/dev/null || { echo 'Marketplace failed to start' >&2; exit 1; }
  sleep 1
done
[[ "$ready" == true ]] || { echo 'Marketplace startup timed out' >&2; exit 1; }
if [[ "${SEED_DEMO_CATALOG:-false}" == true ]]; then
  /app/seed -catalog /app/products.json
fi
MARKETPLACE_ONLY=true MARKETPLACE_SERVICE_URL=http://127.0.0.1:8090 LISTEN_ADDR=0.0.0.0 PORT="${PORT:-10000}" /app/frontend/server &
frontend_pid=$!
# Exit the container if either service dies so the host can restart both.
set +e
wait -n "$backend_pid" "$frontend_pid"
status=$?
exit "$status"
