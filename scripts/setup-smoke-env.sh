#!/usr/bin/env bash
# Bring up a long-lived Redmine smoke-test server (SQLite) for exercising the
# CLI against a real API, then write a working CLI config.
#
# This encodes the setup that previously required trial-and-error:
#   * REDMINE_NO_DB_MIGRATE=1 so the image does NOT auto-migrate on boot
#     (auto-migration races with config injection), migrations are run here.
#   * Migrations run as the container's `redmine` user (docker exec -u redmine).
#     Running rake as root makes sqlite readonly for the app (root-owned db) and
#     forces chown/chmod + restart hacks.
#   * The REST API is disabled by default since Redmine 6.x
#     (config/settings.yml: rest_api_enabled default 0); it must be turned on
#     or API-key auth silently falls back to anonymous and writes fail 403.
#   * db:migrate does not load default data, so trackers/statuses/priorities
#     are empty; load_default_data is required before issue workflows work.
#   * The admin API key is read out of the DB, never typed.
#
# Usage:
#   ./scripts/setup-smoke-env.sh [--seed]
#
# Flags:
#   --seed    also create a demo project "smoke" via the CLI (end-to-end check)
#
# Env overrides:
#   REDMINE_VERSION  image tag (default 6.1.3)
#   SMOKE_PORT       host port for the web UI / API (default 13000)
#
# Writes config to ${XDG_CONFIG_HOME:-~/.config}/redmine-cli/config.yaml and
# prints how to run the CLI. Re-running is safe (idempotent).
set -euo pipefail

REDMINE_VERSION="${REDMINE_VERSION:-6.1.3}"
PORT="${SMOKE_PORT:-13000}"
CONTAINER="rmserver"
VOLUME="redmine-smoke-files"
IMAGE="redmine:${REDMINE_VERSION}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
SEED=0
for arg in "$@"; do
  case "$arg" in
    --seed) SEED=1 ;;
    *) echo "Unknown argument: $arg" >&2; exit 2 ;;
  esac
done

XDG_CONFIG_HOME="${XDG_CONFIG_HOME:-$HOME/.config}"
CONFIG_DIR="${XDG_CONFIG_HOME}/redmine-cli"
CONFIG_FILE="${CONFIG_DIR}/config.yaml"

need() { command -v "$1" >/dev/null 2>&1 || { echo "ERROR: $1 not found" >&2; exit 1; }; }
need docker
need go

echo ">> Smoke env: ${IMAGE} on port ${PORT} (container: ${CONTAINER})"

# --- 1. Container up (reuse if present) -------------------------------------
if docker inspect "${CONTAINER}" >/dev/null 2>&1; then
  STATE="$(docker inspect "${CONTAINER}" --format '{{.State.Status}}')"
  if [ "${STATE}" != "running" ]; then
    echo ">> Container exists but is ${STATE}; starting..."
    docker start "${CONTAINER}" >/dev/null
  else
    echo ">> Container already running; reusing"
  fi
else
  echo ">> Creating container (this pulls ${IMAGE} on first run)..."
  docker run -d --name "${CONTAINER}" \
    -p "${PORT}:3000" \
    -e REDMINE_DB_TYPE=sqlite3 \
    -e REDMINE_NO_DB_MIGRATE=1 \
    -v "${VOLUME}:/usr/src/redmine/files" \
    "${IMAGE}" >/dev/null
fi

# --- 2. Wait for the web server to accept connections -------------------------
echo ">> Waiting for web server on port ${PORT}..."
for i in $(seq 1 60); do
  if (echo > "/dev/tcp/127.0.0.1/${PORT}") 2>/dev/null; then break; fi
  [ "${i}" = 60 ] && { echo "ERROR: server did not come up" >&2; exit 1; }
  sleep 2
done
echo ">> Web server is up"

# --- 3. Normalize ownership, then migrate as the redmine user ------------------
# Any prior root-run rake (or a hand-tweaked container) leaves root-owned files
# (sqlite/redmine.db, db/schema.rb) that make the app hit sqlite readonly or
# schema-dump EACCES. Chown back to the image's `redmine` user — idempotent.
echo ">> Normalizing file ownership for the 'redmine' user..."
docker exec "${CONTAINER}" chown -R redmine:redmine \
  /usr/src/redmine/sqlite \
  /usr/src/redmine/db \
  /usr/src/redmine/log \
  /usr/src/redmine/tmp \
  /usr/src/redmine/files

echo ">> Running migrations as the 'redmine' user..."
docker exec -u redmine "${CONTAINER}" sh -c \
  'cd /usr/src/redmine && bundle exec rake db:migrate 2>&1 | tail -1'

echo ">> Enabling the REST API (disabled by default since Redmine 6.x)..."
docker exec -u redmine "${CONTAINER}" sh -c \
  'cd /usr/src/redmine && bundle exec rails runner "Setting.rest_api_enabled = \"1\""' >/dev/null 2>&1

echo ">> Loading default data (trackers, statuses, priorities)..."
docker exec -u redmine "${CONTAINER}" sh -c \
  'cd /usr/src/redmine && REDMINE_LANG=en bundle exec rake redmine:load_default_data 2>&1 | tail -1'

# --- 4. Read the admin API key out of the DB -----------------------------------
echo ">> Reading admin API key..."
API_KEY="$(docker exec -u redmine "${CONTAINER}" sh -c \
  'cd /usr/src/redmine && bundle exec rails runner "puts User.where(login:\"admin\").first.api_key"' \
  2>/dev/null | grep -oE '[0-9a-f]{40}' | head -1)"
if [ -z "${API_KEY}" ]; then
  echo "ERROR: could not read admin API key" >&2
  exit 1
fi

# --- 5. Write CLI config -------------------------------------------------------
mkdir -p "${CONFIG_DIR}"
umask 077
cat > "${CONFIG_FILE}" <<EOF
url: http://localhost:${PORT}
auth:
  type: api_key
  key: ${API_KEY}
EOF

echo ">> Wrote ${CONFIG_FILE}"
echo ">> Run the CLI with:  XDG_CONFIG_HOME=\"${XDG_CONFIG_HOME}\" go run ./cmd/redmine ..."

# --- 6. Optional end-to-end seed via the CLI -----------------------------------
if [ "${SEED}" = 1 ]; then
  echo ">> Seeding demo project 'smoke'..."
  CLI_BIN="$(mktemp /tmp/redmine-cli-smoke.XXXXXX)"
  trap 'rm -f "${CLI_BIN}"' EXIT
  go build -o "${CLI_BIN}" "${REPO_ROOT}/cmd/redmine"
  if XDG_CONFIG_HOME="${XDG_CONFIG_HOME}" "${CLI_BIN}" project create \
    --name "smoke" --identifier smoke --description "Smoke-test project" >/dev/null 2>&1; then
    echo ">> Created project 'smoke'"
  else
    echo ">> Project create failed (already exists?) — skipping seed"
  fi
fi

echo ">> Done."
