package token_operator

import (
	"errors"
	"fmt"
	"time"

	"github.com/hamba/cmd/v3/observe"
	"github.com/hamba/logger/v2"
	lctx "github.com/hamba/logger/v2/ctx"
	"github.com/hamba/statter/v2"
	"gitlab.com/sickit/token-operator/pkg/source"
	"gitlab.com/sickit/token-operator/pkg/token"
	"gitlab.com/sickit/token-operator/pkg/vault"
	"go.opentelemetry.io/otel/trace"
)

// interface for tokenSource
type TokenSource interface {
	GetToken(source *token.Source) (*token.Token, error)
	CreateToken(config *token.Config) (*token.Token, error)
	RotateToken(config *token.Config) (*token.Token, error)
	DeleteToken(source *token.Source) error
}

// interface for tokenVault
type TokenVault interface {
	WithDryRun(dryRun bool)
	GetItem(vault *token.Vault) (*vault.Item, error)
	CreateItem(vault *token.Vault, value string) (*vault.Item, error)
	UpdateItem(vault *token.Vault, value string) error
	DeleteItem(vault *token.Vault) error
}

// Application represents the application.
type Application struct {
	tokenSource TokenSource
	tokenVault  TokenVault

	log    *logger.Logger
	stats  *statter.Statter
	tracer trace.Tracer
}

// NewApplication creates an instance of Application.
func NewApplication(source TokenSource, vault TokenVault, obsvr *observe.Observer) *Application {
	return &Application{
		tokenSource: source,
		tokenVault:  vault,

		log:    obsvr.Log,
		stats:  obsvr.Stats,
		tracer: obsvr.Tracer("app"),
	}
}

// Reconcile token based on its state.
func (a *Application) Reconcile(cfg token.Config) error {
	switch cfg.State {
	case token.TokenStateInactive:
		a.log.Info("token state is inactive, skipping", lctx.Str("name", cfg.Name))
		return nil
	case token.TokenStateDeleted:
		return a.Delete(cfg)
	case token.TokenStateActive:
		return a.Update(cfg)
	}

	return fmt.Errorf("invalid token state: %s", cfg.State)
}

// Update rotates the given token, if needed, and updates it in the configured vault.
func (a *Application) Update(cfg token.Config) error {
	vaultItemExists := true
	tokenExists := true

	itm, err := a.tokenVault.GetItem(&cfg.Vault)
	if err != nil {
		if !errors.Is(err, vault.ErrItemNotFound) {
			return fmt.Errorf("failed to get vault item: %w", err)
		}
		vaultItemExists = false
	}

	tok, err := a.tokenSource.GetToken(&cfg.Source)
	if err != nil {
		if !errors.Is(err, source.ErrTokenNotFound) {
			return fmt.Errorf("failed to get token: %w", err)
		}
		tokenExists = false
	}

	switch {
	case tokenExists && vaultItemExists:
		if tok.Expiration.After(time.Now().Add(cfg.Rotation.RotateBefore)) && itm != nil && itm.Value != "" {
			a.log.Info("skipping rotation, vault item available and token still valid",
				lctx.Str("name", cfg.Name),
				lctx.Str("secret", maskToken(itm.Value)),
				lctx.Duration("rotateBefore", cfg.Rotation.RotateBefore),
				lctx.Duration("expireDuration", time.Until(tok.Expiration)),
				lctx.Str("expireDate", tok.Expiration.String()),
			)
			return nil
		}

		a.log.Info("rotating token",
			lctx.Str("name", cfg.Name),
			lctx.Duration("rotateBefore", cfg.Rotation.RotateBefore),
			lctx.Duration("expireDuration", time.Until(tok.Expiration)),
			lctx.Str("expireDate", tok.Expiration.String()),
		)
		tok, err = a.tokenSource.RotateToken(&cfg)
		if err != nil {
			return fmt.Errorf("failed to rotate token: %w", err)
		}

		a.log.Info("updating vault item", lctx.Str("path", cfg.Vault.Path), lctx.Str("item", cfg.Vault.Item))
		err = a.tokenVault.UpdateItem(&cfg.Vault, tok.Value)
		if err != nil {
			return fmt.Errorf("failed to update vault item: %w", err)
		}

	case tokenExists && !vaultItemExists:
		a.log.Info("rotating token",
			lctx.Str("name", cfg.Name),
			lctx.Duration("rotateBefore", cfg.Rotation.RotateBefore),
			lctx.Duration("expireDuration", time.Until(tok.Expiration)),
			lctx.Str("expireDate", tok.Expiration.String()),
		)
		tok, err = a.tokenSource.RotateToken(&cfg)
		if err != nil {
			return fmt.Errorf("failed to rotate token: %w", err)
		}

		a.log.Info("creating vault item", lctx.Str("path", cfg.Vault.Path), lctx.Str("item", cfg.Vault.Item))
		_, err = a.tokenVault.CreateItem(&cfg.Vault, tok.Value)
		if err != nil {
			return fmt.Errorf("failed to create vault item: %w", err)
		}

	case !tokenExists && vaultItemExists:
		a.log.Info("creating new token", lctx.Str("name", cfg.Name))
		tok, err = a.tokenSource.CreateToken(&cfg)
		if err != nil {
			return fmt.Errorf("failed to create token: %w", err)
		}

		a.log.Info("updating vault item", lctx.Str("path", cfg.Vault.Path), lctx.Str("item", cfg.Vault.Item))
		err = a.tokenVault.UpdateItem(&cfg.Vault, tok.Value)
		if err != nil {
			return fmt.Errorf("failed to update vault item: %w", err)
		}

	case !tokenExists && !vaultItemExists:
		a.log.Info("creating new token", lctx.Str("name", cfg.Name))
		tok, err = a.tokenSource.CreateToken(&cfg)
		if err != nil {
			return fmt.Errorf("failed to create token: %w", err)
		}

		a.log.Info("creating vault item", lctx.Str("path", cfg.Vault.Path), lctx.Str("item", cfg.Vault.Item))
		_, err = a.tokenVault.CreateItem(&cfg.Vault, tok.Value)
		if err != nil {
			return fmt.Errorf("failed to create vault item: %w", err)
		}
	}

	return nil
}

// Delete removes a token from source and vault.
func (a *Application) Delete(cfg token.Config) error {
	a.log.Info("deleting token in source", lctx.Str("cfg", cfg.Name))
	if err := a.tokenSource.DeleteToken(&cfg.Source); err != nil {
		if !errors.Is(err, source.ErrTokenNotFound) {
			return fmt.Errorf("failed to delete token: %w", err)
		}
		a.log.Debug("token already deleted", lctx.Str("cfg", cfg.Name))
	}

	a.log.Info("deleting item in vault", lctx.Str("cfg", cfg.Name))
	if err := a.tokenVault.DeleteItem(&cfg.Vault); err != nil {
		if !errors.Is(err, vault.ErrItemNotFound) {
			return fmt.Errorf("failed to delete vault item: %w", err)
		}
		a.log.Debug("vault item already deleted", lctx.Str("cfg", cfg.Name))
	}

	return nil
}

func maskToken(token string) string {
	const gitlab_prefix = "glpat-"
	if token[0:len(gitlab_prefix)] == gitlab_prefix {
		return token[0:len(gitlab_prefix)+1] + "..." + token[len(token)-1:]
	}

	return token[0:1] + "..." + token[len(token)-1:]
}
