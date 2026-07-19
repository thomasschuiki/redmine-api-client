#!/usr/bin/env ruby
require File.join(Dir.pwd, 'config', 'environment')

plugin_lib = File.join(Dir.pwd, 'plugins', 'redmine_openapi_generator', 'lib')
require File.join(plugin_lib, 'redmine_openapi_generator', 'introspector')
require File.join(plugin_lib, 'redmine_openapi_generator', 'spec_builder')

introspector = RedmineOpenapiGenerator::Introspector.new
builder = RedmineOpenapiGenerator::SpecBuilder.new(
  introspector,
  redmine_version: Redmine::VERSION.to_s
)

output_path = ENV.fetch('OUTPUT_PATH', '/output/openapi.yaml')
File.write(output_path, builder.to_yaml)
puts ">> #{introspector.routes.length} API routes, #{introspector.models.length} models"
