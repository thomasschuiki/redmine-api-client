#!/usr/bin/env ruby
# Smoke-test the authenticated OpenAPI endpoint inside a Redmine Rails environment.
# The caller must run this with RAILS_ENV set and a migrated test database.

redmine_root = ENV.fetch('REDMINE_ROOT', '/usr/src/redmine')
require File.join(redmine_root, 'config', 'environment')
require 'action_dispatch/testing/integration'
require 'json'

Setting.rest_api_enabled = '1'
session = ActionDispatch::Integration::Session.new(Rails.application)

session.get('/openapi/spec.json')
abort "anonymous JSON status=#{session.response.status}, want 401" unless session.response.status == 401

api_key = User.find(1).api_key
headers = {'X-Redmine-API-Key' => api_key}

session.get('/openapi/spec.json', headers: headers)
abort "authenticated JSON status=#{session.response.status}" unless session.response.status.between?(200, 299)
json = JSON.parse(session.response.body)
abort 'JSON response is not OpenAPI 3.0.3' unless json['openapi'] == '3.0.3'

puts 'OpenAPI JSON endpoint smoke test passed'
