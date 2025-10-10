package token

import (
	"time"
)

// Config defines settings for a specific token.
type Config struct {
	Name     string     `yaml:"name" validate:"required"`
	State    TokenState `yaml:"state" validate:"required"`
	Rotation *Rotation  `yaml:"rotation,omitempty"`
	Source   Source     `yaml:"source" validate:"required"`
	Vault    Vault      `yaml:"vault" validate:"required"`
}

type TokenState string

const (
	TokenStateActive   TokenState = "active"
	TokenStateInactive TokenState = "inactive"
	TokenStateDeleted  TokenState = "deleted"
)

// Status represents the status of a token synchronization.
type Status struct {
	SourceID  string `yaml:"token_id"`
	VaultID   string `yaml:"vault_id"`
	ItemID    string `yaml:"item_id"`
	ExpiresAt string `yaml:"expires_at"`
}

// Rotation defines the validity and
type Rotation struct {
	RotateBefore time.Duration `yaml:"rotate_before" validate:"required"`
	Validity     time.Duration `yaml:"validity" validate:"required"`
}

// Source defines the source of a token.
type Source struct {
	Name        string   `yaml:"name" validate:"required"`
	Description string   `yaml:"description"`
	Type        string   `yaml:"type" validate:"required"` // personal, project, group, ...
	Owner       string   `yaml:"owner"`                    // user/project/group ID or full name
	Role        string   `yaml:"role"`
	Scopes      []string `yaml:"scopes" validate:"required"`
}

// Vault defines the target vault item for a token.
type Vault struct {
	// OrgID is an optional organization ID, required for bitwarden
	OrgID string `yaml:"orgID"`
	// PathID is an optional vault or project ID, used by 1password as vault ID
	PathID string `yaml:"pathID"`
	// ItemID is an optional ID for a vault item, used by 1password as item ID
	ItemID string `yaml:"itemID"`
	// Path is the name of the vault or project
	Path string `yaml:"path" validate:"required"`
	// Item is the name of the vault item
	Item string `yaml:"item" validate:"required"`
	// Field is the name of the password field, does not apply for bitwarden
	Field string `yaml:"field" validate:"required"`
}
