package source

import "github.com/hamba/pkg/v2/errors"

const (
	ErrUnauthorized          = errors.Error("unauthorized")
	ErrForbidden             = errors.Error("forbidden")
	ErrNotFound              = errors.Error("not found")
	ErrTokenNotFound         = errors.Error("token not found")
	ErrTokenCreationFailed   = errors.Error("token could not be created")
	ErrTokenRotationFailed   = errors.Error("token could not be rotated")
	ErrTokenRevocationFailed = errors.Error("token could not be revoked")
	ErrLicenseRequired       = errors.Error("enterprise license is required")
)
