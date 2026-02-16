package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHydrateHint(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantHint string
	}{
		{
			name:     "no git credentials",
			err:      fmt.Errorf("no git credentials found (run 'ox login' first)"),
			wantHint: "ox login",
		},
		{
			name:     "no auth token",
			err:      fmt.Errorf("no auth token (run 'ox login' first)"),
			wantHint: "ox login",
		},
		{
			name:     "empty token",
			err:      fmt.Errorf("git credentials have empty token"),
			wantHint: "ox login",
		},
		{
			name:     "ledger not ready",
			err:      fmt.Errorf("ledger not ready or no repo URL"),
			wantHint: "ox sync",
		},
		{
			name:     "no repo_id",
			err:      fmt.Errorf("no repo_id in project config"),
			wantHint: "ox sync",
		},
		{
			name:     "connection refused",
			err:      fmt.Errorf("batch request failed: connection refused"),
			wantHint: "network connection",
		},
		{
			name:     "timeout",
			err:      fmt.Errorf("batch request failed: context deadline exceeded timeout"),
			wantHint: "network connection",
		},
		{
			name:     "no such host",
			err:      fmt.Errorf("batch request failed: no such host"),
			wantHint: "network connection",
		},
		{
			name:     "HTTP 401",
			err:      fmt.Errorf("LFS batch API returned HTTP 401: unauthorized"),
			wantHint: "ox login",
		},
		{
			name:     "HTTP 403",
			err:      fmt.Errorf("LFS batch API returned HTTP 403: forbidden"),
			wantHint: "ox login",
		},
		{
			name:     "HTTP 404",
			err:      fmt.Errorf("LFS batch API returned HTTP 404: not found"),
			wantHint: "ox sync",
		},
		{
			name:     "unknown error gets generic hint",
			err:      fmt.Errorf("something unexpected happened"),
			wantHint: "ox doctor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hydrateHint(tt.err)

			// original error is preserved
			assert.ErrorIs(t, result, tt.err)

			// actionable hint is present
			assert.Contains(t, result.Error(), tt.wantHint)
		})
	}
}
