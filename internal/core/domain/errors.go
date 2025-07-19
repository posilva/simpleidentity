package domain

import "errors"

var (
	ErrProviderNotFound = errors.New("provider not found")
	ErrAccountNotFound  = errors.New("account not found")
)
