Redmine::Plugin.register :redmine_openapi_generator do
  name 'OpenAPI Spec Generator'
  author 'go-redmine-cli'
  description 'Generates an OpenAPI 3.0.3 specification from Redmine routes and models'
  version '0.1.0'
  url 'https://github.com/tom-redmine/go-redmine-cli/releases'

  permission :view_openapi_spec,
    { openapi_spec: [:show] },
    global: true
end
