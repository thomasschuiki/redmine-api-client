module RedmineOpenapiGenerator
  class Introspector
    attr_reader :routes, :api_controllers, :models

    CONTROLLER_MODEL_MAP = {
      'issues'            => 'Issue',
      'projects'          => 'Project',
      'users'             => 'User',
      'members'           => 'Member',
      'versions'          => 'Version',
      'news'              => 'News',
      'issue_relations'   => 'IssueRelation',
      'issue_statuses'    => 'IssueStatus',
      'trackers'          => 'Tracker',
      'issue_categories'  => 'IssueCategory',
      'enumerations'      => 'IssuePriority',
      'roles'             => 'Role',
      'groups'            => 'Group',
      'custom_fields'     => 'CustomField',
      'queries'           => 'Query',
      'attachments'       => 'Attachment',
      'time_entries'      => 'TimeEntry',
      'boards'            => 'Board',
      'documents'         => 'Document',
      'journals'          => 'Journal',
      'watchers'          => 'Watcher',
      'files'             => 'Attachment',
    }.freeze

    def initialize
      Rails.application.eager_load!
      @routes = collect_api_routes
      @api_controllers = collect_api_controllers
      @models = collect_models
    end

    private

    def collect_api_routes
      routes = []

      Rails.application.routes.routes.each do |route|
        verb = route.verb.to_s
        next if verb.empty?

        controller = route.defaults[:controller]
        action = route.defaults[:action]
        next unless controller && action

        # Include the controller namespace (e.g., "admin/issues")
        namespace = route.defaults[:controller]
        path = route.path.spec.to_s

        routes << {
          verb: verb,
          path: path,
          controller: namespace,
          action: action
        }
      end

      routes
    end

    def collect_api_controllers
      controllers = {}
      excluded = %w[openapi_spec]

      ActionController::Base.descendants.each do |ctrl|
        next unless ctrl.respond_to?(:accept_api_auth_actions)
        next if excluded.include?(ctrl.controller_path.split('/').last)

        actions = ctrl.accept_api_auth_actions
        next if actions.nil? || actions.empty?

        ctrl_name = ctrl.controller_path
        controllers[ctrl_name] = actions.map(&:to_s)
      end

      controllers
    end

    def collect_models
      models = {}

      CONTROLLER_MODEL_MAP.each do |controller, model_name|
        begin
          klass = model_name.constantize
          next unless klass < ActiveRecord::Base

          # Skip if the table doesn't exist (graceful fallback)
          next unless ActiveRecord::Base.connection.table_exists?(klass.table_name)

          models[model_name] = klass.columns.map do |col|
            {
              name: col.name,
              type: col.type.to_s,
              null: col.null,
              default: col.default&.to_s,
              limit: col.limit,
              precision: col.precision,
              scale: col.scale
            }
          end
        rescue NameError, ActiveRecord::StatementInvalid
          next
        end
      end

      models
    end
  end
end
