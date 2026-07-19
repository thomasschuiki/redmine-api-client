#!/usr/bin/env bash
# Generate a routes snapshot from a Redmine Docker image.
#
# This script runs inside a Redmine Docker container to extract:
# 1. The route table from `rake routes`
# 2. API-enabled actions from controller files (accept_api_auth)
#
# Usage:
#   ./scripts/gen-routes-snapshot.sh [redmine_version] [output_path]
#
# Examples:
#   ./scripts/gen-routes-snapshot.sh latest
#   ./scripts/gen-routes-snapshot.sh 7.0 testdata/routes-7.0.yaml
#   ./scripts/gen-routes-snapshot.sh 6.1 testdata/routes-6.1.yaml
set -euo pipefail

REDMINE_VERSION="${1:-latest}"
OUTPUT="${2:-testdata/routes-${REDMINE_VERSION}.yaml}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo ">> Generating routes snapshot for Redmine ${REDMINE_VERSION}"

# Create a temporary dummy database.yml so Rails can boot for `rails routes`
TMPDB=$(mktemp)
cat > "$TMPDB" << 'EOF'
test:
  adapter: sqlite3
  database: ":memory:"
EOF
trap 'rm -f "$TMPDB"' EXIT

# Run the Ruby scanner inside the Docker container
docker run --rm --entrypoint ruby \
  -e RAILS_ENV=test \
  -v "${TMPDB}:/usr/src/redmine/config/database.yml:ro" \
  "redmine:${REDMINE_VERSION}" \
  -e '
require "json"

# Load the database schema into SQLite so we can introspect columns
load "#{Dir.pwd}/db/schema.rb" if File.exist?("db/schema.rb")

# Extract table schemas from ActiveRecord
tables = {}
ActiveRecord::Base.connection.tables.each do |table|
  cols = ActiveRecord::Base.connection.columns(table)
  tables[table] = cols.map { |c|
    col = {
      name: c.name,
      type: c.type.to_s,
      null: c.null
    }
    col[:default] = c.default.to_s if c.default
    col[:limit] = c.limit if c.limit
    col[:precision] = c.precision if c.precision
    col[:scale] = c.scale if c.scale
    col
  }
end

# Parse rails routes output
routes_raw = `bundle exec rails routes`
routes = []
routes_raw.lines.drop(1).each do |line|
  line = line.strip
  next if line.empty?

  # Format: Prefix Verb URI_pattern Controller#Action
  # The prefix may be empty, so we need to handle that
  parts = line.split(/\s+/)
  next if parts.length < 3

  # Find the verb (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS)
  verb_idx = parts.index { |p| p.match?(/^(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)$/) }
  next unless verb_idx

  verb = parts[verb_idx]
  path = parts[verb_idx + 1]
  controller_action = parts[verb_idx + 2]

  next unless path && controller_action

  # Parse controller#action
  if controller_action.include?("#")
    controller, action = controller_action.split("#", 2)
    # Remove any trailing constraints like {:format=>:json}
    action = action.split(/\s+/).first
    routes << {
      verb: verb,
      path: path,
      controller: controller,
      action: action
    }
  end
end

# Scan controllers for accept_api_auth
controllers = {}
Dir.glob("app/controllers/**/*_controller.rb").each do |f|
  content = File.read(f)
  name = File.basename(f, "_controller.rb")

  # Handle namespaced controllers (e.g., admin/issues_controller -> admin/issues)
  rel_path = f.sub("app/controllers/", "").sub("_controller.rb", "")

  if content =~ /accept_api_auth\s+:([\w\s,]+)/
    actions = $1.split(",").map(&:strip)
    controllers[rel_path] = actions
  end
end

# Filter routes to only API-enabled actions
api_routes = routes.select do |r|
  ctrl = r[:controller]
  action = r[:action]

  # Check if this controller has accept_api_auth with this action
  controllers.any? do |ctrl_name, actions|
    ctrl_name == ctrl && actions.include?(action)
  end
end

output = {
  redmine_version: ENV.fetch("REDMINE_VERSION", "unknown"),
  generated_at: Time.now.utc.iso8601,
  api_routes: api_routes.map { |r|
    {
      verb: r[:verb],
      path: r[:path],
      controller: r[:controller],
      action: r[:action]
    }
  },
  api_controllers: controllers,
  tables: tables
}

puts JSON.pretty_generate(output)
' > "${OUTPUT}"

echo ">> Wrote ${OUTPUT}"
echo ">> Done. Snapshot contains routes for Redmine ${REDMINE_VERSION}"
