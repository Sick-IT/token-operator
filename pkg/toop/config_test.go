package toop

import (
	_ "embed"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"gitlab.com/sickit/token-operator/pkg/source"
	"gitlab.com/sickit/token-operator/pkg/token"
)

//go:embed full-config.yaml
var fullConfig string

func TestParseConfig(t *testing.T) {
	config := Config{}
	validate := validator.New()
	dec := yaml.NewDecoder(
		strings.NewReader(fullConfig),
		yaml.Validator(validate),
		yaml.Strict(),
	)
	err := dec.Decode(&config)
	assert.Nil(t, err)
}

func TestConfig_Validate(t *testing.T) {
	type fields struct {
		DefaultRotation *token.Rotation
		Tokens          []token.Config
	}
	var tests = []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "simple personal config",
			fields: fields{
				DefaultRotation: &token.Rotation{
					RotateBefore: 48 * time.Hour,
					Validity:     76 * time.Hour,
				},
				Tokens: []token.Config{
					{
						Name:  "personal-token",
						State: token.TokenStateActive,
						Source: token.Source{
							Name:   "personal-token",
							Scopes: []string{"test-scope"},
							Type:   source.TypePersonal,
						},
						Vault: token.Vault{
							Path:  "myVault",
							Item:  "some-token",
							Field: "password",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "simple group config",
			fields: fields{
				DefaultRotation: &token.Rotation{
					RotateBefore: 48 * time.Hour,
					Validity:     76 * time.Hour,
				},
				Tokens: []token.Config{
					{
						Name:  "group-token",
						State: token.TokenStateActive,
						Source: token.Source{
							Name:   "group-token",
							Scopes: []string{"test-scope"},
							Type:   source.TypeGroup,
							Owner:  "me",
							Role:   "reporter",
						},
						Vault: token.Vault{
							Path:  "myVault",
							Item:  "some-token",
							Field: "password",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "no config",
			fields:  fields{},
			wantErr: true,
		},
		{
			name: "missing tokens",
			fields: fields{
				Tokens: []token.Config{},
			},
			wantErr: true,
		},
		{
			name: "missing rotation",
			fields: fields{
				Tokens: []token.Config{
					{
						Name:  "rotation-token",
						State: token.TokenStateActive,
						Source: token.Source{
							Name: "rotation-token",
						},
						Vault: token.Vault{},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "group token without owner",
			fields: fields{
				DefaultRotation: &token.Rotation{
					RotateBefore: 48 * time.Hour,
					Validity:     76 * time.Hour,
				},
				Tokens: []token.Config{
					{
						Name:  "group-token",
						State: token.TokenStateActive,
						Source: token.Source{
							Name: "group-token",
							Type: source.TypeGroup,
							Role: "reporter",
						},
						Vault: token.Vault{},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "project token without role",
			fields: fields{
				DefaultRotation: &token.Rotation{
					RotateBefore: 48 * time.Hour,
					Validity:     76 * time.Hour,
				},
				Tokens: []token.Config{
					{
						Name:  "group-token",
						State: token.TokenStateActive,
						Source: token.Source{
							Name:  "group-token",
							Type:  source.TypeGroup,
							Owner: "group/project",
						},
						Vault: token.Vault{},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				DefaultRotation: tt.fields.DefaultRotation,
				Tokens:          tt.fields.Tokens,
			}
			if err := c.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
