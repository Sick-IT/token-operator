package token_operator

import (
	"os"
	"testing"
	"time"

	"github.com/hamba/logger/v2"
	"github.com/hamba/statter/v2"
	"gitlab.com/sickit/test-public/pkg/source"
	"gitlab.com/sickit/test-public/pkg/token"
	"gitlab.com/sickit/test-public/pkg/vault"
	"go.opentelemetry.io/otel/trace"
)

func TestApplication_Reconcile(t *testing.T) {
	type fields struct {
		tokenSource TokenSource
		tokenVault  TokenVault
		log         *logger.Logger
		stats       *statter.Statter
		tracer      trace.Tracer
	}
	type args struct {
		cfg token.Config
	}

	log := logger.New(os.Stdout, logger.LogfmtFormat(), logger.Debug)

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test new token",
			fields: fields{
				NewMockTokenSource(nil),
				NewMockTokenVault(nil), log, nil, nil,
			},
			args:    args{simpleConfigPersonal()},
			wantErr: false,
		},
		{
			name: "Test missing vault item",
			fields: fields{
				NewMockTokenSource(validTokenFromConfig(simpleConfigPersonal())),
				NewMockTokenVault(nil), log, nil, nil,
			},
			args:    args{simpleConfigPersonal()},
			wantErr: false,
		},
		{
			name: "Test new token with exiting vault item",
			fields: fields{
				NewMockTokenSource(nil),
				NewMockTokenVault(vaultItemFromConfig(simpleConfigPersonal())),
				log, nil, nil,
			},
			args:    args{simpleConfigPersonal()},
			wantErr: false,
		},
		{
			name: "Test expired token",
			fields: fields{
				NewMockTokenSource(expiredTokenFromConfig(simpleConfigPersonal())),
				NewMockTokenVault(vaultItemFromConfig(simpleConfigPersonal())),
				log, nil, nil,
			},
			args:    args{simpleConfigPersonal()},
			wantErr: false,
		},
		{
			name: "Test valid token",
			fields: fields{
				NewMockTokenSource(validTokenFromConfig(simpleConfigPersonal())),
				NewMockTokenVault(vaultItemFromConfig(simpleConfigPersonal())),
				log, nil, nil,
			},
			args:    args{simpleConfigPersonal()},
			wantErr: false,
		},
		{
			name: "Test deleted token",
			fields: fields{
				NewMockTokenSource(validTokenFromConfig(deletedConfigPersonal())),
				NewMockTokenVault(nil), log, nil, nil,
			},
			args:    args{deletedConfigPersonal()},
			wantErr: false,
		},
		{
			name: "Test inactive token",
			fields: fields{
				NewMockTokenSource(validTokenFromConfig(inactiveConfigPersonal())),
				NewMockTokenVault(nil), log, nil, nil,
			},
			args:    args{inactiveConfigPersonal()},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Application{
				tokenSource: tt.fields.tokenSource,
				tokenVault:  tt.fields.tokenVault,
				log:         tt.fields.log,
				stats:       tt.fields.stats,
				tracer:      tt.fields.tracer,
			}
			if err := a.Reconcile(tt.args.cfg); (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_maskToken(t *testing.T) {
	type args struct {
		token string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test mask glpat token",
			args: args{"glpat-12345678"},
			want: "glpat-1...8",
		},
		{
			name: "Test mask  token",
			args: args{"12345678"},
			want: "1...8",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maskToken(tt.args.token); got != tt.want {
				t.Errorf("maskToken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func simpleConfigPersonal() token.Config {
	return token.Config{
		Name:  "mock",
		State: token.TokenStateActive,
		Rotation: &token.Rotation{
			RotateBefore: time.Hour * 24,
			Validity:     time.Hour * 24 * 7,
		},
		Source: token.Source{
			Name:        "mock",
			Description: "mock description",
			Scopes:      []string{"api", "read_repository"},
			Type:        source.TypePersonal,
		},
		Vault: token.Vault{
			Path:  "mock-vault",
			Item:  "mock-item",
			Field: "password",
		},
	}
}

func deletedConfigPersonal() token.Config {
	cfg := simpleConfigPersonal()
	cfg.State = token.TokenStateDeleted
	return cfg
}

func inactiveConfigPersonal() token.Config {
	cfg := simpleConfigPersonal()
	cfg.State = token.TokenStateInactive
	return cfg
}

func validTokenFromConfig(cfg token.Config) *token.Token {
	return &token.Token{
		Name:        cfg.Source.Name,
		Description: cfg.Source.Description,
		Scopes:      cfg.Source.Scopes,
		Type:        cfg.Source.Type,
		Owner:       cfg.Source.Owner,
		Expiration:  time.Now().Add(cfg.Rotation.Validity),
	}
}

func expiredTokenFromConfig(cfg token.Config) *token.Token {
	return &token.Token{
		Name:        cfg.Source.Name,
		Description: cfg.Source.Description,
		Scopes:      cfg.Source.Scopes,
		Type:        cfg.Source.Type,
		Owner:       cfg.Source.Owner,
		Expiration:  time.Now(),
	}
}

func vaultItemFromConfig(cfg token.Config) *vault.Item {
	return &vault.Item{
		Name:  "mock",
		Path:  cfg.Vault.Path,
		Field: cfg.Vault.Field,
		Value: "secret",
	}
}

type MockTokenSource struct {
	token *token.Token
}

func NewMockTokenSource(tok *token.Token) *MockTokenSource {
	return &MockTokenSource{
		token: tok,
	}
}

func (ts *MockTokenSource) GetToken(src *token.Source) (*token.Token, error) {
	if ts.token == nil {
		return nil, source.ErrTokenNotFound
	}

	return ts.token, nil
}

func (ts *MockTokenSource) CreateToken(cfg *token.Config) (*token.Token, error) {
	ts.token = &token.Token{
		Name:        cfg.Source.Name,
		Description: cfg.Source.Description,
		Scopes:      cfg.Source.Scopes,
		Type:        cfg.Source.Type,
		Value:       "secret",
		Expiration:  time.Now().Add(cfg.Rotation.Validity),
	}
	return ts.token, nil
}

func (ts *MockTokenSource) RotateToken(cfg *token.Config) (*token.Token, error) {
	ts.token.Expiration = time.Time{}.Add(cfg.Rotation.Validity)
	ts.token.Value = "secret-rotated"
	return ts.token, nil
}

func (ts *MockTokenSource) DeleteToken(src *token.Source) error {
	ts.token = nil
	return nil
}

type MockTokenVault struct {
	item *vault.Item
}

func NewMockTokenVault(itm *vault.Item) *MockTokenVault {
	return &MockTokenVault{
		item: itm,
	}
}
func (tv *MockTokenVault) WithDryRun(dryRun bool) {
}

func (tv *MockTokenVault) GetItem(vlt *token.Vault) (*vault.Item, error) {
	if tv.item == nil {
		return nil, vault.ErrItemNotFound
	}
	return tv.item, nil
}

func (tv *MockTokenVault) CreateItem(vlt *token.Vault, value string) (*vault.Item, error) {
	tv.item = &vault.Item{
		Name:  "mock",
		Path:  vlt.Path,
		Field: vlt.Field,
		Value: value,
	}
	return tv.item, nil
}

func (tv *MockTokenVault) UpdateItem(vlt *token.Vault, value string) error {
	tv.item.Value = value
	return nil
}

func (tv *MockTokenVault) DeleteItem(vlt *token.Vault) error {
	tv.item = nil
	return nil
}
