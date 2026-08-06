#!/usr/bin/env bash
# Generate one OpenAPI spec snapshot per supported Redmine version into
# testdata/openapi/. Commit the results so drift is visible in diffs and
# enforced by the snapshot tests and CI.
#
# Usage:
#   ./scripts/gen-spec-snapshots.sh
#   VERSIONS="6.1.3" ./scripts/gen-spec-snapshots.sh   # subset
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SNAP_DIR="${SCRIPT_DIR}/../testdata/openapi"
VERSIONS="${VERSIONS:-5.1.12 6.0.10 6.1.3}"

mkdir -p "${SNAP_DIR}"

for ver in ${VERSIONS}; do
  echo ">> Generating snapshot for Redmine ${ver}"
  "${SCRIPT_DIR}/gen-openapi-spec.sh" "${ver}" "${SNAP_DIR}/redmine-${ver}.yaml"
done

echo ">> Snapshots written to ${SNAP_DIR}"
