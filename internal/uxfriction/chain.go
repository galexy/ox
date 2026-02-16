package uxfriction

// SuggestionEngine chains catalog and Levenshtein suggesters to provide
// the best available correction for CLI errors.
//
// The engine tries suggestions in priority order:
//  1. Full command remap from catalog (highest confidence)
//  2. Token-level catalog lookup
//  3. Levenshtein distance fallback (for typos)
//
// This chain ensures that curated catalog corrections take precedence over
// heuristic-based Levenshtein suggestions.
type SuggestionEngine struct {
	catalog     Catalog
	levenshtein *LevenshteinSuggester
}

// NewSuggestionEngine creates a SuggestionEngine with the given catalog
// and a default LevenshteinSuggester (maxDist: 2).
// The catalog may be nil, in which case only Levenshtein fallback is available.
func NewSuggestionEngine(catalog Catalog) *SuggestionEngine {
	return &SuggestionEngine{
		catalog:     catalog,
		levenshtein: NewLevenshteinSuggester(2),
	}
}

// SuggestForCommand attempts to find a suggestion for the given command.
// It tries suggestions in order of confidence:
//  1. Full command remap from catalog (if catalog is not nil)
//  2. Token-level catalog lookup (if ctx.BadToken is set)
//  3. Levenshtein fallback (if ctx.BadToken and ctx.ValidOptions are set)
//
// Returns nil if no suggestion is found.
//
// Use SuggestForCommandWithMapping if you need the original catalog mapping
// for auto-execute decisions.
func (e *SuggestionEngine) SuggestForCommand(fullCmd string, ctx SuggestContext) *Suggestion {
	suggestion, _ := e.SuggestForCommandWithMapping(fullCmd, ctx)
	return suggestion
}

// SuggestForCommandWithMapping attempts to find a suggestion and returns both
// the suggestion and the original catalog mapping (if from catalog).
//
// The mapping is needed for auto-execute decisions:
//   - Check mapping.AutoExecute to see if auto-execute is enabled
//   - Check suggestion.Confidence against AutoExecuteThreshold
//
// Returns (nil, nil) if no suggestion is found.
// Returns (suggestion, nil) for Levenshtein or token-fix suggestions (no mapping).
// Returns (suggestion, mapping) for command remap suggestions.
func (e *SuggestionEngine) SuggestForCommandWithMapping(fullCmd string, ctx SuggestContext) (*Suggestion, *CommandMapping) {
	// try full command remap from catalog first
	if e.catalog != nil {
		if mapping := e.catalog.LookupCommand(fullCmd); mapping != nil {
			// apply mapping to get corrected command (handles regex captures)
			corrected, ok := mapping.ApplyMapping(fullCmd)
			if !ok {
				corrected = mapping.Target
			}
			return &Suggestion{
				Type:        SuggestionCommandRemap,
				Original:    fullCmd,
				Corrected:   corrected,
				Confidence:  mapping.Confidence,
				Description: mapping.Description,
			}, mapping
		}
	}

	// try token-level catalog lookup (no mapping returned for token fixes)
	if e.catalog != nil && ctx.BadToken != "" {
		if mapping := e.catalog.LookupToken(ctx.BadToken, ctx.Kind); mapping != nil {
			return &Suggestion{
				Type:        SuggestionTokenFix,
				Original:    ctx.BadToken,
				Corrected:   mapping.Target,
				Confidence:  mapping.Confidence,
				Description: "",
			}, nil // no CommandMapping for token fixes
		}
	}

	// fall back to Levenshtein (no mapping for Levenshtein)
	// NOTE: Levenshtein primarily helps human users who make keystroke errors.
	// AI agents rarely produce typos since they generate commands programmatically.
	if ctx.BadToken != "" && len(ctx.ValidOptions) > 0 {
		if suggestion := e.levenshtein.Suggest(ctx.BadToken, ctx); suggestion != nil {
			// levenshtein.Suggest already sets SuggestionLevenshtein type
			return suggestion, nil
		}
	}

	return nil, nil
}
