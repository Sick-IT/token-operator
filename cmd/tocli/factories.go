package main

import (
	"context"
	"fmt"
	"os"

	"github.com/hamba/cmd/v3/observe"
	"github.com/hamba/cmd/v3/term"
	"github.com/urfave/cli/v3"
	"gitlab.com/sickit/token-operator"
	"gitlab.com/sickit/token-operator/pkg/source"
	"gitlab.com/sickit/token-operator/pkg/vault"
)

func newTerm() term.Term {
	return term.Prefixed{
		ErrorPrefix: "Error: ",
		Term: term.Colored{
			ErrorColor: term.Red,
			Term: term.Basic{
				Writer:      os.Stdout,
				ErrorWriter: os.Stderr,
				Verbose:     false,
			},
		},
	}
}

func newApplication(ctx context.Context, cmd *cli.Command, obsvr *observe.Observer) (*token_operator.Application, error) {
	src, err := newSource(ctx, cmd, obsvr)
	if err != nil {
		return nil, fmt.Errorf("failed to create source: %w", err)
	}

	vlt, err := newVault(ctx, cmd, obsvr)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault: %w", err)
	}
	return token_operator.NewApplication(src, vlt, obsvr), nil
}

func newSource(ctx context.Context, cmd *cli.Command, obsvr *observe.Observer) (token_operator.TokenSource, error) {
	if cmd.String(flagSourceToken) == "" {
		return nil, fmt.Errorf("no token for source specified")
	}

	glsrc, err := source.NewGitLabSource(ctx, cmd.String(flagSourceURL), cmd.String(flagSourceToken), obsvr, source.WithDryRun(cmd.Bool(flagDryRun)))
	if err != nil {
		return nil, fmt.Errorf("failed to create source: %w", err)
	}

	return glsrc, nil
}

func newVault(ctx context.Context, cmd *cli.Command, obsvr *observe.Observer) (token_operator.TokenVault, error) {
	if cmd.String(flagVaultToken) == "" {
		return nil, fmt.Errorf("no token for vault specified")
	}

	var opvlt token_operator.TokenVault
	var err error
	switch cmd.String(flagVaultType) {
	case vault.Type1Password:
		opvlt, err = vault.NewOnePasswordVault(ctx, cmd.String(flagVaultToken), obsvr)
		if err != nil {
			return nil, fmt.Errorf("failed to create vault: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown vault type: %s", cmd.String(flagVaultType))
	}

	opvlt.WithDryRun(cmd.Bool(flagDryRun))

	return opvlt, nil
}
