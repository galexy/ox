package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCheck implements the Check interface for testing
type mockCheck struct {
	name     string
	category string
	runFunc  func(fix bool) checkResult
}

func (m *mockCheck) Name() string {
	return m.name
}

func (m *mockCheck) Category() string {
	return m.category
}

func (m *mockCheck) Run(fix bool) checkResult {
	if m.runFunc != nil {
		return m.runFunc(fix)
	}
	return PassedCheck(m.name, "ok")
}

func TestCheckRegistry_Register(t *testing.T) {
	registry := NewCheckRegistry()

	check1 := &mockCheck{name: "test1", category: "cat1"}
	check2 := &mockCheck{name: "test2", category: "cat2"}

	registry.Register(check1)
	registry.Register(check2)

	assert.Equal(t, 2, registry.Count(), "expected 2 checks")
}

func TestCheckRegistry_RunAll(t *testing.T) {
	registry := NewCheckRegistry()

	check1 := &mockCheck{
		name:     "check1",
		category: "Category A",
		runFunc: func(fix bool) checkResult {
			return PassedCheck("check1", "success")
		},
	}
	check2 := &mockCheck{
		name:     "check2",
		category: "Category B",
		runFunc: func(fix bool) checkResult {
			return FailedCheck("check2", "failed", "detail")
		},
	}
	check3 := &mockCheck{
		name:     "check3",
		category: "Category A",
		runFunc: func(fix bool) checkResult {
			return WarningCheck("check3", "warning", "detail")
		},
	}

	registry.Register(check1)
	registry.Register(check2)
	registry.Register(check3)

	categories := registry.RunAll(false)

	assert.Equal(t, 2, len(categories), "expected 2 categories")

	// verify category A has 2 checks
	var catA *checkCategory
	for i := range categories {
		if categories[i].name == "Category A" {
			catA = &categories[i]
			break
		}
	}
	require.NotNil(t, catA, "Category A not found")
	assert.Equal(t, 2, len(catA.checks), "expected 2 checks in Category A")
}

func TestCheckRegistry_RunCategory(t *testing.T) {
	registry := NewCheckRegistry()

	check1 := &mockCheck{name: "check1", category: "cat1"}
	check2 := &mockCheck{name: "check2", category: "cat2"}
	check3 := &mockCheck{name: "check3", category: "cat1"}

	registry.Register(check1)
	registry.Register(check2)
	registry.Register(check3)

	results := registry.RunCategory("cat1", false)

	assert.Equal(t, 2, len(results), "expected 2 results for cat1")
}

func TestCheckRegistry_GetCategories(t *testing.T) {
	registry := NewCheckRegistry()

	registry.Register(&mockCheck{name: "c1", category: "Zebra"})
	registry.Register(&mockCheck{name: "c2", category: "Alpha"})
	registry.Register(&mockCheck{name: "c3", category: "Beta"})
	registry.Register(&mockCheck{name: "c4", category: "Alpha"}) // duplicate category

	categories := registry.GetCategories()

	assert.Equal(t, 3, len(categories), "expected 3 unique categories")

	// verify sorted order
	expected := []string{"Alpha", "Beta", "Zebra"}
	for i, cat := range categories {
		assert.Equal(t, expected[i], cat, "position %d mismatch", i)
	}
}

func TestRegisterSimpleCheck(t *testing.T) {
	registry := NewCheckRegistry()

	called := false
	checkFunc := func(fix bool) checkResult {
		called = true
		return PassedCheck("simple", "ok")
	}

	simpleCheck := &simpleCheck{
		name:     "simple",
		category: "test",
		runFunc:  checkFunc,
	}
	registry.Register(simpleCheck)

	results := registry.RunCategory("test", false)

	require.Equal(t, 1, len(results), "expected 1 result")

	assert.True(t, called, "check function was not called")
	assert.Equal(t, "simple", results[0].name, "expected check name 'simple'")
}

func TestConditionalCheck(t *testing.T) {
	registry := NewCheckRegistry()

	conditionMet := false
	check := &mockCheck{
		name:     "conditional",
		category: "test",
		runFunc: func(fix bool) checkResult {
			return PassedCheck("conditional", "executed")
		},
	}

	conditional := &conditionalCheck{
		check: check,
		condition: func() bool {
			return conditionMet
		},
	}
	registry.Register(conditional)

	// first run with condition not met
	results := registry.RunCategory("test", false)
	require.Equal(t, 1, len(results), "expected 1 result")
	assert.True(t, results[0].skipped, "expected check to be skipped when condition not met")

	// second run with condition met
	conditionMet = true
	results = registry.RunCategory("test", false)
	require.Equal(t, 1, len(results), "expected 1 result")
	assert.False(t, results[0].skipped, "expected check to run when condition is met")
	assert.True(t, results[0].passed, "expected check to pass when condition is met")
}

func TestCheckRegistry_FixParameter(t *testing.T) {
	registry := NewCheckRegistry()

	fixCalled := false
	check := &mockCheck{
		name:     "fixable",
		category: "test",
		runFunc: func(fix bool) checkResult {
			if fix {
				fixCalled = true
				return PassedCheck("fixable", "fixed")
			}
			return FailedCheck("fixable", "needs fix", "run with --fix")
		},
	}
	registry.Register(check)

	// run without fix
	results := registry.RunCategory("test", false)
	assert.False(t, results[0].passed, "expected check to fail without fix")
	assert.False(t, fixCalled, "fix should not be called when fix=false")

	// run with fix
	results = registry.RunCategory("test", true)
	assert.True(t, results[0].passed, "expected check to pass with fix")
	assert.True(t, fixCalled, "fix should be called when fix=true")
}

func TestCheckRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewCheckRegistry()

	// test concurrent registration
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			check := &mockCheck{
				name:     string(rune('a' + n)),
				category: "concurrent",
			}
			registry.Register(check)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	assert.Equal(t, 10, registry.Count(), "expected 10 checks after concurrent registration")

	// test concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			_ = registry.GetCategories()
			_ = registry.RunAll(false)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
