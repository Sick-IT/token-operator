//go:build !enterprise

package source

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/hamba/cmd/v3/observe"
	lctx "github.com/hamba/logger/v2/ctx"
	"github.com/sethvargo/go-retry"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/sickit/token-operator/pkg/token"
)

func NewGitLabSource(ctx context.Context, url, token string, obsvr *observe.Observer, opts ...GitLabOption) (*GitLab, error) {
	glab, err := gitlab.NewClient(token, gitlab.WithBaseURL(url))
	if err != nil {
		return nil, fmt.Errorf("failed to create gitlab client: %w", err)
	}

	b := retry.NewExponential(50 * time.Millisecond)
	b = retry.WithMaxRetries(10, b)
	b = retry.WithMaxDuration(30*time.Second, b)

	glsrc := &GitLab{
		client:  glab,
		admin:   false,
		backoff: b,
		log:     obsvr.Log,
		ctx:     ctx,
	}

	for _, opt := range opts {
		opt(glsrc)
	}

	return glsrc, nil
}

func (g *GitLab) GetToken(source *token.Source) (*token.Token, error) {
	if source.Type != TypePersonal {
		return nil, ErrLicenseRequired
	}

	gltoken, err := g.findPersonalToken(source)
	if err != nil {
		return nil, fmt.Errorf("failed to find personal token: %w", err)
	}

	expires, err := time.Parse(time.DateOnly, gltoken.ExpiresAt.String())
	if err != nil {
		return nil, fmt.Errorf("failed to parse expiration: %w", err)
	}

	return &token.Token{
		Name:        gltoken.Name,
		Description: gltoken.Description,
		Scopes:      gltoken.Scopes,
		Value:       "",
		Expiration:  expires,
	}, nil
}

func (g *GitLab) CreateToken(config *token.Config) (*token.Token, error) {
	if config.Source.Type != TypePersonal {
		return nil, ErrLicenseRequired
	}

	if !g.admin {
		g.log.Error("Personal token creation only supported as admin", lctx.Str("name", config.Source.Name))
		return nil, ErrLicenseRequired
	}

	expireISO, err := gitlab.ParseISOTime(time.Now().Add(config.Rotation.Validity).Format(time.DateOnly))
	if err != nil {
		return nil, fmt.Errorf("failed to parse validity: %w", err)
	}

	opt := &gitlab.CreatePersonalAccessTokenForCurrentUserOptions{
		Name:        &config.Source.Name,
		Description: &config.Source.Description,
		Scopes:      &config.Source.Scopes,
		ExpiresAt:   &expireISO,
	}

	if g.dryRun {
		g.log.Info("dry-run flag set, not creating token for current user", lctx.Str("name", config.Source.Name))
		return &token.Token{
			Name:        config.Source.Name,
			Description: config.Source.Description,
			Scopes:      config.Source.Scopes,
			Type:        TypePersonal,
			Expiration:  time.Now().Add(config.Rotation.Validity),
			Value:       "dry-run",
		}, nil
	}

	b := g.backoff
	tok := &gitlab.PersonalAccessToken{}
	resp := &gitlab.Response{}
	err = retry.Do(g.ctx, b, func(ctx context.Context) error {
		var err error
		tok, resp, err = g.client.Users.CreatePersonalAccessTokenForCurrentUser(opt, gitlab.WithContext(g.ctx))
		if retryErr := g.isRetriable(resp, err); retryErr != nil {
			return retryErr
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create personal access token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		g.log.Error("failed to create personal access token", lctx.Str("name", config.Source.Name), lctx.Str("status", resp.Status))
		return nil, ErrTokenCreationFailed
	}

	expire, err := time.Parse(time.DateOnly, tok.ExpiresAt.String())
	if err != nil {
		return nil, fmt.Errorf("failed to parse expiration: %w", err)
	}

	return &token.Token{
		Name:        tok.Name,
		Description: tok.Description,
		Scopes:      tok.Scopes,
		Type:        TypePersonal,
		Owner:       strconv.FormatInt(tok.UserID, 10),
		Expiration:  expire,
	}, nil
}

func (g *GitLab) RotateToken(config *token.Config) (*token.Token, error) {
	if config.Source.Type != TypePersonal {
		return nil, ErrLicenseRequired
	}

	// we need the token ID for rotation, so we find an active token or abort
	gltoken, err := g.findPersonalToken(&config.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to rotate token: %w", err)
	}

	expireISO, err := gitlab.ParseISOTime(time.Now().Add(config.Rotation.Validity).Format(time.DateOnly))
	if err != nil {
		return nil, fmt.Errorf("failed to rotate token: %w", err)
	}

	rtopt := &gitlab.RotatePersonalAccessTokenOptions{
		ExpiresAt: &expireISO,
	}

	if g.dryRun {
		g.log.Info("dry-run flag set, not rotating token for user", lctx.Str("name", config.Source.Name), lctx.Int64("userID", gltoken.UserID))
		return &token.Token{
			Name:        config.Source.Name,
			Description: config.Source.Description,
			Scopes:      config.Source.Scopes,
			Type:        TypePersonal,
			Owner:       strconv.FormatInt(gltoken.UserID, 10),
			Expiration:  time.Now().Add(config.Rotation.Validity),
			Value:       "dry-run",
		}, nil
	}

	b := g.backoff
	tok := &gitlab.PersonalAccessToken{}
	resp := &gitlab.Response{}
	err = retry.Do(g.ctx, b, func(ctx context.Context) error {
		var err error
		tok, resp, err = g.client.PersonalAccessTokens.RotatePersonalAccessToken(gltoken.ID, rtopt, gitlab.WithContext(g.ctx))
		if retryErr := g.isRetriable(resp, err); retryErr != nil {
			return retryErr
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to rotate token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		g.log.Error("failed to rotate token", lctx.Str("name", config.Source.Name), lctx.Str("status", resp.Status))
		return nil, ErrTokenRotationFailed
	}
	g.log.Debug("rotated personal token", lctx.Str("name", tok.Name), lctx.Int64("id", tok.ID))

	expire, err := time.Parse(time.DateOnly, tok.ExpiresAt.String())
	if err != nil {
		return nil, fmt.Errorf("failed to rotate token: %w", err)
	}

	return &token.Token{
		Name:        tok.Name,
		Description: tok.Description,
		Scopes:      tok.Scopes,
		Type:        TypePersonal,
		Owner:       strconv.FormatInt(tok.UserID, 10),
		Value:       tok.Token,
		Expiration:  expire,
	}, nil
}

func (g *GitLab) DeleteToken(source *token.Source) error {
	if source.Type != TypePersonal {
		return ErrLicenseRequired
	}

	gltoken, err := g.findPersonalToken(source)
	if err != nil {
		return fmt.Errorf("failed to find personal token: %w", err)
	}

	if g.dryRun {
		g.log.Info("dry-run flag set, not deleting token for user", lctx.Str("name", source.Name), lctx.Int64("userID", gltoken.UserID))
		return nil
	}

	b := g.backoff
	resp := &gitlab.Response{}
	err = retry.Do(g.ctx, b, func(ctx context.Context) error {
		var err error
		resp, err = g.client.PersonalAccessTokens.RevokePersonalAccessToken(gltoken.ID, gitlab.WithContext(g.ctx))
		if retryErr := g.isRetriable(resp, err); retryErr != nil {
			return retryErr
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to revoke personal token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		g.log.Error("failed to revoke token", lctx.Str("name", source.Name), lctx.Str("status", resp.Status))
		return ErrTokenRevocationFailed
	}

	return nil
}
