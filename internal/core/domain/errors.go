package domain

import "errors"

var (
	ErrProviderNotFound                 = errors.New("provider not found")
	ErrAccountNotFound                  = errors.New("account not found")
	ErrProviderIDOrAccountAlreadyExists = errors.New("provider ID or account already exists")
	ErrMissingRequiredProviderAuthData  = errors.New("missing required provider authentication data")
)
