package domain

import "errors"

var (
	ErrNotFound       = errors.New("not_found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrForbidden      = errors.New("forbidden")
	ErrConflict       = errors.New("conflict")
	ErrValidation     = errors.New("validation_error")
	ErrInvalidState   = errors.New("invalid_state")
	ErrQuotaExceeded  = errors.New("quota_exceeded")
	ErrRateLimited    = errors.New("rate_limited")
	ErrSuppressed     = errors.New("suppressed")
	ErrInvalidToken   = errors.New("invalid_token")
	ErrProviderAuth   = errors.New("provider_auth")
	ErrTransient      = errors.New("transient")
	ErrPermanent      = errors.New("permanent")
)
