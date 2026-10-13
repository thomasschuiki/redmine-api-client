class OpenapiSpecController < ApplicationController
  accept_api_auth :show
  before_action :require_login
  before_action :authorize_global

  def show
    version = Redmine::VERSION.to_s

    introspector = RedmineOpenapiGenerator::Introspector.new
    builder = RedmineOpenapiGenerator::SpecBuilder.new(introspector, redmine_version: version)

    respond_to do |format|
      format.json { render json: builder.to_h }
    end
  end
end
