module RedmineOpenapiGenerator
  class SpecBuilder
    AR_TO_OPENAPI = {
      'integer'  => 'integer',
      'bigint'   => 'integer',
      'smallint' => 'integer',
      'float'    => 'number',
      'decimal'  => 'number',
      'boolean'  => 'boolean',
      'datetime' => 'string',
      'timestamp'=> 'string',
      'time'     => 'string',
      'date'     => 'string',
      'text'     => 'string',
      'string'   => 'string',
      'binary'   => 'string',
      'json'     => 'object',
      'jsonb'    => 'object',
    }.freeze

    def initialize(introspector, redmine_version: 'unknown')
      @introspector = introspector
      @redmine_version = redmine_version
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

        tag = controller_to_tag(route[:controller])
        unless seen[tag]
          seen[tag] = true
          result << { 'name' => tag }
        end
      end

      result.sort_by { |t| t['name'] }
    end

    def paths
      grouped = {}

      @introspector.routes.each do |route|
        next unless @introspector.api_controllers.key?(route[:controller])

        method = route[:verb].upcase
        next if method == 'HEAD' || method == 'OPTIONS'

        normalized = normalize_path(route[:path])
        grouped[normalized] ||= {}
        grouped[normalized][method] = route
      end

      grouped.each_with_object({}) do |(path, methods), result|
        result[path] = methods.each_with_object({}) do |(method, route), ops|
          ops[method.downcase] = build_operation(route, path)
        end
      end
    end

    def build_operation(route, path)
      tag = controller_to_tag(route[:controller])
      model_name = Introspector::CONTROLLER_MODEL_MAP[route[:controller]]

      op = {
        'tags' => [tag],
        'summary' => operation_summary(route),
        'operationId' => operation_id(route)
      }

      params = build_parameters(path, route)
      op['parameters'] = params unless params.empty?

      if %w[POST PUT PATCH].include?(route[:verb].upcase) && model_name
        op['requestBody'] = build_request_body(model_name)
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
          'schema' => { 'type' => 'integer' }
        }
      end

      if route[:action] == 'index' || route[:action] == 'list'
        params << { 'name' => 'offset', 'in' => 'query', 'schema' => { 'type' => 'integer' } }
        params << { 'name' => 'limit', 'in' => 'query', 'schema' => { 'type' => 'integer' } }
      end

      params
    end

    def build_request_body(model_name)
      {
        'required' => true,
        'content' => {
          'application/json' => {
            'schema' => { '$ref' => "#/components/schemas/#{model_name}" }
          }
        }
      }
    end

    def build_responses(route)
      status = case route[:verb].upcase
               when 'POST' then '201'
               when 'DELETE' then '204'
               else '200'
               end

      {
        status => { 'description' => 'Success' },
        '401' => { '$ref' => '#/components/responses/Unauthorized' },
        '403' => { '$ref' => '#/components/responses/Forbidden' },
        '412' => { '$ref' => '#/components/responses/PreconditionFailed' }
      }
    end

    def schemas
      @introspector.models.each_with_object({}) do |(model_name, columns), result|
        props = columns.each_with_object({}) do |col, h|
          next if col[:name] == 'id' || col[:name] == 'created_on' || col[:name] == 'updated_on' || col[:name] == 'type'

          prop = { 'type' => AR_TO_OPENAPI[col[:type]] || 'string' }
          prop['default'] = col[:default] if col[:default] && !col[:default].empty?
          h[col[:name]] = prop
        end

        result[model_name] = {
          'type' => 'object',
          'properties' => props
        }
      end
    end

    def common_responses
      {
        'Unauthorized' => {
          'description' => 'Authentication required',
          'content' => { 'application/json' => { 'schema' => { 'type' => 'object' } } }
        },
        'Forbidden' => {
          'description' => 'Forbidden',
          'content' => { 'application/json' => { 'schema' => { 'type' => 'object' } } }
        },
        'PreconditionFailed' => {
          'description' => 'Precondition Failed',
          'content' => { 'application/json' => { 'schema' => { 'type' => 'object' } } }
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
      p = p.gsub(/:(\w+)/, '{\1}')
      p = p.gsub('(:format)', '')
      p = p.gsub('(.:format)', '')
      p = p.gsub('.{format}', '')
      p = p.gsub(/\(\.{format}\)/, '')
      p = p.squeeze('/')
      p = '/' if p.empty?
      p
    end

    def controller_to_tag(controller)
      controller.split('/').last.split('_').map(&:capitalize).join(' ')
    end

    def operation_id(route)
      ctrl = route[:controller].split('/').last
      action = route[:action]
      singular = singularize(ctrl)

      case action
      when 'index', 'list' then "get#{capitalize(pluralize(ctrl))}"
      when 'show' then "get#{capitalize(singular)}"
      when 'create' then "create#{capitalize(singular)}"
      when 'update', 'edit' then "update#{capitalize(singular)}"
      when 'destroy', 'delete' then "delete#{capitalize(singular)}"
      else "#{action}#{capitalize(singular)}"
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
