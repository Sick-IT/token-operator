package source

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hamba/logger/v2"
	lctx "github.com/hamba/logger/v2/ctx"
	"github.com/sethvargo/go-retry"
	"gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/sickit/token-operator/pkg/token"
)

const (
	TypePersonal = "personal"
	TypeProject  = "project"
	TypeGroup    = "group"
)

type GitLabOption func(*GitLab)

func WithDryRun(dryRun bool) GitLabOption {
	return func(g *GitLab) {
		g.dryRun = dryRun
	}
}

// GitLab implements the application TokenSource for GitLab tokens.
type GitLab struct {
	client  *gitlab.Client
	admin   bool
	dryRun  bool
	backoff retry.Backoff

	log *logger.Logger
	ctx context.Context
}

func (g *GitLab) findPersonalToken(source *token.Source) (*gitlab.PersonalAccessToken, error) {
	lsopt := &gitlab.ListPersonalAccessTokensOptions{
		Search: gitlab.Ptr(source.Name),
		// Info: we cannot rotate inactive tokens.
		State: gitlab.Ptr(string(gitlab.AccessTokenStateActive)),
	}

	b := g.backoff
	toks := []*gitlab.PersonalAccessToken{}
	resp := &gitlab.Response{}
	err := retry.Do(g.ctx, b, func(ctx context.Context) error {
		var err error
		toks, resp, err = g.client.PersonalAccessTokens.ListPersonalAccessTokens(lsopt, gitlab.WithContext(g.ctx))
		if retryErr := g.isRetriable(resp, err); retryErr != nil {
			return retryErr
		}

		return err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list personal tokens: %w", err)
	}

	gltoken := &gitlab.PersonalAccessToken{}
	for _, tok := range toks {
		if tok.Name == source.Name {
			g.log.Debug("matching personal token", lctx.Str("name", tok.Name), lctx.Int("id", tok.ID))
			gltoken = tok
			break
		}
	}

	if gltoken.ID == 0 {
		return nil, ErrTokenNotFound
	}

	return gltoken, nil
}

func (g *GitLab) isRetriable(resp *gitlab.Response, err error) error {
	if err == nil {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return ErrUnauthorized
		case http.StatusForbidden:
			return ErrForbidden
		case http.StatusNotFound:
			return ErrNotFound
		}
		return nil
	}

	g.log.Debug("retry on err", lctx.Err(err))
	return retry.RetryableError(err)
}
