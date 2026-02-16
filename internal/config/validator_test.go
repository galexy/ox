package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateAPIEndpoint(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name      string
		endpoint  string
		wantError bool
	}{
		{"empty endpoint is valid", "", false},
		{"valid https endpoint", "https://api.sageox.ai/", false},
		{"valid http endpoint", "http://localhost:8080", false},
		{"valid endpoint with path", "https://api.example.com/v1/", false},
		{"missing scheme", "api.sageox.ai", true},
		{"missing host", "https://", true},
		{"invalid url", "ht!tp://invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateAPIEndpoint(tt.endpoint)
			if tt.wantError {
				assert.Error(t, err, "ValidateAPIEndpoint(%q)", tt.endpoint)
			} else {
				assert.NoError(t, err, "ValidateAPIEndpoint(%q)", tt.endpoint)
			}
		})
	}
}

func TestValidateOrgSlug(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name      string
		slug      string
		wantError bool
	}{
		{"empty slug is valid", "", false},
		{"valid lowercase slug", "my-org", false},
		{"valid with numbers", "org123", false},
		{"valid with hyphens", "my-awesome-org", false},
		{"uppercase letters", "MyOrg", true},
		{"starts with hyphen", "-myorg", true},
		{"ends with hyphen", "myorg-", true},
		{"consecutive hyphens", "my--org", true},
		{"special characters", "my_org", true},
		{"spaces", "my org", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateOrgSlug(tt.slug)
			if tt.wantError {
				assert.Error(t, err, "ValidateOrgSlug(%q)", tt.slug)
			} else {
				assert.NoError(t, err, "ValidateOrgSlug(%q)", tt.slug)
			}
		})
	}
}

func TestValidateTeamSlug(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name      string
		slug      string
		wantError bool
	}{
		{"valid team slug", "team-alpha", false},
		{"invalid uppercase", "TeamAlpha", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateTeamSlug(tt.slug)
			if tt.wantError {
				assert.Error(t, err, "ValidateTeamSlug(%q)", tt.slug)
			} else {
				assert.NoError(t, err, "ValidateTeamSlug(%q)", tt.slug)
			}
		})
	}
}

func TestValidateProjectSlug(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name      string
		slug      string
		wantError bool
	}{
		{"valid project slug", "my-project", false},
		{"invalid special chars", "my_project", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateProjectSlug(tt.slug)
			if tt.wantError {
				assert.Error(t, err, "ValidateProjectSlug(%q)", tt.slug)
			} else {
				assert.NoError(t, err, "ValidateProjectSlug(%q)", tt.slug)
			}
		})
	}
}

func TestValidateUpdateFrequency(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name      string
		hours     int
		wantError bool
	}{
		{"valid positive hours", 24, false},
		{"valid small hours", 1, false},
		{"zero hours", 0, true},
		{"negative hours", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateUpdateFrequency(tt.hours)
			if tt.wantError {
				assert.Error(t, err, "ValidateUpdateFrequency(%d)", tt.hours)
			} else {
				assert.NoError(t, err, "ValidateUpdateFrequency(%d)", tt.hours)
			}
		})
	}
}

func TestValidateTimestamp(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name      string
		timestamp string
		wantError bool
	}{
		{"empty timestamp is valid", "", false},
		{"valid RFC3339 timestamp", "2024-01-15T10:30:00Z", false},
		{"valid with timezone", "2024-01-15T10:30:00-05:00", false},
		{"invalid format", "2024-01-15", true},
		{"invalid timestamp", "not-a-timestamp", true},
		{"malformed", "2024-13-45T99:99:99Z", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateTimestamp(tt.timestamp)
			if tt.wantError {
				assert.Error(t, err, "ValidateTimestamp(%q)", tt.timestamp)
			} else {
				assert.NoError(t, err, "ValidateTimestamp(%q)", tt.timestamp)
			}
		})
	}
}

func TestValidateID(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name      string
		id        string
		prefix    string
		wantError bool
	}{
		{"empty id is valid", "", "prj", false},
		{"valid prefixed id", "prj_abc123", "prj", false},
		{"wrong prefix", "ws_abc123", "prj", true},
		{"no prefix", "abc123", "prj", true},
		{"too short", "prj", "prj", true},
		{"no prefix check", "anyid123", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateID(tt.id, tt.prefix)
			if tt.wantError {
				assert.Error(t, err, "ValidateID(%q, %q)", tt.id, tt.prefix)
			} else {
				assert.NoError(t, err, "ValidateID(%q, %q)", tt.id, tt.prefix)
			}
		})
	}
}

func TestValidateProjectID(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name      string
		id        string
		wantError bool
	}{
		{"valid project id", "prj_abc123", false},
		{"wrong prefix", "proj_abc123", true},
		{"no prefix", "abc123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateProjectID(tt.id)
			if tt.wantError {
				assert.Error(t, err, "ValidateProjectID(%q)", tt.id)
			} else {
				assert.NoError(t, err, "ValidateProjectID(%q)", tt.id)
			}
		})
	}
}

func TestValidateWorkspaceID(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name      string
		id        string
		wantError bool
	}{
		{"valid workspace id", "ws_abc123", false},
		{"wrong prefix", "workspace_abc123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateWorkspaceID(tt.id)
			if tt.wantError {
				assert.Error(t, err, "ValidateWorkspaceID(%q)", tt.id)
			} else {
				assert.NoError(t, err, "ValidateWorkspaceID(%q)", tt.id)
			}
		})
	}
}

func TestValidateRepoID(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name      string
		id        string
		wantError bool
	}{
		{"empty id is valid", "", false},
		{"valid repo id with uuid", "repo_01234567-89ab-cdef-0123-456789abcdef", false},
		{"wrong prefix", "repository_01234567-89ab-cdef-0123-456789abcdef", true},
		{"invalid uuid format", "repo_invalid", true},
		{"no prefix", "01234567-89ab-cdef-0123-456789abcdef", true},
		{"uppercase uuid", "repo_01234567-89AB-CDEF-0123-456789ABCDEF", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateRepoID(tt.id)
			if tt.wantError {
				assert.Error(t, err, "ValidateRepoID(%q)", tt.id)
			} else {
				assert.NoError(t, err, "ValidateRepoID(%q)", tt.id)
			}
		})
	}
}

func TestValidateTeamID(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name      string
		id        string
		wantError bool
	}{
		{"valid team id", "team_abc123", false},
		{"wrong prefix", "tm_abc123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateTeamID(tt.id)
			if tt.wantError {
				assert.Error(t, err, "ValidateTeamID(%q)", tt.id)
			} else {
				assert.NoError(t, err, "ValidateTeamID(%q)", tt.id)
			}
		})
	}
}

func TestValidateRemoteHash(t *testing.T) {
	v := NewValidator()

	// generate valid 64-char hex string
	validHash := strings.Repeat("a", 64)
	invalidHash := strings.Repeat("g", 64) // invalid hex char

	tests := []struct {
		name      string
		hash      string
		wantError bool
	}{
		{"empty hash is valid", "", false},
		{"valid 64-char hex", validHash, false},
		{"too short", strings.Repeat("a", 32), true},
		{"too long", strings.Repeat("a", 128), true},
		{"invalid hex chars", invalidHash, true},
		{"uppercase hex", strings.Repeat("A", 64), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateRemoteHash(tt.hash)
			if tt.wantError {
				assert.Error(t, err, "ValidateRemoteHash()")
			} else {
				assert.NoError(t, err, "ValidateRemoteHash()")
			}
		})
	}
}

func TestValidateConfigVersion(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name      string
		version   string
		wantError bool
	}{
		{"valid numeric version", "2", false},
		{"valid multi-digit", "123", false},
		{"empty version", "", true},
		{"non-numeric", "v2", true},
		{"decimal version", "2.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateConfigVersion(tt.version)
			if tt.wantError {
				assert.Error(t, err, "ValidateConfigVersion(%q)", tt.version)
			} else {
				assert.NoError(t, err, "ValidateConfigVersion(%q)", tt.version)
			}
		})
	}
}

func TestValidateAttribution(t *testing.T) {
	v := NewValidator()

	longString := strings.Repeat("a", 501)
	normalString := "Normal attribution"

	tests := []struct {
		name      string
		attr      *Attribution
		wantError bool
	}{
		{"nil attribution is valid", nil, false},
		{
			"valid attribution",
			&Attribution{Commit: &normalString, PR: &normalString},
			false,
		},
		{
			"commit too long",
			&Attribution{Commit: &longString},
			true,
		},
		{
			"PR too long",
			&Attribution{PR: &longString},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := v.ValidateAttribution(tt.attr)
			if tt.wantError {
				assert.NotEmpty(t, errs, "ValidateAttribution() expected errors")
			} else {
				assert.Empty(t, errs, "ValidateAttribution()")
			}
		})
	}
}

func TestValidateProjectConfig(t *testing.T) {
	v := NewValidator()

	validTimestamp := "2024-01-15T10:30:00Z"

	tests := []struct {
		name      string
		cfg       *ProjectConfig
		wantError bool
	}{
		{"nil config", nil, true},
		{
			"valid minimal config",
			&ProjectConfig{
				ConfigVersion:        "2",
				UpdateFrequencyHours: 24,
			},
			false,
		},
		{
			"valid complete config",
			&ProjectConfig{
				ConfigVersion:        "2",
				Org:                  "my-org",
				Team:                 "my-team",
				Project:              "my-project",
				ProjectID:            "prj_abc123",
				WorkspaceID:          "ws_abc123",
				RepoID:               "repo_01234567-89ab-cdef-0123-456789abcdef",
				TeamID:               "team_abc123",
				UpdateFrequencyHours: 24,
				LastUpdateCheckUTC:   &validTimestamp,
				RepoRemoteHashes:     []string{strings.Repeat("a", 64)},
			},
			false,
		},
		{
			"invalid config version",
			&ProjectConfig{
				ConfigVersion:        "",
				UpdateFrequencyHours: 24,
			},
			true,
		},
		{
			"invalid update frequency",
			&ProjectConfig{
				ConfigVersion:        "2",
				UpdateFrequencyHours: 0,
			},
			true,
		},
		{
			"invalid org slug",
			&ProjectConfig{
				ConfigVersion:        "2",
				Org:                  "My_Org",
				UpdateFrequencyHours: 24,
			},
			true,
		},
		{
			"invalid project id",
			&ProjectConfig{
				ConfigVersion:        "2",
				ProjectID:            "invalid",
				UpdateFrequencyHours: 24,
			},
			true,
		},
		{
			"invalid remote hash",
			&ProjectConfig{
				ConfigVersion:        "2",
				UpdateFrequencyHours: 24,
				RepoRemoteHashes:     []string{"tooshort"},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := v.ValidateProjectConfig(tt.cfg)
			if tt.wantError {
				assert.NotEmpty(t, errs, "ValidateProjectConfig() expected errors")
			} else {
				assert.Empty(t, errs, "ValidateProjectConfig()")
			}
		})
	}
}

func TestValidateUserConfig(t *testing.T) {
	v := NewValidator()

	longString := strings.Repeat("a", 501)

	tests := []struct {
		name      string
		cfg       *UserConfig
		wantError bool
	}{
		{"nil config is valid", nil, false},
		{
			"valid config with defaults",
			&UserConfig{},
			false,
		},
		{
			"valid config with attribution",
			&UserConfig{
				Attribution: &Attribution{
					Commit: StringPtr("Custom commit"),
					PR:     StringPtr("Custom PR"),
				},
			},
			false,
		},
		{
			"invalid long attribution",
			&UserConfig{
				Attribution: &Attribution{
					Commit: &longString,
				},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := v.ValidateUserConfig(tt.cfg)
			if tt.wantError {
				assert.NotEmpty(t, errs, "ValidateUserConfig() expected errors")
			} else {
				assert.Empty(t, errs, "ValidateUserConfig()")
			}
		})
	}
}
