package config

import (
	"encoding/json"
	"fmt"
	"math/rand" //nolint:gosec // G404: hint probability doesn't need crypto-secure randomness
	"os"
	"path/filepath"
	"time"
)

const (
	healthFilename = "health.json"
	healthCacheDir = "cache"
)

// Health tracks diagnostic command execution times.
// Stored in .sageox/cache/health.json (machine-specific, not committed).
// Note: omitempty is not used on time.Time fields because encoding/json
// does not treat zero time as empty for nested structs.
type Health struct {
	LastDoctorAt    time.Time `json:"last_doctor_at"`
	LastDoctorFixAt time.Time `json:"last_doctor_fix_at"`
}

// Staleness thresholds and corresponding hint probabilities.
// Progressive nudging increases as time since last doctor run grows.
const (
	StalenessWeek  = 7 * 24 * time.Hour
	StalenessWeek2 = 14 * 24 * time.Hour
	StalenessWeek3 = 21 * 24 * time.Hour
	StalenessMonth = 30 * 24 * time.Hour

	// hint probabilities per staleness tier
	ProbabilityNone  = 0.0  // < 1 week: no hints
	ProbabilityWeek1 = 0.10 // 1 week: 10%
	ProbabilityWeek2 = 0.30 // 2 weeks: 30%
	ProbabilityWeek3 = 0.50 // 3 weeks: 50%
	ProbabilityMonth = 0.80 // 1 month+: 80%
)

// LoadHealth loads health data from .sageox/cache/health.json.
// Returns empty Health if file doesn't exist.
func LoadHealth(projectRoot string) (*Health, error) {
	if projectRoot == "" {
		return &Health{}, nil
	}

	healthPath := filepath.Join(projectRoot, sageoxDir, healthCacheDir, healthFilename)
	data, err := os.ReadFile(healthPath)
	if os.IsNotExist(err) {
		return &Health{}, nil
	}
	if err != nil {
		return nil, err
	}

	var h Health
	if err := json.Unmarshal(data, &h); err != nil {
		// corrupted file - return empty health rather than error
		return &Health{}, nil
	}

	return &h, nil
}

// SaveHealth saves health data to .sageox/cache/health.json.
func SaveHealth(projectRoot string, h *Health) error {
	if projectRoot == "" || h == nil {
		return nil
	}

	// ensure cache directory exists
	cachePath := filepath.Join(projectRoot, sageoxDir, healthCacheDir)
	if err := os.MkdirAll(cachePath, 0755); err != nil {
		return err
	}

	// ensure cache has gitignore (prevents accidental commits)
	ensureCacheGitignore(cachePath)

	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}

	healthPath := filepath.Join(cachePath, healthFilename)
	return os.WriteFile(healthPath, data, 0600)
}

// ensureCacheGitignore creates a .gitignore in the cache directory if it doesn't exist.
func ensureCacheGitignore(cacheDir string) {
	gitignorePath := filepath.Join(cacheDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); err == nil {
		return // already exists
	}

	content := `# Cache files - machine-specific, do not commit
*
!.gitignore
`
	_ = os.WriteFile(gitignorePath, []byte(content), 0644)
}

// RecordDoctorRun updates LastDoctorAt timestamp.
func (h *Health) RecordDoctorRun() {
	h.LastDoctorAt = time.Now().UTC()
}

// RecordDoctorFixRun updates LastDoctorFixAt timestamp.
func (h *Health) RecordDoctorFixRun() {
	h.LastDoctorFixAt = time.Now().UTC()
}

// DoctorStaleness returns duration since last doctor run.
// Returns max duration if never run.
func (h *Health) DoctorStaleness() time.Duration {
	if h.LastDoctorAt.IsZero() {
		return time.Duration(1<<63 - 1) // max duration
	}
	return time.Since(h.LastDoctorAt)
}

// DoctorFixStaleness returns duration since last doctor --fix run.
// Returns max duration if never run.
func (h *Health) DoctorFixStaleness() time.Duration {
	if h.LastDoctorFixAt.IsZero() {
		return time.Duration(1<<63 - 1) // max duration
	}
	return time.Since(h.LastDoctorFixAt)
}

// ShouldHintDoctor returns true if a doctor hint should be shown.
// Uses probabilistic display based on staleness.
func (h *Health) ShouldHintDoctor() bool {
	staleness := h.DoctorStaleness()
	probability := stalenessToProb(staleness)
	return rand.Float64() < probability
}

// ShouldHintDoctorFix returns true if a doctor --fix hint should be shown.
// Uses probabilistic display based on staleness.
func (h *Health) ShouldHintDoctorFix() bool {
	staleness := h.DoctorFixStaleness()
	probability := stalenessToProb(staleness)
	return rand.Float64() < probability
}

// DoctorHintMessage returns appropriate hint message based on staleness.
func (h *Health) DoctorHintMessage() string {
	staleness := h.DoctorStaleness()
	return formatDoctorHint(staleness, false)
}

// DoctorFixHintMessage returns appropriate hint message based on staleness.
func (h *Health) DoctorFixHintMessage() string {
	staleness := h.DoctorFixStaleness()
	return formatDoctorHint(staleness, true)
}

// stalenessToProb converts staleness duration to hint probability.
func stalenessToProb(staleness time.Duration) float64 {
	switch {
	case staleness < StalenessWeek:
		return ProbabilityNone
	case staleness < StalenessWeek2:
		return ProbabilityWeek1
	case staleness < StalenessWeek3:
		return ProbabilityWeek2
	case staleness < StalenessMonth:
		return ProbabilityWeek3
	default:
		return ProbabilityMonth
	}
}

// formatDoctorHint creates a human-readable hint message.
func formatDoctorHint(staleness time.Duration, fix bool) string {
	cmd := "`ox doctor`"
	if fix {
		cmd = "`ox doctor --fix`"
	}

	days := int(staleness.Hours() / 24)
	switch {
	case days < 7:
		return "" // no hint for recent runs
	case days < 14:
		return "It's been about a week since last " + cmd + ". Consider running it periodically."
	case days < 21:
		return "It's been 2 weeks since last " + cmd + ". Running it helps catch config drift."
	case days < 30:
		return "It's been 3 weeks since last " + cmd + ". A quick health check is recommended."
	default:
		weeks := days / 7
		return "It's been " + formatWeeks(weeks) + " since last " + cmd + ". Time for a checkup!"
	}
}

func formatWeeks(weeks int) string {
	if weeks == 1 {
		return "1 week"
	}
	if weeks < 5 {
		return fmt.Sprintf("%d weeks", weeks)
	}
	months := weeks / 4
	if months == 1 {
		return "about a month"
	}
	return fmt.Sprintf("over %d months", months)
}
