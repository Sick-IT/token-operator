package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-yaml"
	"github.com/hamba/cmd/v3/observe"
	lctx "github.com/hamba/logger/v2/ctx"
	errors2 "github.com/hamba/pkg/v2/errors"
	"github.com/urfave/cli/v3"
	"gitlab.com/sickit/token-operator/pkg/token"
	"gitlab.com/sickit/token-operator/pkg/toop"
)

const (
	ErrNoLicense = errors2.Error("no valid license provided")
)

func runCli(ctx context.Context, cmd *cli.Command) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	obsvr, err := observe.New(ctx, cmd, "tocli", &observe.Options{
		StatsRuntime: false,
	})
	if err != nil {
		return fmt.Errorf("failed to create observer: %w", err)
	}
	defer obsvr.Close()

	confFile, err := os.ReadFile(cmd.String(flagConfig))
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	config := toop.Config{}
	validate := validator.New()
	dec := yaml.NewDecoder(
		strings.NewReader(string(confFile)),
		yaml.Validator(validate),
	)
	err = dec.Decode(&config)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	obsvr.Log.Debug("read config", lctx.Str("config", fmt.Sprintf("%+v", config)))

	if err = config.Validate(); err != nil {
		return fmt.Errorf("failed to validate config: %w", err)
	}

	// check if config requires a license
	if err := validateLicense(cmd, obsvr); err != nil {
		if !errors.Is(err, ErrNoLicense) {
			return fmt.Errorf("invalid license: %w", err)
		}

		for _, cfg := range config.Tokens {
			if cfg.Source.Type != "personal" {
				return fmt.Errorf("config requires enterprise license: %s", cfg.Source.Type)
			}
		}
	}

	// set unused flags from config, except credentials
	switch {
	case !cmd.IsSet(flagSourceURL) && config.Source.Url != "":
		err = cmd.Set(flagSourceURL, config.Source.Url)
	case !cmd.IsSet(flagVaultType) && config.Vault.Type != "":
		err = cmd.Set(flagVaultType, config.Vault.Type)
	case !cmd.IsSet(flagVaultURL) && config.Vault.Url != "":
		err = cmd.Set(flagVaultURL, config.Vault.Url)
	case !cmd.IsSet(flagDryRun) && config.DryRun:
		err = cmd.Set(flagDryRun, "true")
	case !cmd.IsSet(flagLicense) && config.License != "":
		err = cmd.Set(flagLicense, config.License)
	}
	if err != nil {
		return fmt.Errorf("failed to set flag from config: %w", err)
	}

	app, err := newApplication(ctx, cmd, obsvr)
	if err != nil {
		return fmt.Errorf("failed to create application: %w", err)
	}

	for _, cfg := range config.Tokens {
		if cfg.Rotation == nil {
			cfg.Rotation = &token.Rotation{
				RotateBefore: config.DefaultRotation.RotateBefore,
				Validity:     config.DefaultRotation.Validity,
			}
			obsvr.Log.Debug("using default rotation for token", lctx.Str("name", cfg.Name), lctx.Duration("rotateBefore", cfg.Rotation.RotateBefore), lctx.Duration("validity", cfg.Rotation.Validity))
		}

		if cmd.Bool(flagForceRotate) {
			// to force rotation, we set rotateBefore to over 1 year (the maximum validity for GitLab tokens).
			cfg.Rotation.RotateBefore = 366 * 24 * time.Hour
			obsvr.Log.Debug("forcing rotation", lctx.Str("name", cfg.Name), lctx.Duration("rotateBefore", cfg.Rotation.RotateBefore))
		}

		obsvr.Log.Info("reconciling token", lctx.Str("name", cfg.Name), lctx.Str("type", cfg.Source.Type))
		if err = app.Reconcile(cfg); err != nil {
			return fmt.Errorf("reconcile error: %w", err)
		}
	}

	obsvr.Log.Debug("token rotation complete, 🙏thank you for using token-operator!")

	return nil
}
