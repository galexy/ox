package errors

import "errors"

// SageOx is multiplayer - login is required. See internal/auth/feature.go for philosophy.
var (
	ErrNotLoggedIn    = errors.New("not logged in")
	ErrConfigNotFound = errors.New("config file not found")
	ErrInvalidSession = errors.New("invalid session")
	ErrNotInitialized = errors.New("sageox not initialized in this repository")
)
