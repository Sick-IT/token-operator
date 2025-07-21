package source

import "github.com/hamba/pkg/v2/errors"

const (
	ErrProjectNotFound       = errors.Error("project not found")
	ErrGroupNotFound         = errors.Error("group not found")
	ErrTokenNotFound         = errors.Error("token not found")
	ErrTokenCreationFailed   = errors.Error("token could not be created")
	ErrTokenRotationFailed   = errors.Error("token could not be rotated")
	ErrTokenRevocationFailed = errors.Error("token could not be revoked")
	ErrOperationNotSupported = errors.Error("this operation is not supported")
	ErrLicenseRequired       = errors.Error("enterprise license is required")
)
