package doctor

import "context"

// Status represents the result status of a doctor check.
type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
	StatusSkip Status = "skip"
)

// CheckResult represents the result of running a single doctor check.
type CheckResult struct {
	Name    string
	Status  Status
	Message string
	Fix     string // Optional remediation hint
}

// Check defines the interface for all doctor checks.
// Each check implements its own diagnostic logic.
type Check interface {
	Name() string
	Run(ctx context.Context) CheckResult
}
