package config

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Validator provides reusable config validation functions
type Validator struct{}

// NewValidator creates a new Validator instance
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateAPIEndpoint validates an API endpoint URL
func (v *Validator) ValidateAPIEndpoint(endpoint string) error {
	if endpoint == "" {
		return nil // optional
	}

	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid API endpoint: %w", err)
	}

	// ensure it has a scheme
	if parsedURL.Scheme == "" {
		return fmt.Errorf("invalid API endpoint: missing scheme (http/https)")
	}

	// ensure it has a host
	if parsedURL.Host == "" {
		return fmt.Errorf("invalid API endpoint: missing host")
	}

	return nil
}

// ValidateOrgSlug validates an organization slug
// slugs should be lowercase alphanumeric with hyphens (kebab-case)
func (v *Validator) ValidateOrgSlug(slug string) error {
	if slug == "" {
		return nil // optional
	}

	// slugs should be lowercase alphanumeric with hyphens
	for _, r := range slug {
		isLower := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		isHyphen := r == '-'
		if !isLower && !isDigit && !isHyphen {
			return fmt.Errorf("invalid org slug: must be lowercase alphanumeric with hyphens")
		}
	}

	// cannot start or end with hyphen
	if strings.HasPrefix(slug, "-") || strings.HasSuffix(slug, "-") {
		return fmt.Errorf("invalid org slug: cannot start or end with hyphen")
	}

	// cannot have consecutive hyphens
	if strings.Contains(slug, "--") {
		return fmt.Errorf("invalid org slug: cannot contain consecutive hyphens")
	}

	return nil
}

// ValidateTeamSlug validates a team slug (same rules as org slug)
func (v *Validator) ValidateTeamSlug(slug string) error {
	if slug == "" {
		return nil // optional
	}
	return v.ValidateOrgSlug(slug) // same validation rules
}

// ValidateProjectSlug validates a project slug (same rules as org slug)
func (v *Validator) ValidateProjectSlug(slug string) error {
	if slug == "" {
		return nil // optional
	}
	return v.ValidateOrgSlug(slug) // same validation rules
}

// ValidateUpdateFrequency validates update frequency in hours
func (v *Validator) ValidateUpdateFrequency(hours int) error {
	if hours <= 0 {
		return fmt.Errorf("update frequency must be greater than 0")
	}
	return nil
}

// ValidateTimestamp validates an ISO 8601 timestamp string
func (v *Validator) ValidateTimestamp(ts string) error {
	if ts == "" {
		return nil // optional
	}

	if _, err := time.Parse(time.RFC3339, ts); err != nil {
		return fmt.Errorf("not a valid ISO 8601 timestamp: %s", ts)
	}

	return nil
}

// ValidateID validates a prefixed ID (e.g., prj_xxx, ws_xxx, repo_xxx)
func (v *Validator) ValidateID(id, prefix string) error {
	if id == "" {
		return nil // optional
	}

	if prefix != "" && !strings.HasPrefix(id, prefix+"_") {
		return fmt.Errorf("id must start with %s_", prefix)
	}

	// basic length check - IDs should not be too short
	if len(id) < 5 {
		return fmt.Errorf("id is too short")
	}

	return nil
}

// ValidateProjectID validates a project ID (prj_xxx format)
func (v *Validator) ValidateProjectID(id string) error {
	return v.ValidateID(id, "prj")
}

// ValidateWorkspaceID validates a workspace ID (ws_xxx format)
func (v *Validator) ValidateWorkspaceID(id string) error {
	return v.ValidateID(id, "ws")
}

// ValidateRepoID validates a repo ID (repo_xxx format with UUIDv7)
func (v *Validator) ValidateRepoID(id string) error {
	if id == "" {
		return nil // optional
	}

	if !strings.HasPrefix(id, "repo_") {
		return fmt.Errorf("repo id must start with repo_")
	}

	// extract the UUID part after repo_
	uuidPart := strings.TrimPrefix(id, "repo_")

	// UUIDv7 pattern: 8-4-4-4-12 hex characters
	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if !uuidPattern.MatchString(uuidPart) {
		return fmt.Errorf("repo id must be in format repo_<uuid>")
	}

	return nil
}

// ValidateTeamID validates a team ID
func (v *Validator) ValidateTeamID(id string) error {
	return v.ValidateID(id, "team")
}

// ValidateRemoteHash validates a remote hash (SHA256 hex string)
func (v *Validator) ValidateRemoteHash(hash string) error {
	if hash == "" {
		return nil // optional
	}

	// SHA256 produces 64 hex characters
	if len(hash) != 64 {
		return fmt.Errorf("remote hash must be 64 characters (SHA256)")
	}

	// must be hex
	hexPattern := regexp.MustCompile(`^[0-9a-f]{64}$`)
	if !hexPattern.MatchString(hash) {
		return fmt.Errorf("remote hash must be hexadecimal")
	}

	return nil
}

// ValidateConfigVersion validates a config version string
func (v *Validator) ValidateConfigVersion(version string) error {
	if version == "" {
		return fmt.Errorf("config version cannot be empty")
	}

	// version should be numeric
	for _, r := range version {
		if r < '0' || r > '9' {
			return fmt.Errorf("config version must be numeric")
		}
	}

	return nil
}

// ValidateAttribution validates attribution configuration
func (v *Validator) ValidateAttribution(attr *Attribution) []error {
	if attr == nil {
		return nil
	}

	var errs []error

	// commit and PR attribution can be any string or empty
	// no specific validation needed, but we could add length limits if desired

	// optional: warn if attribution is excessively long
	maxLength := 500
	if attr.Commit != nil && len(*attr.Commit) > maxLength {
		errs = append(errs, fmt.Errorf("commit attribution is too long (max %d characters)", maxLength))
	}
	if attr.PR != nil && len(*attr.PR) > maxLength {
		errs = append(errs, fmt.Errorf("PR attribution is too long (max %d characters)", maxLength))
	}

	return errs
}

// ValidateProjectConfig validates a complete ProjectConfig object
func (v *Validator) ValidateProjectConfig(cfg *ProjectConfig) []error {
	if cfg == nil {
		return []error{fmt.Errorf("config is nil")}
	}

	var errs []error

	// validate config version
	if err := v.ValidateConfigVersion(cfg.ConfigVersion); err != nil {
		errs = append(errs, err)
	}

	// validate update frequency
	if err := v.ValidateUpdateFrequency(cfg.UpdateFrequencyHours); err != nil {
		errs = append(errs, err)
	}

	// validate last update check timestamp
	if cfg.LastUpdateCheckUTC != nil && *cfg.LastUpdateCheckUTC != "" {
		if err := v.ValidateTimestamp(*cfg.LastUpdateCheckUTC); err != nil {
			errs = append(errs, fmt.Errorf("last update check: %w", err))
		}
	}

	// validate slugs
	if err := v.ValidateOrgSlug(cfg.Org); err != nil {
		errs = append(errs, err)
	}
	if err := v.ValidateTeamSlug(cfg.Team); err != nil {
		errs = append(errs, err)
	}
	if err := v.ValidateProjectSlug(cfg.Project); err != nil {
		errs = append(errs, err)
	}

	// validate IDs
	if err := v.ValidateProjectID(cfg.ProjectID); err != nil {
		errs = append(errs, err)
	}
	if err := v.ValidateWorkspaceID(cfg.WorkspaceID); err != nil {
		errs = append(errs, err)
	}
	if err := v.ValidateRepoID(cfg.RepoID); err != nil {
		errs = append(errs, err)
	}
	if err := v.ValidateTeamID(cfg.TeamID); err != nil {
		errs = append(errs, err)
	}

	// validate remote hashes
	for i, hash := range cfg.RepoRemoteHashes {
		if err := v.ValidateRemoteHash(hash); err != nil {
			errs = append(errs, fmt.Errorf("remote hash[%d]: %w", i, err))
		}
	}

	// validate attribution
	if attrErrs := v.ValidateAttribution(cfg.Attribution); len(attrErrs) > 0 {
		errs = append(errs, attrErrs...)
	}

	return errs
}

// ValidateUserConfig validates a UserConfig object
func (v *Validator) ValidateUserConfig(cfg *UserConfig) []error {
	if cfg == nil {
		return nil
	}

	var errs []error

	// validate attribution
	if attrErrs := v.ValidateAttribution(cfg.Attribution); len(attrErrs) > 0 {
		errs = append(errs, attrErrs...)
	}

	// validate display name
	if err := ValidateDisplayName(cfg.DisplayName); err != nil {
		errs = append(errs, err)
	}

	return errs
}
