module RedmineOpenapiGenerator
  require 'json'
  require 'yaml'
  require_relative 'introspector'
  require_relative 'contract'

  class SpecBuilder
    def initialize(introspector, redmine_version: 'unknown')
      @introspector = introspector
      @redmine_version = redmine_version
      @components = Contract.components
    end

    def to_h
      {
        'openapi' => '3.0.3',
        'info' => info,
        'servers' => [{ 'url' => '/' }],
        'security' => security,
        'tags' => tags,
        'paths' => paths,
        'components' => {
          'schemas' => schemas,
          'responses' => common_responses,
          'securitySchemes' => security_schemes
        }
      }
    end

    def to_json
      JSON.pretty_generate(to_h)
    end

    def to_yaml
      YAML.dump(to_h)
    end

    private

    def info
      {
        'title' => 'Redmine API',
        'description' => "Auto-generated OpenAPI specification for Redmine #{@redmine_version}.",
        'version' => "1.0.0+redmine.#{@redmine_version}"
      }
    end

    def security
      [
        { 'Basic' => [] },
        { 'ApiKey' => [] },
        { 'ApiKeyInQuery' => [] },
        { 'OAuth2' => [] }
      ]
    end

    def tags
      seen = {}
      result = []

      @introspector.routes.each do |route|
        next unless @introspector.api_controllers.key?(route[:controller])
        next unless contract_resource?(route)

        tag = controller_to_tag(route[:controller])
        unless seen[tag]
          seen[tag] = true
          result << { 'name' => tag }
        end
      end

      result.sort_by { |t| t['name'] }
    end

    # Only emit routes backed by a contract resource (and thus by schemas).
    # Routes for non-contract controllers (attachments, groups, roles, news,
    # memberships, watchers, files, ...) are filtered out so every operation in
    # the spec carries a schema'd response.
    REST_ACTIONS = %w[index list show create update destroy delete].freeze

    def contract_resource?(route)
      return false unless REST_ACTIONS.include?(route[:action])

      normalized = normalize_path(route[:path])
      if %w[show update destroy delete].include?(route[:action])
        return false unless normalized =~ /\{\w+\}/ || normalized.end_with?('/current')
      end

      model = model_name_for(route)
      model && Contract::RESOURCES.key?(model)
    end

    def paths
      grouped = {}
      meta = []

      @introspector.routes.each do |route|
        next unless @introspector.api_controllers.key?(route[:controller])
        next unless contract_resource?(route)

        method = route[:verb].upcase
        next if %w[HEAD OPTIONS].include?(method)

        normalized = normalize_path(route[:path])
        grouped[normalized] ||= {}
        grouped[normalized][method] = route
        meta << [normalized, method, route]
      end

      counts = Hash.new(0)
      meta.each { |path, method, route| counts[operation_id(route, path)] += 1 }

      used_ids = {}
      path_ops = grouped.each_with_object({}) do |(path, methods), result|
        ops = {}
        methods.each do |method, route|
          base = operation_id(route, path)
          id = base
          if counts[base] > 1
            refined = refined_operation_id(route, path)
            id = refined if refined && !used_ids[refined]
          end
          id = disambiguate(id, used_ids) if used_ids[id]
          used_ids[id] = true
          ops[method.downcase] = build_operation(route, path, id)
        end
        result[path] = ops
      end
      path_ops
    end

    # A nicer operationId for colliding routes, when the path shape suggests one.
    #   /projects/{id}/roadmap          -> getProjectRoadmap
    #   /users/current                  -> getCurrentUser
    # Returns nil when no clean name applies (caller falls back to a numeric suffix).
    def refined_operation_id(route, path)
      verb = verb_for(route[:action])
      ctrl = route[:controller].split('/').last
      segments = path.split('/').reject(&:empty?)
      last = segments.last

      if %w[index list].include?(route[:action]) && last && !last.start_with?('{') && last != 'index' && last != ctrl
        scopes = parent_scopes(path, route)
        return "#{verb}#{scopes.map(&:capitalize).join + capitalize(last)}"
      end

      if route[:action] == 'show' && last == 'current'
        scopes = parent_scopes(path, route)
        return "#{verb}#{scopes.map(&:capitalize).join + 'Current' + resource_noun(route)}"
      end

      nil
    end

    def disambiguate(id, used_ids)
      candidate = id
      n = 2
      while used_ids[candidate]
        candidate = "#{id}#{n}"
        n += 1
      end
      candidate
    end

    def build_operation(route, path, op_id)
      tag = controller_to_tag(route[:controller])
      model_name = model_name_for(route)

      op = {
        'tags' => [tag],
        'summary' => operation_summary(route),
        'operationId' => op_id
      }

      params = build_parameters(path, route)
      op['parameters'] = params unless params.empty?

      if %w[POST PUT PATCH].include?(route[:verb].upcase) && model_name
        schema = if %w[PUT PATCH].include?(route[:verb].upcase) && @components.key?("#{model_name}UpdateRequest")
                   "#{model_name}UpdateRequest"
                 elsif @components.key?("#{model_name}Create")
                   "#{model_name}Create"
                 end
        op['requestBody'] = build_request_body(schema) if schema
      end

      op['responses'] = build_responses(route)
      op
    end

    def build_parameters(path, route)
      params = []

      path.scan(/\{(\w+)\}/).flatten.each do |name|
        params << {
          'name' => name,
          'in' => 'path',
          'required' => true,
          'schema' => { 'type' => path_param_type(name, route) }
        }
      end

      if %w[index list].include?(route[:action])
        params << { 'name' => 'offset', 'in' => 'query', 'schema' => { 'type' => 'integer', 'minimum' => 0 } }
        params << { 'name' => 'limit', 'in' => 'query', 'schema' => { 'type' => 'integer', 'minimum' => 1 } }

        scopes = parent_scopes(path, route).map(&:downcase)
        QUERY_FILTERS.fetch(route[:controller], []).each do |filter|
          next if scopes.include?(filter['name'].sub(/_id\z/, ''))

          params << query_param(filter)
        end
      end

      params
    end

    def query_param(filter)
      param = {
        'name' => filter['name'],
        'in' => 'query',
        'schema' => { 'type' => filter['type'] }
      }
      param['description'] = filter['description'] if filter['description']
      param
    end

    # Per-resource query filters for index/list operations, following the
    # Redmine REST API documentation. Filters are dropped when they duplicate a
    # parent scope of a nested route (e.g. project_id is not repeated on
    # /projects/{id}/issues). Controllers without an entry emit no filters.
    QUERY_FILTERS = {
      'issues' => [
        { 'name' => 'status_id', 'type' => 'string', 'description' => 'Issue status (open, closed, *, or an id)' },
        { 'name' => 'tracker_id', 'type' => 'integer' },
        { 'name' => 'priority_id', 'type' => 'integer' },
        { 'name' => 'author_id', 'type' => 'integer', 'description' => 'Author id (or me)' },
        { 'name' => 'assigned_to_id', 'type' => 'integer', 'description' => 'Assignee id (or me)' },
        { 'name' => 'project_id', 'type' => 'string', 'description' => 'Project id or identifier' },
        { 'name' => 'subproject_id', 'type' => 'string', 'description' => 'Subproject id or identifier' },
        { 'name' => 'category_id', 'type' => 'integer' },
        { 'name' => 'fixed_version_id', 'type' => 'integer' },
        { 'name' => 'subject', 'type' => 'string', 'description' => 'Search within the subject' },
        { 'name' => 'created_on', 'type' => 'string', 'description' => 'Creation date or comparison (e.g. >=2024-01-01)' },
        { 'name' => 'updated_on', 'type' => 'string', 'description' => 'Update date or comparison' },
        { 'name' => 'start_date', 'type' => 'string', 'description' => 'Start date or comparison' },
        { 'name' => 'due_date', 'type' => 'string', 'description' => 'Due date or comparison' },
        { 'name' => 'done_ratio', 'type' => 'integer', 'description' => 'Done ratio or comparison' },
        { 'name' => 'sort', 'type' => 'string', 'description' => 'Sort attributes (comma-separated, e.g. priority:desc)' },
        { 'name' => 'include', 'type' => 'string', 'description' => 'Associated data to include (journals, relations, attachments, watchers)' }
      ],
      'projects' => [
        { 'name' => 'name', 'type' => 'string', 'description' => 'Filter by project name' },
        { 'name' => 'include', 'type' => 'string', 'description' => 'Associated data to include (subprojects)' }
      ],
      'users' => [
        { 'name' => 'name', 'type' => 'string', 'description' => 'Search for users by name' },
        { 'name' => 'group_id', 'type' => 'integer', 'description' => 'Filter users by group' },
        { 'name' => 'status', 'type' => 'integer', 'description' => 'User status (1 active, 2 locked, 3 registered; admins only)' },
        { 'name' => 'include', 'type' => 'string', 'description' => 'Associated data to include (groups, memberships)' }
      ],
      'timelog' => [
        { 'name' => 'project_id', 'type' => 'string', 'description' => 'Project id or identifier' },
        { 'name' => 'issue_id', 'type' => 'integer' },
        { 'name' => 'user_id', 'type' => 'integer' },
        { 'name' => 'from', 'type' => 'string', 'description' => 'Starting date (YYYY-MM-DD)' },
        { 'name' => 'to', 'type' => 'string', 'description' => 'Ending date (YYYY-MM-DD)' },
        { 'name' => 'spent_on', 'type' => 'string', 'description' => 'Date of the time entries (YYYY-MM-DD)' },
        { 'name' => 'include', 'type' => 'string', 'description' => 'Associated data to include (project, issue, user, activity)' },
        { 'name' => 'sort', 'type' => 'string', 'description' => 'Sort attributes (comma-separated)' }
      ],
      'versions' => [
        { 'name' => 'status', 'type' => 'string', 'description' => 'Version status (open, locked, closed)' }
      ],
      'wiki' => [
        { 'name' => 'include', 'type' => 'string', 'description' => 'Associated data to include (attachments)' }
      ]
    }.freeze

    # Path parameters address the resource by contract 'path_key'. Resources keyed
    # by a string handle (projects by identifier, wiki pages by title) take string
    # params. Id-style params (id, project_id, issue_id, ...) are integers; any
    # other param (a type name, a version, a filename) is a string.
    def path_param_type(name, route)
      model = model_name_for(route)
      key = model && Contract::RESOURCES[model]&.fetch('path_key', 'id')
      return 'string' if %w[identifier title].include?(key)
      return 'integer' if name =~ /(?:^|_)id$/

      'string'
    end

    def build_request_body(schema_name)
      {
        'required' => true,
        'content' => {
          'application/json' => {
            'schema' => { '$ref' => "#/components/schemas/#{schema_name}" }
          }
        }
      }
    end

    def build_responses(route)
      model = model_name_for(route)
      status = case route[:verb].upcase
               when 'POST' then '201'
               when 'DELETE' then '204'
               else '200'
               end

      response = { 'description' => success_description(route) }
      schema_name = response_schema(model, route[:action])
      if schema_name && @components.key?(schema_name)
        response['content'] = {
          'application/json' => { 'schema' => { '$ref' => "#/components/schemas/#{schema_name}" } }
        }
      end

      result = { status => response }
      result['401'] = { '$ref' => '#/components/responses/Unauthorized' }
      result['403'] = { '$ref' => '#/components/responses/Forbidden' }
      result['412'] = { '$ref' => '#/components/responses/PreconditionFailed' }
      if %w[show update edit destroy delete].include?(route[:action])
        result['404'] = { '$ref' => '#/components/responses/NotFound' }
      end
      if %w[create update edit].include?(route[:action])
        result['422'] = { '$ref' => '#/components/responses/UnprocessableEntity' }
      end
      result
    end

    def success_description(route)
      case route[:verb].upcase
      when 'POST' then 'Created'
      when 'DELETE' then 'Deleted'
      else 'Success'
      end
    end

    # The response schema for an action: a {Name}Single wrapper for object
    # endpoints, a {Name}List wrapper for collection endpoints.
    def response_schema(model, action)
      case action
      when 'index', 'list' then "#{model}List" if model
      when 'show', 'create', 'update', 'edit' then "#{model}Single" if model
      else nil
      end
    end

    def model_name_for(route)
      Introspector::CONTROLLER_MODEL_MAP[route[:controller]]
    end

    def schemas
      resolve_refs(@components)
    end

    def resolve_refs(value)
      case value
      when Hash
        if value.key?('$ref') && value['$ref'].is_a?(String) && !value['$ref'].start_with?('#')
          { '$ref' => "#/components/schemas/#{value['$ref']}" }
        else
          value.each_with_object({}) do |(k, v), h|
            h[k] = resolve_refs(v)
          end
        end
      when Array
        value.map { |v| resolve_refs(v) }
      else
        value
      end
    end

    def common_responses
      {
        'Unauthorized' => {
          'description' => 'Authentication required',
          'content' => { 'application/json' => { 'schema' => { '$ref' => '#/components/schemas/Error' } } }
        },
        'Forbidden' => {
          'description' => 'Forbidden',
          'content' => { 'application/json' => { 'schema' => { '$ref' => '#/components/schemas/Error' } } }
        },
        'PreconditionFailed' => {
          'description' => 'Precondition Failed',
          'content' => { 'application/json' => { 'schema' => { '$ref' => '#/components/schemas/Error' } } }
        },
        'NotFound' => {
          'description' => 'Resource not found',
          'content' => { 'application/json' => { 'schema' => { '$ref' => '#/components/schemas/Error' } } }
        },
        'UnprocessableEntity' => {
          'description' => 'Validation failed',
          'content' => { 'application/json' => { 'schema' => { '$ref' => '#/components/schemas/Error' } } }
        }
      }
    end

    def security_schemes
      {
        'Basic' => {
          'type' => 'http',
          'scheme' => 'basic',
          'description' => 'HTTP Basic Authentication'
        },
        'ApiKey' => {
          'type' => 'apiKey',
          'in' => 'header',
          'name' => 'X-Redmine-API-Key',
          'description' => 'API key passed as a header'
        },
        'ApiKeyInQuery' => {
          'type' => 'apiKey',
          'in' => 'query',
          'name' => 'key',
          'description' => 'API key passed as a query parameter'
        },
        'OAuth2' => {
          'type' => 'oauth2',
          'flows' => {
            'authorizationCode' => {
              'authorizationUrl' => '/oauth/authorize',
              'tokenUrl' => '/oauth/token',
              'scopes' => { 'api' => 'Redmine API access' }
            }
          }
        }
      }
    end

    def normalize_path(path)
      p = path.gsub(/\{(\w+):[^}]+\}/, '{\1}')
      p = p.gsub(/\(\.?:?format\)/, '')
      p = p.sub(/\.:?format$/, '')
      p = p.gsub(/:(\w+)/, '{\1}')
      p = p.squeeze('/')
      p = '/' if p.empty?
      p
    end

    def controller_to_tag(controller)
      controller.split('/').last.split('_').map(&:capitalize).join(' ')
    end

    # Deterministic operationId: verb + optional parent scopes + resource noun.
    # Nested routes (e.g. /projects/{project_id}/issues) get a scope prefix
    # ("getProjectIssues"); parent scopes that are already implied by the noun
    # (e.g. "issue" + IssueRelation) are dropped. Collisions are de-duplicated
    # with a numeric suffix in #paths.
    def operation_id(route, path)
      verb = verb_for(route[:action])
      noun = resource_noun(route)
      scopes = parent_scopes(path, route)
      "#{verb}#{scopes.map(&:capitalize).join + noun}"
    end

    def verb_for(action)
      case action
      when 'index', 'list' then 'get'
      when 'show' then 'get'
      when 'create' then 'create'
      when 'update', 'edit' then 'update'
      when 'destroy', 'delete' then 'delete'
      else action
      end
    end

    def resource_noun(route)
      model = model_name_for(route)
      if model
        return pluralize_model(model) if %w[index list].include?(route[:action])

        return model
      end

      ctrl = route[:controller].split('/').last
      %w[index list].include?(route[:action]) ? pluralize(ctrl) : singularize(ctrl)
    end

    # Parent resource scopes derived from path parameters, e.g. /projects/{id}/issues
    # yields ["project"]. The resource's own trailing parameter (show/update/delete)
    # is excluded.
    def parent_scopes(path, route)
      segments = path.split('/').reject(&:empty?)
      noun = resource_noun(route).downcase
      scopes = []

      segments.each_with_index do |seg, i|
        next unless seg =~ /\A\{(\w+)\}\z/
        next unless i.positive? && !segments[i - 1].start_with?('{')

        scope = singularize(segments[i - 1]).downcase
        next if noun.start_with?(scope)

        scopes << scope
      end

      scopes.pop if %w[show update edit destroy delete].include?(route[:action]) && scopes.any?
      scopes
    end

    def pluralize_model(model)
      case model
      when 'TimeEntry' then 'TimeEntries'
      when 'IssueCategory' then 'IssueCategories'
      when 'IssuePriority' then 'IssuePriorities'
      when 'IssueRelation' then 'IssueRelations'
      when 'IssueStatus' then 'IssueStatuses'
      when 'WikiPage' then 'WikiPages'
      when 'CustomField' then 'CustomFields'
      when 'News' then 'News'
      else "#{model}s"
      end
    end

    def operation_summary(route)
      ctrl = route[:controller].split('/').last
      action = route[:action]
      singular = singularize(ctrl)

      case action
      when 'index', 'list' then "List #{singularize(ctrl)}"
      when 'show' then "Get #{singular}"
      when 'create' then "Create #{singular}"
      when 'update', 'edit' then "Update #{singular}"
      when 'destroy', 'delete' then "Delete #{singular}"
      else "#{capitalize(action)} #{singular}"
      end
    end

    def capitalize(s)
      s.to_s.empty? ? '' : s[0].upcase + s[1..]
    end

    def singularize(s)
      case s
      when /ies$/ then s.sub(/ies$/, 'y')
      when /ses$/, /xes$/, /hes$/, /ches$/ then s.sub(/s$/, '')
      when /s$/ then s.sub(/s$/, '')
      else s
      end
    end

    def pluralize(s)
      case s
      when /[^aeiou]y$/ then s.sub(/y$/, 'ies')
      when /s$/, /x$/, /h$/, /ch$/ then s + 'es'
      else s + 's'
      end
    end
  end
end
