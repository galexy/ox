package doctor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createInitializedProject creates a temp directory with .sageox/ initialized.
func createInitializedProject(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	sageoxPath := filepath.Join(tmpDir, ".sageox")
	require.NoError(t, os.MkdirAll(sageoxPath, 0755), "failed to create .sageox/")
	return tmpDir
}

func TestNeedsDoctorHuman(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, gitRoot string)
		wantResult bool
	}{
		{
			name:       "returns false when .sageox does not exist",
			setup:      func(t *testing.T, gitRoot string) {},
			wantResult: false,
		},
		{
			name: "returns false when marker does not exist",
			setup: func(t *testing.T, gitRoot string) {
				require.NoError(t, os.MkdirAll(filepath.Join(gitRoot, ".sageox"), 0755))
			},
			wantResult: false,
		},
		{
			name: "returns true when marker exists",
			setup: func(t *testing.T, gitRoot string) {
				sageoxPath := filepath.Join(gitRoot, ".sageox")
				require.NoError(t, os.MkdirAll(sageoxPath, 0755))
				markerPath := filepath.Join(sageoxPath, NeedsDoctorMarker)
				f, err := os.Create(markerPath)
				require.NoError(t, err)
				require.NoError(t, f.Close())
			},
			wantResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setup(t, tmpDir)
			result := NeedsDoctorHuman(tmpDir)
			assert.Equal(t, tt.wantResult, result)
		})
	}
}

func TestNeedsDoctorAgent(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, gitRoot string)
		wantResult bool
	}{
		{
			name:       "returns false when .sageox does not exist",
			setup:      func(t *testing.T, gitRoot string) {},
			wantResult: false,
		},
		{
			name: "returns false when marker does not exist",
			setup: func(t *testing.T, gitRoot string) {
				require.NoError(t, os.MkdirAll(filepath.Join(gitRoot, ".sageox"), 0755))
			},
			wantResult: false,
		},
		{
			name: "returns true when marker exists",
			setup: func(t *testing.T, gitRoot string) {
				sageoxPath := filepath.Join(gitRoot, ".sageox")
				require.NoError(t, os.MkdirAll(sageoxPath, 0755))
				markerPath := filepath.Join(sageoxPath, NeedsDoctorAgentMarker)
				f, err := os.Create(markerPath)
				require.NoError(t, err)
				require.NoError(t, f.Close())
			},
			wantResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setup(t, tmpDir)
			result := NeedsDoctorAgent(tmpDir)
			assert.Equal(t, tt.wantResult, result)
		})
	}
}

func TestSetNeedsDoctorHuman(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, gitRoot string)
		wantError bool
	}{
		{
			name:      "returns error when .sageox does not exist",
			setup:     func(t *testing.T, gitRoot string) {},
			wantError: true,
		},
		{
			name: "creates marker when it does not exist",
			setup: func(t *testing.T, gitRoot string) {
				require.NoError(t, os.MkdirAll(filepath.Join(gitRoot, ".sageox"), 0755))
			},
			wantError: false,
		},
		{
			name: "updates mtime when marker already exists",
			setup: func(t *testing.T, gitRoot string) {
				sageoxPath := filepath.Join(gitRoot, ".sageox")
				require.NoError(t, os.MkdirAll(sageoxPath, 0755))
				markerPath := filepath.Join(sageoxPath, NeedsDoctorMarker)
				f, err := os.Create(markerPath)
				require.NoError(t, err)
				require.NoError(t, f.Close())
				// set old mtime
				oldTime := time.Now().Add(-24 * time.Hour)
				require.NoError(t, os.Chtimes(markerPath, oldTime, oldTime))
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setup(t, tmpDir)

			err := SetNeedsDoctorHuman(tmpDir)

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			// verify marker exists
			markerPath := filepath.Join(tmpDir, ".sageox", NeedsDoctorMarker)
			info, err := os.Stat(markerPath)
			require.NoError(t, err)
			assert.True(t, info.Size() == 0, "marker should be empty file")
		})
	}
}

func TestSetNeedsDoctorAgent(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, gitRoot string)
		wantError bool
	}{
		{
			name:      "returns error when .sageox does not exist",
			setup:     func(t *testing.T, gitRoot string) {},
			wantError: true,
		},
		{
			name: "creates marker when it does not exist",
			setup: func(t *testing.T, gitRoot string) {
				require.NoError(t, os.MkdirAll(filepath.Join(gitRoot, ".sageox"), 0755))
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setup(t, tmpDir)

			err := SetNeedsDoctorAgent(tmpDir)

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			// verify marker exists
			markerPath := filepath.Join(tmpDir, ".sageox", NeedsDoctorAgentMarker)
			info, err := os.Stat(markerPath)
			require.NoError(t, err)
			assert.True(t, info.Size() == 0, "marker should be empty file")
		})
	}
}

func TestClearNeedsDoctorHuman(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, gitRoot string)
		wantError bool
	}{
		{
			name:      "no error when .sageox does not exist",
			setup:     func(t *testing.T, gitRoot string) {},
			wantError: false,
		},
		{
			name: "no error when marker does not exist",
			setup: func(t *testing.T, gitRoot string) {
				require.NoError(t, os.MkdirAll(filepath.Join(gitRoot, ".sageox"), 0755))
			},
			wantError: false,
		},
		{
			name: "removes marker when it exists",
			setup: func(t *testing.T, gitRoot string) {
				sageoxPath := filepath.Join(gitRoot, ".sageox")
				require.NoError(t, os.MkdirAll(sageoxPath, 0755))
				markerPath := filepath.Join(sageoxPath, NeedsDoctorMarker)
				f, err := os.Create(markerPath)
				require.NoError(t, err)
				require.NoError(t, f.Close())
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setup(t, tmpDir)

			err := ClearNeedsDoctorHuman(tmpDir)

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			// verify marker does not exist
			markerPath := filepath.Join(tmpDir, ".sageox", NeedsDoctorMarker)
			_, err = os.Stat(markerPath)
			assert.True(t, os.IsNotExist(err), "marker should not exist")
		})
	}
}

func TestClearNeedsDoctorAgent(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, gitRoot string)
		wantError bool
	}{
		{
			name:      "no error when .sageox does not exist",
			setup:     func(t *testing.T, gitRoot string) {},
			wantError: false,
		},
		{
			name: "no error when marker does not exist",
			setup: func(t *testing.T, gitRoot string) {
				require.NoError(t, os.MkdirAll(filepath.Join(gitRoot, ".sageox"), 0755))
			},
			wantError: false,
		},
		{
			name: "removes marker when it exists",
			setup: func(t *testing.T, gitRoot string) {
				sageoxPath := filepath.Join(gitRoot, ".sageox")
				require.NoError(t, os.MkdirAll(sageoxPath, 0755))
				markerPath := filepath.Join(sageoxPath, NeedsDoctorAgentMarker)
				f, err := os.Create(markerPath)
				require.NoError(t, err)
				require.NoError(t, f.Close())
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setup(t, tmpDir)

			err := ClearNeedsDoctorAgent(tmpDir)

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			// verify marker does not exist
			markerPath := filepath.Join(tmpDir, ".sageox", NeedsDoctorAgentMarker)
			_, err = os.Stat(markerPath)
			assert.True(t, os.IsNotExist(err), "marker should not exist")
		})
	}
}

func TestGetDoctorNeeds(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, gitRoot string)
		wantHuman bool
		wantAgent bool
	}{
		{
			name:      "both false when .sageox does not exist",
			setup:     func(t *testing.T, gitRoot string) {},
			wantHuman: false,
			wantAgent: false,
		},
		{
			name: "both false when no markers exist",
			setup: func(t *testing.T, gitRoot string) {
				require.NoError(t, os.MkdirAll(filepath.Join(gitRoot, ".sageox"), 0755))
			},
			wantHuman: false,
			wantAgent: false,
		},
		{
			name: "only human true when only human marker exists",
			setup: func(t *testing.T, gitRoot string) {
				sageoxPath := filepath.Join(gitRoot, ".sageox")
				require.NoError(t, os.MkdirAll(sageoxPath, 0755))
				f, err := os.Create(filepath.Join(sageoxPath, NeedsDoctorMarker))
				require.NoError(t, err)
				require.NoError(t, f.Close())
			},
			wantHuman: true,
			wantAgent: false,
		},
		{
			name: "only agent true when only agent marker exists",
			setup: func(t *testing.T, gitRoot string) {
				sageoxPath := filepath.Join(gitRoot, ".sageox")
				require.NoError(t, os.MkdirAll(sageoxPath, 0755))
				f, err := os.Create(filepath.Join(sageoxPath, NeedsDoctorAgentMarker))
				require.NoError(t, err)
				require.NoError(t, f.Close())
			},
			wantHuman: false,
			wantAgent: true,
		},
		{
			name: "both true when both markers exist",
			setup: func(t *testing.T, gitRoot string) {
				sageoxPath := filepath.Join(gitRoot, ".sageox")
				require.NoError(t, os.MkdirAll(sageoxPath, 0755))
				f1, err := os.Create(filepath.Join(sageoxPath, NeedsDoctorMarker))
				require.NoError(t, err)
				require.NoError(t, f1.Close())
				f2, err := os.Create(filepath.Join(sageoxPath, NeedsDoctorAgentMarker))
				require.NoError(t, err)
				require.NoError(t, f2.Close())
			},
			wantHuman: true,
			wantAgent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setup(t, tmpDir)

			human, agent := GetDoctorNeeds(tmpDir)

			assert.Equal(t, tt.wantHuman, human, "human marker mismatch")
			assert.Equal(t, tt.wantAgent, agent, "agent marker mismatch")
		})
	}
}

func TestTouchUpdatesModTime(t *testing.T) {
	tmpDir := createInitializedProject(t)
	markerPath := filepath.Join(tmpDir, ".sageox", NeedsDoctorMarker)

	// create marker with old mtime
	f, err := os.Create(markerPath)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	oldTime := time.Now().Add(-24 * time.Hour)
	require.NoError(t, os.Chtimes(markerPath, oldTime, oldTime))

	// verify old mtime
	info1, err := os.Stat(markerPath)
	require.NoError(t, err)
	assert.True(t, info1.ModTime().Before(time.Now().Add(-23*time.Hour)), "mtime should be old")

	// touch the marker
	require.NoError(t, SetNeedsDoctorHuman(tmpDir))

	// verify mtime is updated
	info2, err := os.Stat(markerPath)
	require.NoError(t, err)
	assert.True(t, info2.ModTime().After(time.Now().Add(-1*time.Minute)), "mtime should be recent")
}

func TestIdempotentOperations(t *testing.T) {
	tmpDir := createInitializedProject(t)

	// set multiple times should not error
	require.NoError(t, SetNeedsDoctorHuman(tmpDir))
	require.NoError(t, SetNeedsDoctorHuman(tmpDir))
	require.NoError(t, SetNeedsDoctorHuman(tmpDir))
	assert.True(t, NeedsDoctorHuman(tmpDir))

	// clear multiple times should not error
	require.NoError(t, ClearNeedsDoctorHuman(tmpDir))
	require.NoError(t, ClearNeedsDoctorHuman(tmpDir))
	require.NoError(t, ClearNeedsDoctorHuman(tmpDir))
	assert.False(t, NeedsDoctorHuman(tmpDir))
}
