package paths

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/sageox/ox/internal/endpoint"
)

var (
	migrationOnce sync.Once
	migrationErr  error
)

// LegacyPaths contains the old path locations for migration detection.
type LegacyPaths struct {
	// ConfigDir is the old XDG config location (~/.config/sageox)
	ConfigDir string
	// GuidanceCache is the old hardcoded guidance cache (~/.sageox/guidance/cache)
	GuidanceCache string
	// SessionCache is the old XDG cache location (~/.cache/sageox)
	SessionCache string
}

// GetLegacyPaths returns the legacy path locations for migration checks.
func GetLegacyPaths() LegacyPaths {
	home := getHomeDir()
	return LegacyPaths{
		ConfigDir:     filepath.Join(home, ".config", "sageox"),
		GuidanceCache: filepath.Join(home, ".sageox", "guidance", "cache"),
		SessionCache:  filepath.Join(home, ".cache", "sageox"),
	}
}

// MigrationStatus represents the result of checking migration state.
type MigrationStatus struct {
	// Needed is true if migration should be performed
	Needed bool
	// LegacyConfigExists is true if ~/.config/sageox exists
	LegacyConfigExists bool
	// LegacyGuidanceCacheExists is true if ~/.sageox/guidance/cache exists
	LegacyGuidanceCacheExists bool
	// LegacySessionCacheExists is true if ~/.cache/sageox exists
	LegacySessionCacheExists bool
	// NewStructureExists is true if ~/.sageox/config exists (new structure)
	NewStructureExists bool
}

// CheckMigrationStatus checks whether migration from legacy paths is needed.
// This does NOT perform migration, only checks the current state.
func CheckMigrationStatus() MigrationStatus {
	// in XDG mode, no migration needed - user wants XDG paths
	if useXDGMode() {
		return MigrationStatus{Needed: false}
	}

	legacy := GetLegacyPaths()
	status := MigrationStatus{}

	// check if new structure already exists
	newConfigDir := ConfigDir()
	if _, err := os.Stat(newConfigDir); err == nil {
		status.NewStructureExists = true
	}

	// check legacy locations
	if _, err := os.Stat(legacy.ConfigDir); err == nil {
		status.LegacyConfigExists = true
	}
	if _, err := os.Stat(legacy.GuidanceCache); err == nil {
		status.LegacyGuidanceCacheExists = true
	}
	if _, err := os.Stat(legacy.SessionCache); err == nil {
		status.LegacySessionCacheExists = true
	}

	// migration needed if:
	// 1. New structure doesn't exist yet, AND
	// 2. At least one legacy location exists
	status.Needed = !status.NewStructureExists &&
		(status.LegacyConfigExists || status.LegacyGuidanceCacheExists || status.LegacySessionCacheExists)

	return status
}

// MigrationResult contains the outcome of a migration attempt.
type MigrationResult struct {
	// ConfigMigrated is true if config was migrated
	ConfigMigrated bool
	// GuidanceCacheMigrated is true if guidance cache was migrated
	GuidanceCacheMigrated bool
	// SessionCacheMigrated is true if session cache was migrated
	SessionCacheMigrated bool
	// Errors contains any errors encountered (migration continues on error)
	Errors []error
}

// Migrate performs migration from legacy paths to the new consolidated structure.
// This is safe to call multiple times - it will only migrate once.
// Migration is skipped if OX_XDG_ENABLE is set.
//
// Migration moves:
//   - ~/.config/sageox/* → ~/.sageox/config/
//   - ~/.sageox/guidance/cache/* → ~/.sageox/cache/guidance/
//   - ~/.cache/sageox/* → ~/.sageox/cache/
//
// Note: Team contexts in sibling directories are NOT automatically migrated.
// Use TeamContextMigrationNeeded() and MigrateTeamContext() for those.
func Migrate() MigrationResult {
	result := MigrationResult{}

	// skip in XDG mode
	if useXDGMode() {
		return result
	}

	status := CheckMigrationStatus()
	if !status.Needed {
		return result
	}

	legacy := GetLegacyPaths()

	// migrate config
	if status.LegacyConfigExists {
		if err := migrateDirectory(legacy.ConfigDir, ConfigDir()); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("config migration: %w", err))
		} else {
			result.ConfigMigrated = true
		}
	}

	// migrate guidance cache
	if status.LegacyGuidanceCacheExists {
		if err := migrateDirectory(legacy.GuidanceCache, GuidanceCacheDir()); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("guidance cache migration: %w", err))
		} else {
			result.GuidanceCacheMigrated = true
		}
	}

	// migrate session cache
	if status.LegacySessionCacheExists {
		// sessions go to cache/sessions, but legacy was flat in ~/.cache/sageox
		// we need to be careful not to overwrite other things in the cache dir
		if err := migrateSessionCache(legacy.SessionCache); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("session cache migration: %w", err))
		} else {
			result.SessionCacheMigrated = true
		}
	}

	return result
}

// EnsureMigrated runs migration once if needed.
// This is intended to be called early in application startup.
// Returns any error from the migration attempt.
func EnsureMigrated() error {
	migrationOnce.Do(func() {
		result := Migrate()
		if len(result.Errors) > 0 {
			migrationErr = result.Errors[0]
		}
	})
	return migrationErr
}

// migrateDirectory copies contents from src to dst, then removes src.
// Creates dst if it doesn't exist. Preserves file permissions.
func migrateDirectory(src, dst string) error {
	// ensure destination exists
	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("create destination: %w", err)
	}

	// read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}

	// copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		// skip if destination already exists
		if _, err := os.Stat(dstPath); err == nil {
			continue
		}

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return fmt.Errorf("copy directory %s: %w", entry.Name(), err)
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return fmt.Errorf("copy file %s: %w", entry.Name(), err)
			}
		}
	}

	// remove source directory after successful migration
	// use RemoveAll to handle any remaining files
	if err := os.RemoveAll(src); err != nil {
		// log but don't fail - files are already copied
		return nil
	}

	return nil
}

// migrateSessionCache handles the special case of session cache migration.
// The legacy structure had sessions directly in ~/.cache/sageox/context/
// We need to move just the relevant subdirectories.
func migrateSessionCache(legacyDir string) error {
	// look for context/ subdirectory which contains session data
	contextDir := filepath.Join(legacyDir, "context")
	if _, err := os.Stat(contextDir); os.IsNotExist(err) {
		// no context dir, nothing to migrate
		return nil
	}

	// migrate context/* to cache/sessions/
	dstDir := SessionCacheDir("")
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("create sessions dir: %w", err)
	}

	entries, err := os.ReadDir(contextDir)
	if err != nil {
		return fmt.Errorf("read context dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		srcPath := filepath.Join(contextDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())

		if _, err := os.Stat(dstPath); err == nil {
			continue // skip if exists
		}

		if err := copyDir(srcPath, dstPath); err != nil {
			return fmt.Errorf("copy %s: %w", entry.Name(), err)
		}
	}

	// clean up legacy context directory
	_ = os.RemoveAll(contextDir)

	// also migrate daemon logs if present
	daemonLogsPattern := filepath.Join(legacyDir, "daemon-*.log")
	matches, _ := filepath.Glob(daemonLogsPattern)
	if len(matches) > 0 {
		dstLogDir := filepath.Join(TempDir(), "logs")
		_ = os.MkdirAll(dstLogDir, 0755)
		for _, src := range matches {
			dst := filepath.Join(dstLogDir, filepath.Base(src))
			_ = copyFile(src, dst)
			_ = os.Remove(src)
		}
	}

	return nil
}

// copyFile copies a single file preserving permissions.
func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, info.Mode())
}

// copyDir recursively copies a directory.
func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// TeamContextLegacyPath represents a team context in the old sibling directory format.
type TeamContextLegacyPath struct {
	TeamID      string
	LegacyPath  string // e.g., /Users/foo/Code/sageox_team_abc123_context
	ProjectPath string // the project that referenced it
}

// TeamContextMigrationNeeded checks if there are team contexts in sibling directories
// that should be migrated to ~/.sageox/data/teams/
// Note: Legacy paths are always migrated to production endpoint location.
func TeamContextMigrationNeeded(projectRoot, teamID, legacyPath string) bool {
	if useXDGMode() {
		return false
	}

	// check if legacy path exists
	if _, err := os.Stat(legacyPath); os.IsNotExist(err) {
		return false
	}

	// check if new location already exists (using production endpoint path)
	newPath := TeamContextDir(teamID, endpoint.Production)
	if _, err := os.Stat(newPath); err == nil {
		return false // already migrated
	}

	return true
}

// MigrateTeamContext moves a team context from sibling directory to ~/.sageox/data/teams/
// Note: Legacy paths are always migrated to production endpoint location.
func MigrateTeamContext(teamID, legacyPath string) error {
	if useXDGMode() {
		return fmt.Errorf("cannot migrate team context in XDG mode")
	}

	// use production endpoint path for migration target
	newPath := TeamContextDir(teamID, endpoint.Production)

	// ensure parent directory exists
	if err := os.MkdirAll(TeamsDataDir(endpoint.Production), 0755); err != nil {
		return fmt.Errorf("create teams directory: %w", err)
	}

	// move the directory
	if err := os.Rename(legacyPath, newPath); err != nil {
		// if rename fails (cross-device), fall back to copy
		if err := copyDir(legacyPath, newPath); err != nil {
			return fmt.Errorf("copy team context: %w", err)
		}
		// best effort cleanup of legacy path - ignore errors
		_ = os.RemoveAll(legacyPath)
	}

	return nil
}

// -----------------------------------------------------------------------------
// Ledger Migration: repo_sageox_ledger/ → repo_sageox/<endpoint>/ledger/
// -----------------------------------------------------------------------------

// LedgerMigrationStatus represents the result of checking ledger migration state.
type LedgerMigrationStatus struct {
	// NeedsMigration is true if old structure exists and new doesn't
	NeedsMigration bool
	// LegacyPath is the old ledger path (repo_sageox_ledger/)
	LegacyPath string
	// NewPath is the new ledger path (repo_sageox/<endpoint>/ledger/)
	NewPath string
	// LegacyExists is true if the legacy path exists
	LegacyExists bool
	// NewExists is true if the new path already exists
	NewExists bool
}

// DetectLegacyLedgerPath checks for the old repo_sageox_ledger/ sibling directory.
// Returns the path if it exists, empty string if not.
//
// The legacy format is: <project_parent>/<repo_name>_sageox_ledger/
// For example: /Users/dev/Code/myrepo_sageox_ledger/
func DetectLegacyLedgerPath(projectRoot string) string {
	if projectRoot == "" {
		return ""
	}

	parentDir := filepath.Dir(projectRoot)
	repoName := filepath.Base(projectRoot)
	safeName := sanitizePathComponent(repoName)
	legacyPath := filepath.Join(parentDir, safeName+"_sageox_ledger")

	if info, err := os.Stat(legacyPath); err == nil && info.IsDir() {
		return legacyPath
	}

	return ""
}

// LedgerNeedsMigration checks if ledger migration from the old structure to the new
// endpoint-namespaced structure is needed.
//
// Returns true if:
// - Old structure exists (repo_sageox_ledger/)
// - New structure doesn't exist (repo_sageox/<endpoint>/ledger/)
//
// The endpoint parameter determines the target namespace. If empty, uses current endpoint.
func LedgerNeedsMigration(projectRoot, ep string) bool {
	if projectRoot == "" {
		return false
	}

	legacyPath := DetectLegacyLedgerPath(projectRoot)
	if legacyPath == "" {
		return false // no legacy structure to migrate
	}

	newPath := NewLedgerPath(projectRoot, ep)
	if _, err := os.Stat(newPath); err == nil {
		return false // new structure already exists
	}

	return true
}

// CheckLedgerMigrationStatus returns detailed status about ledger migration.
// This is useful for displaying migration status to users.
func CheckLedgerMigrationStatus(projectRoot, ep string) LedgerMigrationStatus {
	status := LedgerMigrationStatus{}

	if projectRoot == "" {
		return status
	}

	status.LegacyPath = DetectLegacyLedgerPath(projectRoot)
	status.LegacyExists = status.LegacyPath != ""

	// if legacy doesn't exist, construct what it would be for reporting
	if status.LegacyPath == "" {
		parentDir := filepath.Dir(projectRoot)
		repoName := filepath.Base(projectRoot)
		safeName := sanitizePathComponent(repoName)
		status.LegacyPath = filepath.Join(parentDir, safeName+"_sageox_ledger")
	}

	status.NewPath = NewLedgerPath(projectRoot, ep)
	if _, err := os.Stat(status.NewPath); err == nil {
		status.NewExists = true
	}

	status.NeedsMigration = status.LegacyExists && !status.NewExists

	return status
}

// NewLedgerPath returns the new endpoint-namespaced ledger path.
//
// Format: <project_parent>/<repo_name>_sageox/<endpoint>/ledger/
// For production endpoints: <project_parent>/<repo_name>_sageox/ledger/
//
// Examples:
//   - Production: /Users/dev/Code/myrepo_sageox/ledger/
//   - Staging: /Users/dev/Code/myrepo_sageox/staging.sageox.ai/ledger/
//   - Localhost: /Users/dev/Code/myrepo_sageox/localhost/ledger/
//
// IMPORTANT: ep is REQUIRED. Use endpoint.GetForProject(projectRoot) to get the
// correct endpoint for a project context.
func NewLedgerPath(projectRoot, ep string) string {
	if projectRoot == "" {
		return ""
	}

	if ep == "" {
		panic("NewLedgerPath: endpoint is required - use endpoint.GetForProject(projectRoot) not empty string")
	}

	parentDir := filepath.Dir(projectRoot)
	repoName := filepath.Base(projectRoot)
	safeName := sanitizePathComponent(repoName)
	basePath := filepath.Join(parentDir, safeName+"_sageox")

	// for production endpoints, skip the endpoint directory level
	if endpoint.IsProduction(ep) {
		return filepath.Join(basePath, "ledger")
	}

	// for non-production, include endpoint in path
	safeEndpoint := endpoint.SanitizeForPath(ep)
	return filepath.Join(basePath, safeEndpoint, "ledger")
}

// MigrateLedgerToNewStructure migrates from the old repo_sageox_ledger/
// sibling directory to the new repo_sageox/<endpoint>/ledger/ structure.
//
// Steps:
//  1. Verify old path exists
//  2. Create new directory structure (repo_sageox/<endpoint>/)
//  3. Move ledger contents (not rename, to preserve git)
//  4. Create team symlinks (placeholder for future implementation)
//  5. Remove old directory
//
// Returns error if migration fails, nil on success or if no migration needed.
//
// The migration is designed to be safe and reversible:
//   - Files are copied before the original is removed
//   - If copy fails partway through, original remains intact
//   - The old directory is only removed after successful copy
func MigrateLedgerToNewStructure(projectRoot, ep string) error {
	if projectRoot == "" {
		return fmt.Errorf("project root cannot be empty")
	}

	legacyPath := DetectLegacyLedgerPath(projectRoot)
	if legacyPath == "" {
		// no legacy structure to migrate
		slog.Debug("no legacy ledger found", "project_root", projectRoot)
		return nil
	}

	newPath := NewLedgerPath(projectRoot, ep)

	// check if new path already exists
	if _, err := os.Stat(newPath); err == nil {
		slog.Debug("new ledger structure already exists", "path", newPath)
		return nil // already migrated
	}

	slog.Info("migrating ledger to new structure", "from", legacyPath, "to", newPath)

	// ensure parent directory exists (repo_sageox/<endpoint>/)
	parentDir := filepath.Dir(newPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}

	// try rename first (fast path, same filesystem)
	if err := os.Rename(legacyPath, newPath); err == nil {
		slog.Info("ledger migration complete (renamed)", "new_path", newPath)
		return nil
	}

	// rename failed, fall back to copy (cross-filesystem or other issues)
	slog.Debug("rename failed, falling back to copy", "from", legacyPath, "to", newPath)

	if err := copyDir(legacyPath, newPath); err != nil {
		return fmt.Errorf("copy ledger contents: %w", err)
	}

	// verify the copy succeeded by checking for .git directory
	gitDir := filepath.Join(newPath, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		// rollback: remove partial copy
		_ = os.RemoveAll(newPath)
		return fmt.Errorf("copy verification failed (.git not found): %w", err)
	}

	// remove old directory only after successful copy
	if err := os.RemoveAll(legacyPath); err != nil {
		// log warning but don't fail - migration succeeded
		slog.Warn("failed to remove legacy ledger directory", "path", legacyPath, "error", err)
	}

	slog.Info("ledger migration complete", "new_path", newPath)
	return nil
}

// CreateTeamSymlinks creates symlinks from team context directories to the ledger.
// This is a placeholder for future implementation when team context integration
// with the new ledger structure is defined.
//
// The intent is to create bidirectional references:
//   - From ledger to team contexts it uses
//   - From team contexts to ledgers that reference them
func CreateTeamSymlinks(ledgerPath string, teamIDs []string) error {
	// placeholder for future implementation
	// this will create symlinks between ledger and team contexts
	if len(teamIDs) == 0 {
		return nil
	}

	slog.Debug("team symlinks not yet implemented", "ledger", ledgerPath, "teams", teamIDs)
	return nil
}
