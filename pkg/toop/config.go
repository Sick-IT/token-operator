package toop

import (
	"fmt"

	"github.com/hamba/pkg/v2/errors"
	"gitlab.com/sickit/test-public/pkg/source"
	"gitlab.com/sickit/test-public/pkg/token"
)

const (
	ErrMissingRotation        = errors.Error("missing rotation definition")
	ErrMissingTokenDefinition = errors.Error("missing token definition")
	ErrMissingTokenOwner      = errors.Error("missing token owner for source")
	ErrMissingTokenRole       = errors.Error("missing token role for source")
)

type Config struct {
	Tokens          []token.Config  `yaml:"tokens" validate:"required"`
	DefaultRotation *token.Rotation `yaml:"default_rotation,omitempty"`
	DryRun          bool            `yaml:"dry_run,omitempty"`
	ForceRotate     bool            `yaml:"force_rotate,omitempty"`
	License         string          `yaml:"license,omitempty"`
	Source          Source          `yaml:"source,omitempty"`
	Vault           Vault           `yaml:"vault,omitempty"`
}

type Source struct {
	Url string `yaml:"url"`
}

type Vault struct {
	Url  string `yaml:"url"`
	Type string `yaml:"type"`
}

// Validate checks logical/structural requirements that can't be validated with go-yaml.
func (c *Config) Validate() error {
	if len(c.Tokens) == 0 {
		return ErrMissingTokenDefinition
	}

	for _, t := range c.Tokens {
		if t.Rotation == nil && c.DefaultRotation == nil {
			return fmt.Errorf("invalid config for token source '%s': %w", t.Source.Name, ErrMissingRotation)
		}

		// Group and Project tokens require "owner" and "role"
		switch t.Source.Type {
		case source.TypeGroup:
			fallthrough
		case source.TypeProject:
			if t.Source.Owner == "" {
				return fmt.Errorf("invalid config for token source '%s': %w", t.Source.Name, ErrMissingTokenOwner)
			}
			if t.Source.Role == "" {
				return fmt.Errorf("invalid config for token source '%s': %w", t.Source.Name, ErrMissingTokenRole)
			}
		}
	}

	return nil
}
