#!/usr/bin/env bash
# Generate an OpenAPI spec by injecting the redmine_openapi_generator plugin
# into a Redmine Docker container, running migrations, and introspecting routes/models.
#
# Usage:
#   ./scripts/gen-openapi-spec.sh [redmine_version] [output_path]
#
# Examples:
#   ./scripts/gen-openapi-spec.sh latest
#   ./scripts/gen-openapi-spec.sh 7.0 docs/openapi/openapi.yaml
set -euo pipefail

REDMINE_VERSION="${1:-latest}"
OUTPUT="${2:-docs/openapi/openapi.yaml}"
CONTAINER_NAME="redmine-openapi-gen-$$"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLUGIN_DIR="${SCRIPT_DIR}/../plugin/redmine_openapi_generator"

echo ">> Generating OpenAPI spec for Redmine ${REDMINE_VERSION}"

if [ ! -d "$PLUGIN_DIR" ]; then
  echo "ERROR: Plugin directory not found: $PLUGIN_DIR" >&2
  exit 1
fi

TMPOUT=$(mktemp -d)
trap 'rm -rf "$TMPOUT"; docker rm -f redmine-mysql-$$ 2>/dev/null; docker rm -f ${CONTAINER_NAME} 2>/dev/null' EXIT

# Start MySQL
echo ">> Starting MySQL..."
docker run --rm --name "redmine-mysql-$$" \
  -e MYSQL_ROOT_PASSWORD=root \
  -e MYSQL_DATABASE=redmine_test \
  -d mysql:8.0 >/dev/null

# Wait for MySQL
docker exec "redmine-mysql-$$" sh -c \
  'until mysqladmin ping -h localhost -uroot -proot 2>/dev/null; do sleep 2; done'

echo ">> MySQL ready"

# Run migrations + introspection in one container
echo ">> Running migrations and introspecting..."
docker run --rm --name "${CONTAINER_NAME}" \
  --link "redmine-mysql-$$:mysql" \
  -v "${PLUGIN_DIR}:/usr/src/redmine/plugins/redmine_openapi_generator:ro" \
  -v "${TMPOUT}:/output" \
  "redmine:${REDMINE_VERSION}" sh -c '
cat > /usr/src/redmine/config/database.yml << DEOF
test:
  adapter: mysql2
  host: mysql
  port: 3306
  database: redmine_test
  username: root
  password: root
  encoding: utf8mb4
DEOF

cd /usr/src/redmine
RAILS_ENV=test bundle exec rake db:migrate 2>&1 | tail -1

RAILS_ENV=test bundle exec ruby plugins/redmine_openapi_generator/generate_spec.rb
'

# Copy output to final destination
mkdir -p "$(dirname "$OUTPUT")"
cp "${TMPOUT}/openapi.yaml" "$OUTPUT"

echo ">> Wrote ${OUTPUT}"
