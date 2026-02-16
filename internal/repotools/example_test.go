package repotools_test

import (
	"fmt"

	"github.com/sageox/ox/internal/repotools"
)

func ExampleGenerateRepoID() {
	id := repotools.GenerateRepoID()

	// validate the ID
	valid := repotools.IsValidRepoID(id)
	fmt.Printf("Valid: %v\n", valid)

	// parse back to UUID
	uuid, err := repotools.ParseRepoID(id)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("UUID version: %d\n", uuid.Version())
	// Output:
	// Valid: true
	// UUID version: 7
}

func ExampleParseRepoID() {
	// generate a repo ID
	id := repotools.GenerateRepoID()

	// parse it back to UUID
	uuid, err := repotools.ParseRepoID(id)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Parsed UUID version: %d\n", uuid.Version())
	// Output:
	// Parsed UUID version: 7
}

func ExampleIsValidRepoID() {
	// valid ID
	validID := repotools.GenerateRepoID()
	fmt.Printf("Valid ID: %v\n", repotools.IsValidRepoID(validID))

	// invalid IDs
	fmt.Printf("Invalid prefix: %v\n", repotools.IsValidRepoID("invalid_abc123"))
	fmt.Printf("Empty: %v\n", repotools.IsValidRepoID(""))
	fmt.Printf("No prefix: %v\n", repotools.IsValidRepoID("abc123"))

	// Output:
	// Valid ID: true
	// Invalid prefix: false
	// Empty: false
	// No prefix: false
}
