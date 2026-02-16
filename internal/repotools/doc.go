// Package repotools provides VCS abstraction for repository operations.
//
// This package supports multiple version control systems with git as the
// primary focus. SVN support is included for enterprises like LinkedIn
// that still use SVN as of 2025.
//
// Key components:
//   - VCS detection and requirement checking
//   - Git identity detection for per-user session isolation
//   - Repository fingerprinting (initial commit hash, remote URL hashing)
//   - Secure hashing with salt to prevent enumeration attacks
package repotools
