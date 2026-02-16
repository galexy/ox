package config

// Attribution configures how ox-guided work is credited in plans, git commits, and PRs.
// Use pointer fields to distinguish between "not set" (nil) and "explicitly disabled" ("").
type Attribution struct {
	Plan   *string `yaml:"plan,omitempty" json:"plan,omitempty"`
	Commit *string `yaml:"commit,omitempty" json:"commit,omitempty"`
	PR     *string `yaml:"pr,omitempty" json:"pr,omitempty"`
}

// default attribution values - friendly for humans, concise for git
// IMPORTANT: Commit attribution MUST use exact format "Co-Authored-By: Name <email>"
// for GitHub to recognize and display contributor avatars/profile links.
// canonical email: constants.SageOxGitEmail (ox@sageox.ai)
var (
	defaultPlanAttribution       = "This plan was informed by SageOx context"
	defaultPlanFooterAttribution = "Guided by SageOx"
	defaultCommitAttribution     = "Co-Authored-By: SageOx <ox@sageox.ai>"
	defaultPRAttribution         = "Co-Authored-By: [SageOx](https://github.com/SageOx)"
)

// DefaultAttribution returns the default attribution settings.
// These are used when no user or project config overrides are set.
func DefaultAttribution() Attribution {
	return Attribution{
		Plan:   &defaultPlanAttribution,
		Commit: &defaultCommitAttribution,
		PR:     &defaultPRAttribution,
	}
}

// ResolvedAttribution holds the final resolved attribution values (non-pointer).
// Use this after merging configs for easier consumption.
type ResolvedAttribution struct {
	Plan       string `json:"plan"`
	PlanFooter string `json:"plan_footer"` // exact footer text for plans ("Guided by SageOx")
	Commit     string `json:"commit"`
	PR         string `json:"pr"`
}

// GetPlan returns the plan attribution value, or empty string if nil
func (a *Attribution) GetPlan() string {
	if a == nil || a.Plan == nil {
		return ""
	}
	return *a.Plan
}

// GetCommit returns the commit attribution value, or empty string if nil
func (a *Attribution) GetCommit() string {
	if a == nil || a.Commit == nil {
		return ""
	}
	return *a.Commit
}

// GetPR returns the PR attribution value, or empty string if nil
func (a *Attribution) GetPR() string {
	if a == nil || a.PR == nil {
		return ""
	}
	return *a.PR
}

// IsPlanSet returns true if plan attribution is explicitly set (including to empty string)
func (a *Attribution) IsPlanSet() bool {
	return a != nil && a.Plan != nil
}

// IsCommitSet returns true if commit attribution is explicitly set (including to empty string)
func (a *Attribution) IsCommitSet() bool {
	return a != nil && a.Commit != nil
}

// IsPRSet returns true if PR attribution is explicitly set (including to empty string)
func (a *Attribution) IsPRSet() bool {
	return a != nil && a.PR != nil
}

// MergeAttribution merges project and user attribution with project taking precedence.
// Returns resolved values with defaults applied where not overridden.
//
// Precedence (highest to lowest):
//  1. Project config (repo-specific)
//  2. User config (user preferences)
//  3. Default values
//
// Setting a field to empty string ("") explicitly disables that attribution type.
// Leaving a field unset (nil) means "use lower priority config or default".
func MergeAttribution(project, user *Attribution) ResolvedAttribution {
	result := ResolvedAttribution{
		Plan:       defaultPlanAttribution,
		PlanFooter: defaultPlanFooterAttribution,
		Commit:     defaultCommitAttribution,
		PR:         defaultPRAttribution,
	}

	// apply user config first (lower priority)
	if user != nil {
		if user.Plan != nil {
			result.Plan = *user.Plan
		}
		if user.Commit != nil {
			result.Commit = *user.Commit
		}
		if user.PR != nil {
			result.PR = *user.PR
		}
	}

	// project config overrides (higher priority)
	if project != nil {
		if project.Plan != nil {
			result.Plan = *project.Plan
		}
		if project.Commit != nil {
			result.Commit = *project.Commit
		}
		if project.PR != nil {
			result.PR = *project.PR
		}
	}

	return result
}

// DefaultPlanFooterAttribution returns the canonical plan footer text.
// This is always-on (not config-gated) as a transparency requirement.
func DefaultPlanFooterAttribution() string {
	return defaultPlanFooterAttribution
}

// StringPtr is a helper to create a pointer to a string value
func StringPtr(s string) *string {
	return &s
}
