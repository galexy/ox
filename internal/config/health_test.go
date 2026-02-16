package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadHealth_NoFile(t *testing.T) {
	tmpDir := t.TempDir()

	health, err := LoadHealth(tmpDir)
	require.NoError(t, err)
	assert.NotNil(t, health)
	assert.True(t, health.LastDoctorAt.IsZero())
	assert.True(t, health.LastDoctorFixAt.IsZero())
}

func TestLoadHealth_EmptyProjectRoot(t *testing.T) {
	health, err := LoadHealth("")
	require.NoError(t, err)
	assert.NotNil(t, health)
}

func TestSaveAndLoadHealth(t *testing.T) {
	tmpDir := t.TempDir()

	// create .sageox directory
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, sageoxDir), 0755))

	original := &Health{
		LastDoctorAt:    time.Now().UTC().Add(-24 * time.Hour),
		LastDoctorFixAt: time.Now().UTC().Add(-48 * time.Hour),
	}

	err := SaveHealth(tmpDir, original)
	require.NoError(t, err)

	loaded, err := LoadHealth(tmpDir)
	require.NoError(t, err)

	// compare within 1 second (JSON may lose subsecond precision)
	assert.WithinDuration(t, original.LastDoctorAt, loaded.LastDoctorAt, time.Second)
	assert.WithinDuration(t, original.LastDoctorFixAt, loaded.LastDoctorFixAt, time.Second)
}

func TestHealth_RecordDoctorRun(t *testing.T) {
	h := &Health{}
	assert.True(t, h.LastDoctorAt.IsZero())

	before := time.Now()
	h.RecordDoctorRun()
	after := time.Now()

	assert.False(t, h.LastDoctorAt.IsZero())
	assert.True(t, h.LastDoctorAt.After(before) || h.LastDoctorAt.Equal(before))
	assert.True(t, h.LastDoctorAt.Before(after) || h.LastDoctorAt.Equal(after))
}

func TestHealth_RecordDoctorFixRun(t *testing.T) {
	h := &Health{}
	assert.True(t, h.LastDoctorFixAt.IsZero())

	before := time.Now()
	h.RecordDoctorFixRun()
	after := time.Now()

	assert.False(t, h.LastDoctorFixAt.IsZero())
	assert.True(t, h.LastDoctorFixAt.After(before) || h.LastDoctorFixAt.Equal(before))
	assert.True(t, h.LastDoctorFixAt.Before(after) || h.LastDoctorFixAt.Equal(after))
}

func TestHealth_DoctorStaleness(t *testing.T) {
	tests := []struct {
		name       string
		lastDoctor time.Time
		wantRecent bool // if true, expect staleness < 1 week
	}{
		{
			name:       "never run",
			lastDoctor: time.Time{},
			wantRecent: false,
		},
		{
			name:       "run 1 hour ago",
			lastDoctor: time.Now().Add(-1 * time.Hour),
			wantRecent: true,
		},
		{
			name:       "run 1 day ago",
			lastDoctor: time.Now().Add(-24 * time.Hour),
			wantRecent: true,
		},
		{
			name:       "run 2 weeks ago",
			lastDoctor: time.Now().Add(-14 * 24 * time.Hour),
			wantRecent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Health{LastDoctorAt: tt.lastDoctor}
			staleness := h.DoctorStaleness()

			if tt.wantRecent {
				assert.Less(t, staleness, StalenessWeek)
			} else {
				assert.GreaterOrEqual(t, staleness, StalenessWeek)
			}
		})
	}
}

func TestStalenessToProb(t *testing.T) {
	tests := []struct {
		name      string
		staleness time.Duration
		wantProb  float64
	}{
		{"fresh", 0, ProbabilityNone},
		{"1 day", 24 * time.Hour, ProbabilityNone},
		{"6 days", 6 * 24 * time.Hour, ProbabilityNone},
		{"7 days", 7 * 24 * time.Hour, ProbabilityWeek1},
		{"10 days", 10 * 24 * time.Hour, ProbabilityWeek1},
		{"14 days", 14 * 24 * time.Hour, ProbabilityWeek2},
		{"21 days", 21 * 24 * time.Hour, ProbabilityWeek3},
		{"30 days", 30 * 24 * time.Hour, ProbabilityMonth},
		{"60 days", 60 * 24 * time.Hour, ProbabilityMonth},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prob := stalenessToProb(tt.staleness)
			assert.Equal(t, tt.wantProb, prob)
		})
	}
}

func TestHealth_DoctorHintMessage(t *testing.T) {
	tests := []struct {
		name       string
		staleness  time.Duration
		wantEmpty  bool
		wantSubstr string
	}{
		{
			name:      "fresh - no hint",
			staleness: 3 * 24 * time.Hour,
			wantEmpty: true,
		},
		{
			name:       "1 week",
			staleness:  8 * 24 * time.Hour,
			wantSubstr: "week",
		},
		{
			name:       "2 weeks",
			staleness:  15 * 24 * time.Hour,
			wantSubstr: "2 weeks",
		},
		{
			name:       "1 month",
			staleness:  35 * 24 * time.Hour,
			wantSubstr: "checkup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Health{
				LastDoctorAt: time.Now().Add(-tt.staleness),
			}
			msg := h.DoctorHintMessage()

			if tt.wantEmpty {
				assert.Empty(t, msg)
			} else {
				assert.Contains(t, msg, tt.wantSubstr)
			}
		})
	}
}

func TestHealth_DoctorFixHintMessage(t *testing.T) {
	h := &Health{
		LastDoctorFixAt: time.Now().Add(-15 * 24 * time.Hour),
	}
	msg := h.DoctorFixHintMessage()
	assert.Contains(t, msg, "ox doctor --fix")
}

func TestFormatWeeks(t *testing.T) {
	tests := []struct {
		weeks int
		want  string
	}{
		{1, "1 week"},
		{2, "2 weeks"},
		{4, "4 weeks"},
		{5, "about a month"},
		{8, "over 2 months"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatWeeks(tt.weeks)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoadHealth_CorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()
	sageoxPath := filepath.Join(tmpDir, sageoxDir)
	require.NoError(t, os.MkdirAll(sageoxPath, 0755))

	// write corrupted JSON
	healthPath := filepath.Join(sageoxPath, healthFilename)
	require.NoError(t, os.WriteFile(healthPath, []byte("not valid json"), 0600))

	// should return empty health, not error
	health, err := LoadHealth(tmpDir)
	require.NoError(t, err)
	assert.NotNil(t, health)
	assert.True(t, health.LastDoctorAt.IsZero())
}

func TestSaveHealth_NilInputs(t *testing.T) {
	// nil health should not error
	err := SaveHealth("", &Health{})
	assert.NoError(t, err)

	err = SaveHealth("/tmp", nil)
	assert.NoError(t, err)
}
