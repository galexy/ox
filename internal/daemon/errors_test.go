package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestErrorStore_AddAndGet(t *testing.T) {
	// use temp directory for test
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "errors.json")

	store := NewErrorStore(storePath)

	// should start empty
	if count := store.Count(); count != 0 {
		t.Errorf("expected empty store, got %d errors", count)
	}

	// add an error
	err := StoredError{
		Code:     "test_error",
		Message:  "Test error message",
		Severity: "error",
	}
	store.Add(err)

	// should have one error
	if count := store.Count(); count != 1 {
		t.Errorf("expected 1 error, got %d", count)
	}

	// should be unviewed
	unviewed := store.GetUnviewed()
	if len(unviewed) != 1 {
		t.Errorf("expected 1 unviewed error, got %d", len(unviewed))
	}

	if unviewed[0].Message != "Test error message" {
		t.Errorf("expected message 'Test error message', got %q", unviewed[0].Message)
	}

	// ID should have been generated
	if unviewed[0].ID == "" {
		t.Error("expected ID to be generated")
	}

	// timestamp should have been set
	if unviewed[0].Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}
}

func TestErrorStore_MarkViewed(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "errors.json")

	store := NewErrorStore(storePath)

	err := StoredError{
		Code:     "test_error",
		Message:  "Test error",
		Severity: "warning",
	}
	store.Add(err)

	// get the error to find its ID
	unviewed := store.GetUnviewed()
	if len(unviewed) != 1 {
		t.Fatalf("expected 1 unviewed error, got %d", len(unviewed))
	}

	id := unviewed[0].ID

	// mark as viewed
	store.MarkViewed(id)

	// should no longer be unviewed
	unviewed = store.GetUnviewed()
	if len(unviewed) != 0 {
		t.Errorf("expected 0 unviewed errors after mark, got %d", len(unviewed))
	}

	// but should still exist
	if count := store.Count(); count != 1 {
		t.Errorf("expected 1 total error, got %d", count)
	}
}

func TestErrorStore_MarkAllViewed(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "errors.json")

	store := NewErrorStore(storePath)

	// add multiple errors
	store.Add(StoredError{Code: "err1", Message: "Error 1", Severity: "error"})
	store.Add(StoredError{Code: "err2", Message: "Error 2", Severity: "warning"})
	store.Add(StoredError{Code: "err3", Message: "Error 3", Severity: "error"})

	// all should be unviewed
	if count := store.UnviewedCount(); count != 3 {
		t.Errorf("expected 3 unviewed errors, got %d", count)
	}

	// mark all as viewed
	store.MarkAllViewed()

	// none should be unviewed
	if count := store.UnviewedCount(); count != 0 {
		t.Errorf("expected 0 unviewed errors after mark all, got %d", count)
	}
}

func TestErrorStore_Dedup(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "errors.json")

	store := NewErrorStore(storePath)

	// add error with same code twice
	store.Add(StoredError{Code: "same_code", Message: "First message", Severity: "error"})
	store.Add(StoredError{Code: "same_code", Message: "Updated message", Severity: "warning"})

	// should only have one error (deduped by code)
	if count := store.Count(); count != 1 {
		t.Errorf("expected 1 error after dedup, got %d", count)
	}

	// message should be updated
	all := store.GetAll()
	if all[0].Message != "Updated message" {
		t.Errorf("expected updated message, got %q", all[0].Message)
	}
}

func TestErrorStore_Cleanup(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "errors.json")

	store := NewErrorStore(storePath)

	// add an old error and a recent error
	oldErr := StoredError{
		Code:      "old_error",
		Message:   "Old error",
		Severity:  "error",
		Timestamp: time.Now().Add(-48 * time.Hour), // 2 days old
	}
	store.Add(oldErr)

	recentErr := StoredError{
		Code:     "recent_error",
		Message:  "Recent error",
		Severity: "warning",
	}
	store.Add(recentErr)

	// cleanup errors older than 1 day
	store.Cleanup(24 * time.Hour)

	// should only have the recent error
	if count := store.Count(); count != 1 {
		t.Errorf("expected 1 error after cleanup, got %d", count)
	}

	all := store.GetAll()
	if all[0].Code != "recent_error" {
		t.Errorf("expected recent_error to remain, got %q", all[0].Code)
	}
}

func TestErrorStore_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "errors.json")

	// create store and add error
	store1 := NewErrorStore(storePath)
	store1.Add(StoredError{
		Code:     "persist_test",
		Message:  "Persisted error",
		Severity: "error",
	})

	// create new store from same path (simulates daemon restart)
	store2 := NewErrorStore(storePath)

	// should have the error
	if count := store2.Count(); count != 1 {
		t.Errorf("expected 1 error after reload, got %d", count)
	}

	all := store2.GetAll()
	if all[0].Code != "persist_test" {
		t.Errorf("expected persist_test error, got %q", all[0].Code)
	}
}

func TestErrorStore_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "errors.json")

	store := NewErrorStore(storePath)

	// add some errors
	store.Add(StoredError{Code: "err1", Message: "Error 1", Severity: "error"})
	store.Add(StoredError{Code: "err2", Message: "Error 2", Severity: "warning"})

	if count := store.Count(); count != 2 {
		t.Errorf("expected 2 errors, got %d", count)
	}

	// clear all
	store.Clear()

	if count := store.Count(); count != 0 {
		t.Errorf("expected 0 errors after clear, got %d", count)
	}

	// verify persisted
	store2 := NewErrorStore(storePath)
	if count := store2.Count(); count != 0 {
		t.Errorf("expected 0 errors after reload, got %d", count)
	}
}

func TestErrorStore_SortByTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "errors.json")

	store := NewErrorStore(storePath)

	// add errors with specific timestamps
	store.Add(StoredError{
		Code:      "oldest",
		Message:   "Oldest error",
		Severity:  "error",
		Timestamp: time.Now().Add(-3 * time.Hour),
	})
	store.Add(StoredError{
		Code:      "newest",
		Message:   "Newest error",
		Severity:  "error",
		Timestamp: time.Now().Add(-1 * time.Hour),
	})
	store.Add(StoredError{
		Code:      "middle",
		Message:   "Middle error",
		Severity:  "error",
		Timestamp: time.Now().Add(-2 * time.Hour),
	})

	// GetUnviewed should return newest first
	unviewed := store.GetUnviewed()
	if len(unviewed) != 3 {
		t.Fatalf("expected 3 errors, got %d", len(unviewed))
	}

	if unviewed[0].Code != "newest" {
		t.Errorf("expected newest first, got %q", unviewed[0].Code)
	}
	if unviewed[1].Code != "middle" {
		t.Errorf("expected middle second, got %q", unviewed[1].Code)
	}
	if unviewed[2].Code != "oldest" {
		t.Errorf("expected oldest last, got %q", unviewed[2].Code)
	}
}

func TestNewStoredError(t *testing.T) {
	err := NewStoredError(ErrorCodeSyncFailed, "Sync failed due to network error", "error")

	if err.Code != ErrorCodeSyncFailed {
		t.Errorf("expected code %q, got %q", ErrorCodeSyncFailed, err.Code)
	}
	if err.Message != "Sync failed due to network error" {
		t.Errorf("unexpected message: %q", err.Message)
	}
	if err.Severity != "error" {
		t.Errorf("expected severity error, got %q", err.Severity)
	}
	if err.Viewed {
		t.Error("expected Viewed to be false")
	}
	if err.ID == "" {
		t.Error("expected ID to be generated")
	}
	if err.Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}
}

func TestErrorStorePath(t *testing.T) {
	path := ErrorStorePath()
	if path == "" {
		t.Error("expected non-empty path")
	}

	// should be in daemon state directory
	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got %q", path)
	}
}

func TestErrorStore_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "errors.json")

	// create empty file
	if err := os.WriteFile(storePath, []byte{}, 0600); err != nil {
		t.Fatalf("failed to create empty file: %v", err)
	}

	// should handle empty file gracefully
	store := NewErrorStore(storePath)
	if count := store.Count(); count != 0 {
		t.Errorf("expected 0 errors for empty file, got %d", count)
	}
}

func TestErrorStore_CorruptFile(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "errors.json")

	// create corrupt file
	if err := os.WriteFile(storePath, []byte("not valid json"), 0600); err != nil {
		t.Fatalf("failed to create corrupt file: %v", err)
	}

	// should handle corrupt file gracefully (start fresh)
	store := NewErrorStore(storePath)
	if count := store.Count(); count != 0 {
		t.Errorf("expected 0 errors for corrupt file, got %d", count)
	}

	// should be able to add errors
	store.Add(StoredError{Code: "test", Message: "Test", Severity: "error"})
	if count := store.Count(); count != 1 {
		t.Errorf("expected 1 error after add, got %d", count)
	}
}
