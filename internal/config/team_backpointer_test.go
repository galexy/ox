package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterWorkspace(t *testing.T) {
	t.Run("registers new workspace", func(t *testing.T) {
		teamDir := t.TempDir()

		err := RegisterWorkspace(teamDir, "ws-123", "/path/to/project")
		require.NoError(t, err)

		backpointers, err := LoadBackpointers(teamDir)
		require.NoError(t, err)
		require.Len(t, backpointers, 1)

		assert.Equal(t, "ws-123", backpointers[0].WorkspaceID)
		assert.Equal(t, "/path/to/project", backpointers[0].ProjectPath)
		assert.False(t, backpointers[0].LastActive.IsZero())
	})

	t.Run("updates existing workspace", func(t *testing.T) {
		teamDir := t.TempDir()

		// register first time
		err := RegisterWorkspace(teamDir, "ws-123", "/old/path")
		require.NoError(t, err)

		// register again with new path
		err = RegisterWorkspace(teamDir, "ws-123", "/new/path")
		require.NoError(t, err)

		backpointers, err := LoadBackpointers(teamDir)
		require.NoError(t, err)
		require.Len(t, backpointers, 1, "should not duplicate")

		assert.Equal(t, "/new/path", backpointers[0].ProjectPath)
	})

	t.Run("registers multiple workspaces", func(t *testing.T) {
		teamDir := t.TempDir()

		err := RegisterWorkspace(teamDir, "ws-1", "/path/1")
		require.NoError(t, err)
		err = RegisterWorkspace(teamDir, "ws-2", "/path/2")
		require.NoError(t, err)
		err = RegisterWorkspace(teamDir, "ws-3", "/path/3")
		require.NoError(t, err)

		backpointers, err := LoadBackpointers(teamDir)
		require.NoError(t, err)
		assert.Len(t, backpointers, 3)
	})

	t.Run("returns error for empty path", func(t *testing.T) {
		err := RegisterWorkspace("", "ws-123", "/path")
		assert.Error(t, err)
	})

	t.Run("returns error for empty workspace ID", func(t *testing.T) {
		err := RegisterWorkspace("/tmp/team", "", "/path")
		assert.Error(t, err)
	})
}

func TestLoadBackpointers(t *testing.T) {
	t.Run("returns nil for non-existent file", func(t *testing.T) {
		teamDir := t.TempDir()

		backpointers, err := LoadBackpointers(teamDir)
		require.NoError(t, err)
		assert.Nil(t, backpointers)
	})

	t.Run("loads existing backpointers", func(t *testing.T) {
		teamDir := t.TempDir()

		// create backpointer file manually
		sageoxDir := filepath.Join(teamDir, ".sageox")
		require.NoError(t, os.MkdirAll(sageoxDir, 0755))

		content := `{"workspace_id":"ws-1","project_path":"/path/1","last_active":"2025-01-01T10:00:00Z"}
{"workspace_id":"ws-2","project_path":"/path/2","last_active":"2025-01-02T10:00:00Z"}
`
		require.NoError(t, os.WriteFile(
			filepath.Join(sageoxDir, "workspaces.jsonl"),
			[]byte(content),
			0600,
		))

		backpointers, err := LoadBackpointers(teamDir)
		require.NoError(t, err)
		require.Len(t, backpointers, 2)

		assert.Equal(t, "ws-1", backpointers[0].WorkspaceID)
		assert.Equal(t, "ws-2", backpointers[1].WorkspaceID)
	})

	t.Run("skips malformed lines", func(t *testing.T) {
		teamDir := t.TempDir()

		sageoxDir := filepath.Join(teamDir, ".sageox")
		require.NoError(t, os.MkdirAll(sageoxDir, 0755))

		content := `{"workspace_id":"ws-1","project_path":"/path/1","last_active":"2025-01-01T10:00:00Z"}
invalid json line
{"workspace_id":"ws-2","project_path":"/path/2","last_active":"2025-01-02T10:00:00Z"}
`
		require.NoError(t, os.WriteFile(
			filepath.Join(sageoxDir, "workspaces.jsonl"),
			[]byte(content),
			0600,
		))

		backpointers, err := LoadBackpointers(teamDir)
		require.NoError(t, err)
		assert.Len(t, backpointers, 2, "should skip malformed line")
	})

	t.Run("returns error for empty path", func(t *testing.T) {
		_, err := LoadBackpointers("")
		assert.Error(t, err)
	})
}

func TestUpdateWorkspaceActivity(t *testing.T) {
	t.Run("updates last active time", func(t *testing.T) {
		teamDir := t.TempDir()

		// register workspace with old time
		err := RegisterWorkspace(teamDir, "ws-123", "/path")
		require.NoError(t, err)

		// manually set an old time
		backpointers, _ := LoadBackpointers(teamDir)
		backpointers[0].LastActive = time.Now().Add(-24 * time.Hour)
		_ = SaveBackpointers(teamDir, backpointers)

		// update activity
		before := time.Now()
		err = UpdateWorkspaceActivity(teamDir, "ws-123")
		require.NoError(t, err)

		// verify time was updated
		backpointers, _ = LoadBackpointers(teamDir)
		assert.True(t, backpointers[0].LastActive.After(before) || backpointers[0].LastActive.Equal(before))
	})

	t.Run("no-op for unknown workspace", func(t *testing.T) {
		teamDir := t.TempDir()

		err := RegisterWorkspace(teamDir, "ws-123", "/path")
		require.NoError(t, err)

		// try to update unknown workspace
		err = UpdateWorkspaceActivity(teamDir, "ws-unknown")
		assert.NoError(t, err, "should not error for unknown workspace")
	})

	t.Run("no-op for empty inputs", func(t *testing.T) {
		assert.NoError(t, UpdateWorkspaceActivity("", "ws-123"))
		assert.NoError(t, UpdateWorkspaceActivity("/tmp", ""))
	})
}

func TestAnalyzeTeamContextHealth(t *testing.T) {
	t.Run("non-existent path", func(t *testing.T) {
		health := AnalyzeTeamContextHealth("team-1", "/non/existent/path", time.Time{})

		assert.False(t, health.Exists)
		assert.False(t, health.IsGitRepo)
	})

	t.Run("existing directory not git repo", func(t *testing.T) {
		teamDir := t.TempDir()

		health := AnalyzeTeamContextHealth("team-1", teamDir, time.Time{})

		assert.True(t, health.Exists)
		assert.False(t, health.IsGitRepo)
	})

	t.Run("git repo with no backpointers", func(t *testing.T) {
		teamDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(teamDir, ".git"), 0755))

		health := AnalyzeTeamContextHealth("team-1", teamDir, time.Now())

		assert.True(t, health.Exists)
		assert.True(t, health.IsGitRepo)
		assert.Empty(t, health.Workspaces)
		assert.Equal(t, 0, health.ActiveCount)
	})

	t.Run("counts active and stale workspaces", func(t *testing.T) {
		teamDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(teamDir, ".git"), 0755))

		// create temp project dirs
		activeProject := t.TempDir()
		staleProject := t.TempDir()

		// register workspaces with different activity times
		backpointers := []WorkspaceBackpointer{
			{WorkspaceID: "ws-active", ProjectPath: activeProject, LastActive: time.Now()},
			{WorkspaceID: "ws-stale", ProjectPath: staleProject, LastActive: time.Now().Add(-60 * 24 * time.Hour)},
			{WorkspaceID: "ws-orphan", ProjectPath: "/non/existent/project", LastActive: time.Now()},
		}
		require.NoError(t, SaveBackpointers(teamDir, backpointers))

		health := AnalyzeTeamContextHealth("team-1", teamDir, time.Now())

		assert.Equal(t, 1, health.ActiveCount)
		assert.Equal(t, 1, health.StaleCount)
		assert.Equal(t, 1, health.OrphanedCount)
		assert.Len(t, health.Workspaces, 3)
	})

	t.Run("detects orphaned team context", func(t *testing.T) {
		teamDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(teamDir, ".git"), 0755))

		// all workspaces point to non-existent paths
		backpointers := []WorkspaceBackpointer{
			{WorkspaceID: "ws-1", ProjectPath: "/gone/1", LastActive: time.Now()},
			{WorkspaceID: "ws-2", ProjectPath: "/gone/2", LastActive: time.Now()},
		}
		require.NoError(t, SaveBackpointers(teamDir, backpointers))

		health := AnalyzeTeamContextHealth("team-1", teamDir, time.Now())

		assert.True(t, health.IsOrphaned)
		assert.Equal(t, 2, health.OrphanedCount)
	})

	t.Run("detects stale team context", func(t *testing.T) {
		teamDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(teamDir, ".git"), 0755))

		staleProject := t.TempDir()

		// workspace with old activity
		backpointers := []WorkspaceBackpointer{
			{WorkspaceID: "ws-1", ProjectPath: staleProject, LastActive: time.Now().Add(-60 * 24 * time.Hour)},
		}
		require.NoError(t, SaveBackpointers(teamDir, backpointers))

		// last sync was 60 days ago
		lastSync := time.Now().Add(-60 * 24 * time.Hour)
		health := AnalyzeTeamContextHealth("team-1", teamDir, lastSync)

		assert.True(t, health.IsStale)
		assert.Equal(t, 0, health.ActiveCount)
	})
}

func TestDiscoverTeamContexts(t *testing.T) {
	t.Run("finds team contexts in centralized location", func(t *testing.T) {
		tempHome := t.TempDir()
		// use XDG mode to avoid cached HOME from previous tests
		t.Setenv("OX_XDG_ENABLE", "1")
		t.Setenv("HOME", tempHome)
		t.Setenv("XDG_DATA_HOME", filepath.Join(tempHome, ".local", "share"))
		t.Setenv("SAGEOX_ENDPOINT", "") // ensure default production endpoint

		// create team directories with .git in XDG data location
		// all endpoints now use namespaced structure: .../sageox/<endpoint>/teams/
		teamsDir := filepath.Join(tempHome, ".local", "share", "sageox", "sageox.ai", "teams")
		require.NoError(t, os.MkdirAll(filepath.Join(teamsDir, "team-1", ".git"), 0755))
		require.NoError(t, os.MkdirAll(filepath.Join(teamsDir, "team-2", ".git"), 0755))
		// create a non-git directory (should be ignored)
		require.NoError(t, os.MkdirAll(filepath.Join(teamsDir, "not-a-repo"), 0755))

		teams, err := DiscoverTeamContexts()
		require.NoError(t, err)
		// deduplicate results for assertion (DiscoverTeamContexts may return duplicates
		// when production and endpoint paths overlap with the new unified structure)
		uniqueTeams := make(map[string]bool)
		for _, team := range teams {
			uniqueTeams[team] = true
		}
		assert.Len(t, uniqueTeams, 2, "should find 2 unique team contexts")
	})

	t.Run("returns nil for non-existent teams dir", func(t *testing.T) {
		tempHome := t.TempDir()
		// use XDG mode to avoid cached HOME from previous tests
		t.Setenv("OX_XDG_ENABLE", "1")
		t.Setenv("HOME", tempHome)
		t.Setenv("XDG_DATA_HOME", filepath.Join(tempHome, ".local", "share"))

		teams, err := DiscoverTeamContexts()
		require.NoError(t, err)
		assert.Nil(t, teams)
	})
}

func TestDiscoverLegacyTeamContexts(t *testing.T) {
	t.Run("finds legacy team contexts", func(t *testing.T) {
		parentDir := t.TempDir()
		projectDir := filepath.Join(parentDir, "my-project")
		require.NoError(t, os.MkdirAll(projectDir, 0755))

		// create legacy team context directories
		legacy1 := filepath.Join(parentDir, "sageox_team_abc123_context")
		legacy2 := filepath.Join(parentDir, "sageox_team_def456_context")
		require.NoError(t, os.MkdirAll(filepath.Join(legacy1, ".git"), 0755))
		require.NoError(t, os.MkdirAll(filepath.Join(legacy2, ".git"), 0755))

		// create a non-matching directory
		require.NoError(t, os.MkdirAll(filepath.Join(parentDir, "other_dir"), 0755))

		legacyDirs, err := DiscoverLegacyTeamContexts(projectDir)
		require.NoError(t, err)
		assert.Len(t, legacyDirs, 2)
	})

	t.Run("ignores non-git directories", func(t *testing.T) {
		parentDir := t.TempDir()
		projectDir := filepath.Join(parentDir, "my-project")
		require.NoError(t, os.MkdirAll(projectDir, 0755))

		// create legacy directory without .git
		legacy := filepath.Join(parentDir, "sageox_team_abc123_context")
		require.NoError(t, os.MkdirAll(legacy, 0755))

		legacyDirs, err := DiscoverLegacyTeamContexts(projectDir)
		require.NoError(t, err)
		assert.Empty(t, legacyDirs)
	})

	t.Run("returns nil for empty project root", func(t *testing.T) {
		legacyDirs, err := DiscoverLegacyTeamContexts("")
		require.NoError(t, err)
		assert.Nil(t, legacyDirs)
	})
}

func TestCleanupOrphanedBackpointers(t *testing.T) {
	t.Run("removes orphaned backpointers", func(t *testing.T) {
		teamDir := t.TempDir()
		existingProject := t.TempDir()

		// register workspaces - one exists, one doesn't
		backpointers := []WorkspaceBackpointer{
			{WorkspaceID: "ws-exists", ProjectPath: existingProject, LastActive: time.Now()},
			{WorkspaceID: "ws-gone", ProjectPath: "/non/existent/path", LastActive: time.Now()},
		}
		require.NoError(t, SaveBackpointers(teamDir, backpointers))

		removed, err := CleanupOrphanedBackpointers(teamDir)
		require.NoError(t, err)
		assert.Equal(t, 1, removed)

		// verify only valid backpointer remains
		remaining, _ := LoadBackpointers(teamDir)
		assert.Len(t, remaining, 1)
		assert.Equal(t, "ws-exists", remaining[0].WorkspaceID)
	})

	t.Run("no-op when no orphans", func(t *testing.T) {
		teamDir := t.TempDir()
		existingProject := t.TempDir()

		backpointers := []WorkspaceBackpointer{
			{WorkspaceID: "ws-1", ProjectPath: existingProject, LastActive: time.Now()},
		}
		require.NoError(t, SaveBackpointers(teamDir, backpointers))

		removed, err := CleanupOrphanedBackpointers(teamDir)
		require.NoError(t, err)
		assert.Equal(t, 0, removed)
	})
}
