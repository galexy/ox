package hooks

import (
	"encoding/json"
	"fmt"
	"os"
)

// EnsureDir creates directory and all parents if they don't exist
func EnsureDir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return nil
}

// ReadJSON reads a JSON file into the provided struct
// handles non-existent and empty files gracefully by leaving v unchanged
func ReadJSON(path string, v any) error {
	// check if file exists
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// file doesn't exist - not an error, v remains unchanged
			return nil
		}
		return fmt.Errorf("failed to read file: %w", err)
	}

	// handle empty file
	if len(data) == 0 {
		return nil
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	return nil
}

// WriteJSON writes a struct to a JSON file with given permissions
// marshals with indentation for readability
func WriteJSON(path string, v any, perm os.FileMode) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(path, data, perm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// FileExists returns true if the file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
