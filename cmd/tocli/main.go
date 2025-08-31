package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/ettle/strcase"
	"github.com/hamba/cmd/v3"
	_ "github.com/joho/godotenv/autoload"
	"github.com/urfave/cli/v3"
)

const (
	flagConfig      = "config"
	flagDryRun      = "dry-run"
	flagForceRotate = "force-rotate"
	flagLicense     = "license"
	flagLicence     = "licence"
	flagSourceToken = "source.token"
	flagSourceURL   = "source.url"
	flagVaultToken  = "vault.token"
	flagVaultType   = "vault.type"
	flagVaultURL    = "vault.url"
)

var version = "¯\\_(ツ)_/¯"

var flags = cmd.Flags{
	&cli.StringFlag{
		Name:    flagConfig,
		Value:   "./tocli.yaml",
		Usage:   "The path to the configuration file",
		Sources: cli.EnvVars(strcase.ToSNAKE(flagConfig)),
	},
	&cli.StringFlag{
		Name:    flagSourceURL,
		Value:   "https://gitlab.com/api/v4",
		Usage:   "The Source API URL to use",
		Sources: cli.EnvVars(strcase.ToSNAKE(flagSourceURL)),
	},
	&cli.StringFlag{
		Name:     flagSourceToken,
		Value:    "",
		Required: true,
		Usage:    "The Source token to use",
		Sources:  cli.EnvVars(strcase.ToSNAKE(flagSourceToken)),
	},
	&cli.StringFlag{
		Name:    flagVaultType,
		Value:   "1password",
		Usage:   "Which Vault backend to use",
		Sources: cli.EnvVars(strcase.ToSNAKE(flagVaultType)),
	},
	&cli.StringFlag{
		Name:    flagVaultURL,
		Value:   "",
		Usage:   "The Vault API URL to use, required for HashiCorp Vault",
		Sources: cli.EnvVars(strcase.ToSNAKE(flagVaultURL)),
	},
	&cli.StringFlag{
		Name:     flagVaultToken,
		Value:    "",
		Required: true,
		Usage:    "The Vault token to use",
		Sources:  cli.EnvVars(strcase.ToSNAKE(flagVaultToken)),
	},
	&cli.StringFlag{
		Name:    flagLicense,
		Aliases: []string{flagLicence},
		Value:   "",
		Usage:   "The enterprise license to use",
		Sources: cli.EnvVars(strcase.ToSNAKE(flagLicense), strcase.ToSNAKE(flagLicence)),
	},
	&cli.BoolFlag{
		Name:    flagDryRun,
		Value:   false,
		Usage:   "Do a 'dry-run', don't change anything",
		Sources: cli.EnvVars(strcase.ToSNAKE(flagDryRun)),
	},
	&cli.BoolFlag{
		Name:    flagForceRotate,
		Value:   false,
		Usage:   "Force rotation of all tokens by setting RotateBefore to 1 year.",
		Sources: cli.EnvVars(strcase.ToSNAKE(flagForceRotate)),
	},
}.Merge(cmd.LogFlags)

func main() {
	os.Exit(realMain())
}

func realMain() (code int) {
	ui := newTerm()
	cli.RootCommandHelpTemplate = fmt.Sprintf(`%s
EXAMPLES:

	# Rotate personal tokens on GitLab.com using a 1password service account
	tocli --source.token glpat-.... --vault.token ops-ey... \
		--config personal-tokens.yaml --dry-run

	# Rotate tokens on a self-hosted GitLab using HashiCorp Vault (EE version)
	tocli --source.url https://gitlab.example.com/api/v4 --source.token glpat-.... \
		--vault.type hashicorp --vault.url https://vault.example.com --vault.token ... \
		--config gitlab-example-tokens.yaml --dry-run

	# Example configuration
	https://gitlab.com/sickit/token-operator/-/blob/main/pkg/toop/full-config.yaml

SUPPORT: mailto:toop@sickit.eu

`, cli.RootCommandHelpTemplate)

	defer func() {
		if v := recover(); v != nil {
			ui.Error(fmt.Sprintf("Panic: %v\n%s", v, string(debug.Stack())))
			code = 1
			return
		}
	}()

	app := cli.Command{
		Name:    "tocli",
		Usage:   "The token-operator CLI",
		Version: version,
		Action:  runCli,
		Flags:   flags,
		Suggest: true,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := app.Run(ctx, os.Args); err != nil {
		ui.Error(err.Error())
		return 1
	}
	return 0
}
