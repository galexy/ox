package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name     string
		sentinel error
		wrapped  error
	}{
		{
			name:     "ErrNotLoggedIn",
			sentinel: ErrNotLoggedIn,
			wrapped:  fmt.Errorf("%w: Run 'ox login' to authenticate", ErrNotLoggedIn),
		},
		{
			name:     "ErrConfigNotFound",
			sentinel: ErrConfigNotFound,
			wrapped:  fmt.Errorf("%w in directory /path/to/dir", ErrConfigNotFound),
		},
		{
			name:     "ErrInvalidSession",
			sentinel: ErrInvalidSession,
			wrapped:  fmt.Errorf("%w: session expired", ErrInvalidSession),
		},
		{
			name:     "ErrNotInitialized",
			sentinel: ErrNotInitialized,
			wrapped:  fmt.Errorf("%w: SAGEOX.md not found", ErrNotInitialized),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, errors.Is(tt.wrapped, tt.sentinel), "wrapped error does not match sentinel %v", tt.sentinel)
		})
	}
}

func TestSentinelErrorMessages(t *testing.T) {
	tests := []struct {
		name     string
		sentinel error
		expected string
	}{
		{
			name:     "ErrNotLoggedIn",
			sentinel: ErrNotLoggedIn,
			expected: "not logged in",
		},
		{
			name:     "ErrConfigNotFound",
			sentinel: ErrConfigNotFound,
			expected: "config file not found",
		},
		{
			name:     "ErrInvalidSession",
			sentinel: ErrInvalidSession,
			expected: "invalid session",
		},
		{
			name:     "ErrNotInitialized",
			sentinel: ErrNotInitialized,
			expected: "sageox not initialized in this repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.sentinel.Error())
		})
	}
}

func TestDistinguishBetweenDifferentErrors(t *testing.T) {
	wrappedNotLoggedIn := fmt.Errorf("%w: details", ErrNotLoggedIn)
	wrappedConfigNotFound := fmt.Errorf("%w: details", ErrConfigNotFound)

	// wrapped errors should match their own sentinels
	assert.True(t, errors.Is(wrappedNotLoggedIn, ErrNotLoggedIn), "wrappedNotLoggedIn should match ErrNotLoggedIn")

	// but not other sentinels
	assert.False(t, errors.Is(wrappedNotLoggedIn, ErrConfigNotFound), "wrappedNotLoggedIn should not match ErrConfigNotFound")

	// same for the other direction
	assert.True(t, errors.Is(wrappedConfigNotFound, ErrConfigNotFound), "wrappedConfigNotFound should match ErrConfigNotFound")
	assert.False(t, errors.Is(wrappedConfigNotFound, ErrNotLoggedIn), "wrappedConfigNotFound should not match ErrNotLoggedIn")
}
