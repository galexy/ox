package uxfriction

// Suggester provides command/flag suggestions for CLI errors.
type Suggester interface {
	// Suggest returns a correction for the given input.
	// Returns nil if no suggestion available.
	Suggest(input string, ctx SuggestContext) *Suggestion
}
