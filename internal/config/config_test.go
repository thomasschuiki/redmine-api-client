package config

import (
	"testing"
)

func TestAuthValidate(t *testing.T) {
	tests := []struct {
		name string
		auth AuthConfig
		wantErr bool
	}{
		{"valid api_key", AuthConfig{Type: "api_key", APIKey: "key123"}, false},
		{"missing api_key", AuthConfig{Type: "api_key"}, true},
		{"valid basic", AuthConfig{Type: "basic", Username: "u", Password: "p"}, false},
		{"missing basic password", AuthConfig{Type: "basic", Username: "u"}, true},
		{"missing basic username", AuthConfig{Type: "basic", Password: "p"}, true},
		{"valid oauth2", AuthConfig{Type: "oauth2", Token: "tok"}, false},
		{"missing oauth2 token", AuthConfig{Type: "oauth2"}, true},
		{"empty type", AuthConfig{Type: ""}, true},
		{"unknown type", AuthConfig{Type: "bad"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.auth.validate()
			if tt.wantErr && err == nil {
				t.Errorf("validate() expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validate() unexpected error: %v", err)
			}
		})
	}
}
