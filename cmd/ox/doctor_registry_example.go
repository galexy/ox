package main

// This file demonstrates how to migrate existing doctor checks to the registry pattern.
// It is not used in the actual doctor command yet, but serves as documentation
// for future refactoring work.

// Example 1: Converting a simple check function to use the registry
//
// Before (current pattern):
//
//	func checkOxInPath() checkResult {
//	    // ... implementation
//	}
//
// After (registry pattern):
//
//	type oxInPathCheck struct{}
//
//	func (o *oxInPathCheck) Name() string     { return "ox in PATH" }
//	func (o *oxInPathCheck) Category() string { return "Ecosystem" }
//	func (o *oxInPathCheck) Run(fix bool) checkResult {
//	    // ... same implementation as before
//	}
//
//	func init() {
//	    DefaultRegistry.Register(&oxInPathCheck{})
//	}

// Example 2: Using RegisterSimpleCheck for gradual migration
//
// This allows using the registry without full refactoring:
//
//	func init() {
//	    RegisterSimpleCheck("ox-in-path", "Ecosystem", checkOxInPath)
//	}

// Example 3: Conditional checks (only run when tool is detected)
//
// Before (current pattern in runDoctorChecks):
//
//	if detectClaudeCode() {
//	    integrationChecks = append(integrationChecks, checkClaudeCodeHooks(fix))
//	}
//
// After (registry pattern):
//
//	type claudeCodeHooksCheck struct{}
//
//	func (c *claudeCodeHooksCheck) Name() string     { return "Claude Code hooks" }
//	func (c *claudeCodeHooksCheck) Category() string { return "Integration" }
//	func (c *claudeCodeHooksCheck) Run(fix bool) checkResult {
//	    return checkClaudeCodeHooks(fix)
//	}
//
//	func init() {
//	    RegisterConditionalCheck(&claudeCodeHooksCheck{}, detectClaudeCode)
//	}

// Example 4: Using the registry in the doctor command
//
// Before (current pattern in runDoctorChecks):
//
//	func runDoctorChecks(fix bool) []checkCategory {
//	    var categories []checkCategory
//	    categories = append(categories, checkCategory{
//	        name: "Ecosystem",
//	        checks: []checkResult{
//	            checkOxInPath(),
//	            checkForUpdates(),
//	        },
//	    })
//	    return categories
//	}
//
// After (using registry):
//
//	func runDoctorChecks(fix bool) []checkCategory {
//	    return DefaultRegistry.RunAll(fix)
//	}

// Example 5: Checks with child results
//
// Some checks like checkConfigFile have nested children. These can be handled
// by having the Run() method populate the children field:
//
//	type configCheck struct{}
//
//	func (c *configCheck) Name() string     { return "config.json" }
//	func (c *configCheck) Category() string { return "Project Structure" }
//	func (c *configCheck) Run(fix bool) checkResult {
//	    result := checkConfigFile(fix)
//	    if result.passed && !result.skipped {
//	        result.children = []checkResult{checkConfigFields(fix)}
//	    }
//	    return result
//	}

// Benefits of the registry pattern:
//
// 1. Centralized check management - no need to manually organize checks by category
// 2. Easier testing - checks can be tested in isolation
// 3. Extensibility - new checks can be added without modifying runDoctorChecks
// 4. Reusability - checks can be run individually or by category
// 5. Thread safety - concurrent check execution is built-in
// 6. Gradual migration - RegisterSimpleCheck allows incremental refactoring
//
// Migration strategy:
//
// Phase 1: Create registry system (DONE)
// Phase 2: Migrate simple checks using RegisterSimpleCheck
// Phase 3: Convert conditional checks to use RegisterConditionalCheck
// Phase 4: Refactor complex checks to implement Check interface
// Phase 5: Update runDoctorChecks to use DefaultRegistry.RunAll()
// Phase 6: Clean up old check execution code
