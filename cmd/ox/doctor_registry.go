package main

import (
	"sort"
	"sync"
)

// Check represents a single diagnostic check that can be run.
// Implementations should be stateless and safe for concurrent execution.
type Check interface {
	// Name returns the display name for this check
	Name() string

	// Category returns the category this check belongs to
	Category() string

	// Run executes the check and returns a result.
	// The fix parameter indicates whether the check should attempt repairs.
	Run(fix bool) checkResult
}

// CheckRegistry manages registration and execution of diagnostic checks.
// It provides a centralized way to register checks and organize them by category.
type CheckRegistry struct {
	mu     sync.RWMutex
	checks []Check
}

// NewCheckRegistry creates a new empty check registry.
func NewCheckRegistry() *CheckRegistry {
	return &CheckRegistry{
		checks: make([]Check, 0),
	}
}

// Register adds a check to the registry.
// This is typically called during package initialization.
func (r *CheckRegistry) Register(check Check) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checks = append(r.checks, check)
}

// RunAll executes all registered checks organized by category.
// Returns a slice of checkCategory results suitable for display.
func (r *CheckRegistry) RunAll(fix bool) []checkCategory {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// group checks by category
	categoryMap := make(map[string][]checkResult)
	categoryOrder := make([]string, 0)

	for _, check := range r.checks {
		cat := check.Category()
		if _, exists := categoryMap[cat]; !exists {
			categoryOrder = append(categoryOrder, cat)
		}
		result := check.Run(fix)
		categoryMap[cat] = append(categoryMap[cat], result)
	}

	// convert to checkCategory slice preserving registration order
	categories := make([]checkCategory, 0, len(categoryOrder))
	for _, name := range categoryOrder {
		categories = append(categories, checkCategory{
			name:   name,
			checks: categoryMap[name],
		})
	}

	return categories
}

// RunCategory executes all checks in a specific category.
func (r *CheckRegistry) RunCategory(category string, fix bool) []checkResult {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make([]checkResult, 0)
	for _, check := range r.checks {
		if check.Category() == category {
			results = append(results, check.Run(fix))
		}
	}
	return results
}

// GetCategories returns a sorted list of all registered categories.
func (r *CheckRegistry) GetCategories() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	categorySet := make(map[string]bool)
	for _, check := range r.checks {
		categorySet[check.Category()] = true
	}

	categories := make([]string, 0, len(categorySet))
	for cat := range categorySet {
		categories = append(categories, cat)
	}
	sort.Strings(categories)
	return categories
}

// Count returns the total number of registered checks.
func (r *CheckRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.checks)
}

// DefaultRegistry is the global check registry used by the doctor command.
// Checks should register themselves here during package initialization.
var DefaultRegistry = NewCheckRegistry()

// simpleCheck provides a wrapper to convert function-based checks to the Check interface.
// This allows gradual migration of existing check functions to the registry pattern.
type simpleCheck struct {
	name     string
	category string
	runFunc  func(fix bool) checkResult
}

func (s *simpleCheck) Name() string {
	return s.name
}

func (s *simpleCheck) Category() string {
	return s.category
}

func (s *simpleCheck) Run(fix bool) checkResult {
	return s.runFunc(fix)
}

// RegisterSimpleCheck is a convenience function to register a check from a simple function.
// This helps bridge existing check functions to the registry pattern without full refactoring.
//
// Example:
//
//	RegisterSimpleCheck("ox-in-path", "Ecosystem", checkOxInPath)
func RegisterSimpleCheck(name, category string, runFunc func(fix bool) checkResult) {
	DefaultRegistry.Register(&simpleCheck{
		name:     name,
		category: category,
		runFunc:  runFunc,
	})
}

// conditionalCheck wraps a check with a condition function.
// The check only runs if the condition returns true.
type conditionalCheck struct {
	check     Check
	condition func() bool
}

func (c *conditionalCheck) Name() string {
	return c.check.Name()
}

func (c *conditionalCheck) Category() string {
	return c.check.Category()
}

func (c *conditionalCheck) Run(fix bool) checkResult {
	if !c.condition() {
		return SkippedCheck(c.check.Name(), "condition not met", "")
	}
	return c.check.Run(fix)
}

// RegisterConditionalCheck registers a check that only runs if the condition is met.
// Useful for checks that depend on external tools being present.
//
// Example:
//
//	RegisterConditionalCheck(claudeCodeCheck, detectClaudeCode)
func RegisterConditionalCheck(check Check, condition func() bool) {
	DefaultRegistry.Register(&conditionalCheck{
		check:     check,
		condition: condition,
	})
}
