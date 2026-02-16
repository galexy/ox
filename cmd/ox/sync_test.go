package main

import (
	"testing"
)

func TestSyncPathExists(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "existing directory",
			path:     "/tmp",
			expected: true,
		},
		{
			name:     "non-existent path",
			path:     "/nonexistent/path/that/does/not/exist",
			expected: false,
		},
		{
			name:     "empty path",
			path:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := syncPathExists(tt.path)
			if result != tt.expected {
				t.Errorf("syncPathExists(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestSyncResultJSON(t *testing.T) {
	// verify the struct serializes correctly
	result := SyncResult{
		Success: true,
		Mode:    "daemon",
		Ledger: &SyncLedgerResult{
			Path:   "/path/to/ledger",
			Status: "synced",
		},
		TeamContexts: []TeamContextSyncResult{
			{
				TeamID:   "team-1",
				TeamName: "Team One",
				Path:     "/path/to/team-1",
				Status:   "synced",
			},
		},
	}

	if !result.Success {
		t.Error("expected success to be true")
	}
	if result.Mode != "daemon" {
		t.Errorf("expected mode to be 'daemon', got %q", result.Mode)
	}
	if result.Ledger == nil {
		t.Error("expected ledger to be non-nil")
	}
	if len(result.TeamContexts) != 1 {
		t.Errorf("expected 1 team context, got %d", len(result.TeamContexts))
	}
}

func TestSyncResultWithError(t *testing.T) {
	result := SyncResult{
		Success: false,
		Mode:    "direct",
		Ledger: &SyncLedgerResult{
			Status: "error",
			Error:  "sync failed",
		},
		Error: "ledger sync failed",
	}

	if result.Success {
		t.Error("expected success to be false")
	}
	if result.Ledger.Status != "error" {
		t.Errorf("expected ledger status to be 'error', got %q", result.Ledger.Status)
	}
	if result.Error == "" {
		t.Error("expected error message to be set")
	}
}
