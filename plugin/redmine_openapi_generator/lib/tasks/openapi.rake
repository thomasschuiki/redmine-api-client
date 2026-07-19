require_relative '../redmine_openapi_generator/introspector'
require_relative '../redmine_openapi_generator/spec_builder'

namespace :redmine do
  namespace :openapi do
    desc 'Generate OpenAPI spec from Redmine routes and models'
    task generate: :environment do
      output_dir = ENV.fetch('OUTPUT_DIR', 'public')
      format = ENV.fetch('FORMAT', 'yaml')

      version = Redmine::VERSION.to_s

      puts ">> Introspecting Redmine #{version}..."

      introspector = RedmineOpenapiGenerator::Introspector.new
      builder = RedmineOpenapiGenerator::SpecBuilder.new(introspector, redmine_version: version)

      filename = format == 'json' ? 'openapi.json' : 'openapi.yaml'
      output_path = File.join(output_dir, filename)

      FileUtils.mkdir_p(output_dir)

      case format
      when 'json'
        File.write(output_path, builder.to_json)
      else
        File.write(output_path, builder.to_yaml)
      end

      puts ">> Wrote #{output_path}"
      puts ">> #{introspector.routes.length} routes, #{introspector.models.length} models"
    end
  end
end
