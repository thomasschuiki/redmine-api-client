Rails.application.routes.draw do
  get 'openapi/spec', to: 'openapi_spec#show'
end
