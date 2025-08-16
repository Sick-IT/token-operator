//go:build !enterprise

package main

import (
	"github.com/hamba/cmd/v3/observe"
	"github.com/urfave/cli/v3"
)

func validateLicense(_ *cli.Command, _ *observe.Observer) error {
	return nil
}
