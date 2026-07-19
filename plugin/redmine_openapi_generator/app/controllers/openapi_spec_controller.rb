class OpenapiSpecController < ApplicationController
  accept_api_auth :show

  def show
    version = Setting.plugin_redmine_openapi_generator&.fetch('redmine_version', nil)
    version ||= Redmine::VERSION.to_s

    introspector = RedmineOpenapiGenerator::Introspector.new
    builder = RedmineOpenapiGenerator::SpecBuilder.new(introspector, redmine_version: version)

    respond_to do |format|
      format.json { render json: builder.to_h }
      format.yaml { render yaml: builder.to_h }
    end
  end
end
