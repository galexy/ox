package doctor

import (
	"errors"
	"os"
	"path/filepath"
	"time"
)

// Marker file names (in .sageox/ directory)
const (
	NeedsDoctorMarker      = ".needs-doctor"
	NeedsDoctorAgentMarker = ".needs-doctor-agent"
	sageoxDir              = ".sageox"
)

// NeedsDoctorHuman checks if .sageox/.needs-doctor exists.
// Returns false if the file doesn't exist or .sageox/ directory doesn't exist.
func NeedsDoctorHuman(gitRoot string) bool {
	path := filepath.Join(gitRoot, sageoxDir, NeedsDoctorMarker)
	_, err := os.Stat(path)
	return err == nil
}

// NeedsDoctorAgent checks if .sageox/.needs-doctor-agent exists.
// Returns false if the file doesn't exist or .sageox/ directory doesn't exist.
func NeedsDoctorAgent(gitRoot string) bool {
	path := filepath.Join(gitRoot, sageoxDir, NeedsDoctorAgentMarker)
	_, err := os.Stat(path)
	return err == nil
}

// SetNeedsDoctorHuman creates .sageox/.needs-doctor marker file.
// Creates an empty file if it doesn't exist, updates mtime if it does.
// Returns error if .sageox/ directory doesn't exist.
func SetNeedsDoctorHuman(gitRoot string) error {
	return touchMarker(gitRoot, NeedsDoctorMarker)
}

// SetNeedsDoctorAgent creates .sageox/.needs-doctor-agent marker file.
// Creates an empty file if it doesn't exist, updates mtime if it does.
// Returns error if .sageox/ directory doesn't exist.
func SetNeedsDoctorAgent(gitRoot string) error {
	return touchMarker(gitRoot, NeedsDoctorAgentMarker)
}

// ClearNeedsDoctorHuman removes .sageox/.needs-doctor marker file.
// Idempotent: returns nil if file doesn't exist.
func ClearNeedsDoctorHuman(gitRoot string) error {
	return clearMarker(gitRoot, NeedsDoctorMarker)
}

// ClearNeedsDoctorAgent removes .sageox/.needs-doctor-agent marker file.
// Idempotent: returns nil if file doesn't exist.
func ClearNeedsDoctorAgent(gitRoot string) error {
	return clearMarker(gitRoot, NeedsDoctorAgentMarker)
}

// GetDoctorNeeds returns which doctor types are needed.
// Returns (human, agent) booleans indicating which markers exist.
func GetDoctorNeeds(gitRoot string) (human, agent bool) {
	return NeedsDoctorHuman(gitRoot), NeedsDoctorAgent(gitRoot)
}

// touchMarker creates or updates a marker file in .sageox/ directory.
// Creates an empty file if it doesn't exist, updates mtime if it does.
func touchMarker(gitRoot, markerName string) error {
	sageoxPath := filepath.Join(gitRoot, sageoxDir)
	if _, err := os.Stat(sageoxPath); os.IsNotExist(err) {
		return errors.New(".sageox directory does not exist")
	}

	markerPath := filepath.Join(sageoxPath, markerName)

	// check if file exists
	if _, err := os.Stat(markerPath); err == nil {
		// file exists, update mtime
		now := time.Now()
		return os.Chtimes(markerPath, now, now)
	}

	// create empty file
	f, err := os.Create(markerPath)
	if err != nil {
		return err
	}
	return f.Close()
}

// clearMarker removes a marker file from .sageox/ directory.
// Idempotent: returns nil if file doesn't exist.
func clearMarker(gitRoot, markerName string) error {
	markerPath := filepath.Join(gitRoot, sageoxDir, markerName)
	err := os.Remove(markerPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
