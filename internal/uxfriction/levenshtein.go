package uxfriction

// LevenshteinDistance calculates the edit distance between two strings
// using dynamic programming. Returns the minimum number of single-character
// edits (insertions, deletions, substitutions) needed to transform a into b.
func LevenshteinDistance(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// convert to runes for proper Unicode handling
	runesA := []rune(a)
	runesB := []rune(b)
	lenA := len(runesA)
	lenB := len(runesB)

	// use single row optimization to reduce memory from O(m*n) to O(min(m,n))
	// ensure b is the shorter string for memory efficiency
	if lenA < lenB {
		runesA, runesB = runesB, runesA
		lenA, lenB = lenB, lenA
	}

	// previous and current row of distances
	prev := make([]int, lenB+1)
	curr := make([]int, lenB+1)

	// initialize first row
	for j := 0; j <= lenB; j++ {
		prev[j] = j
	}

	// fill in the rest of the matrix
	for i := 1; i <= lenA; i++ {
		curr[0] = i
		for j := 1; j <= lenB; j++ {
			cost := 1
			if runesA[i-1] == runesB[j-1] {
				cost = 0
			}
			// minimum of deletion, insertion, or substitution
			curr[j] = min(
				prev[j]+1,      // deletion
				curr[j-1]+1,    // insertion
				prev[j-1]+cost, // substitution
			)
		}
		prev, curr = curr, prev
	}

	return prev[lenB]
}

// min returns the minimum of three integers.
func min(a, b, c int) int {
	if a <= b && a <= c {
		return a
	}
	if b <= c {
		return b
	}
	return c
}

// LevenshteinSuggester suggests corrections based on edit distance.
type LevenshteinSuggester struct {
	maxDistance int
}

// NewLevenshteinSuggester creates a suggester that only suggests matches
// within maxDist edits.
func NewLevenshteinSuggester(maxDist int) *LevenshteinSuggester {
	return &LevenshteinSuggester{
		maxDistance: maxDist,
	}
}

// Suggest finds the closest match to input from ctx.ValidOptions.
// Returns nil if no match is within maxDistance or if ValidOptions is empty.
func (s *LevenshteinSuggester) Suggest(input string, ctx SuggestContext) *Suggestion {
	if len(ctx.ValidOptions) == 0 {
		return nil
	}

	bestMatch := ""
	bestDistance := s.maxDistance + 1

	for _, option := range ctx.ValidOptions {
		dist := LevenshteinDistance(input, option)
		if dist < bestDistance {
			bestDistance = dist
			bestMatch = option
		}
	}

	// no match within threshold
	if bestDistance > s.maxDistance {
		return nil
	}

	// confidence decreases with distance: 1.0 for exact match, approaches 0 as distance increases
	confidence := 1.0 - float64(bestDistance)/float64(s.maxDistance+1)

	return &Suggestion{
		Type:       SuggestionLevenshtein,
		Original:   input,
		Corrected:  bestMatch,
		Confidence: confidence,
	}
}
