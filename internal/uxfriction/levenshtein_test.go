//nolint:misspell // intentional misspellings in test data for Levenshtein distance testing
package uxfriction

import (
	"math"
	"testing"
)

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		// same strings = 0
		{name: "identical empty strings", a: "", b: "", expected: 0},
		{name: "identical single char", a: "a", b: "a", expected: 0},
		{name: "identical word", a: "hello", b: "hello", expected: 0},
		{name: "identical long string", a: "levenshtein", b: "levenshtein", expected: 0},

		// empty string cases
		{name: "empty to single char", a: "", b: "a", expected: 1},
		{name: "single char to empty", a: "a", b: "", expected: 1},
		{name: "empty to word", a: "", b: "hello", expected: 5},
		{name: "word to empty", a: "world", b: "", expected: 5},

		// single character differences
		{name: "single substitution at start", a: "cat", b: "bat", expected: 1},
		{name: "single substitution at end", a: "cat", b: "car", expected: 1},
		{name: "single substitution in middle", a: "cat", b: "cot", expected: 1},
		{name: "single insertion", a: "cat", b: "cats", expected: 1},
		{name: "single deletion", a: "cats", b: "cat", expected: 1},
		{name: "single insertion at start", a: "at", b: "cat", expected: 1},
		{name: "single deletion at start", a: "cat", b: "at", expected: 1},

		// transpositions (adjacent swaps - costs 2 in standard Levenshtein)
		{name: "adjacent transposition ab->ba", a: "ab", b: "ba", expected: 2},
		{name: "transposition in word", a: "cat", b: "act", expected: 2},
		{name: "transposition teh->the", a: "teh", b: "the", expected: 2},

		// multiple edits
		{name: "two substitutions", a: "cat", b: "dog", expected: 3},
		{name: "kitten to sitting", a: "kitten", b: "sitting", expected: 3},
		{name: "saturday to sunday", a: "saturday", b: "sunday", expected: 3},
		{name: "flaw to lawn", a: "flaw", b: "lawn", expected: 2},

		// CLI typos (realistic use cases)
		{name: "typo: stauts -> status", a: "stauts", b: "status", expected: 2},
		{name: "typo: hepl -> help", a: "hepl", b: "help", expected: 2},
		{name: "typo: agnet -> agent", a: "agnet", b: "agent", expected: 2},
		{name: "typo: cofnig -> config", a: "cofnig", b: "config", expected: 2},
		{name: "typo: inital -> initial", a: "inital", b: "initial", expected: 1},
		{name: "typo: transcirpt -> transcript", a: "transcirpt", b: "transcript", expected: 2},

		// unicode support
		{name: "unicode identical", a: "日本語", b: "日本語", expected: 0},
		{name: "unicode single substitution", a: "日本語", b: "日本人", expected: 1},
		{name: "unicode different length", a: "日本", b: "日本語", expected: 1},
		{name: "unicode to ascii", a: "cafe", b: "café", expected: 1},
		{name: "emoji identical", a: "hello\U0001F600", b: "hello\U0001F600", expected: 0},
		{name: "emoji substitution", a: "hi\U0001F600", b: "hi\U0001F601", expected: 1},
		{name: "mixed unicode", a: "hello世界", b: "hello世界!", expected: 1},

		// case sensitivity
		{name: "case difference single", a: "A", b: "a", expected: 1},
		{name: "case difference word", a: "Hello", b: "hello", expected: 1},
		{name: "case difference all caps", a: "HELLO", b: "hello", expected: 5},

		// edge cases with repeated chars
		{name: "repeated to single", a: "aaa", b: "a", expected: 2},
		{name: "single to repeated", a: "a", b: "aaa", expected: 2},
		{name: "different repeated", a: "aaa", b: "bbb", expected: 3},

		// completely different
		{name: "completely different same length", a: "abc", b: "xyz", expected: 3},
		{name: "completely different diff length", a: "abc", b: "wxyz", expected: 4},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := LevenshteinDistance(tc.a, tc.b)
			if got != tc.expected {
				t.Errorf("LevenshteinDistance(%q, %q) = %d; want %d", tc.a, tc.b, got, tc.expected)
			}
		})
	}
}

func TestLevenshteinDistance_Symmetry(t *testing.T) {
	// property: distance(a, b) == distance(b, a)
	pairs := [][2]string{
		{"", "abc"},
		{"hello", "world"},
		{"kitten", "sitting"},
		{"日本語", "中文"},
		{"cafe", "café"},
		{"agent", "agnet"},
	}

	for _, pair := range pairs {
		a, b := pair[0], pair[1]
		t.Run(a+"_"+b, func(t *testing.T) {
			distAB := LevenshteinDistance(a, b)
			distBA := LevenshteinDistance(b, a)
			if distAB != distBA {
				t.Errorf("symmetry violation: distance(%q, %q)=%d != distance(%q, %q)=%d",
					a, b, distAB, b, a, distBA)
			}
		})
	}
}

func TestLevenshteinDistance_TriangleInequality(t *testing.T) {
	// property: distance(a, c) <= distance(a, b) + distance(b, c)
	triples := [][3]string{
		{"cat", "bat", "bar"},
		{"hello", "hallo", "halli"},
		{"", "a", "ab"},
		{"kitten", "mitten", "sitting"},
		{"agent", "agnet", "agent"},
	}

	for _, triple := range triples {
		a, b, c := triple[0], triple[1], triple[2]
		t.Run(a+"_"+b+"_"+c, func(t *testing.T) {
			distAC := LevenshteinDistance(a, c)
			distAB := LevenshteinDistance(a, b)
			distBC := LevenshteinDistance(b, c)
			if distAC > distAB+distBC {
				t.Errorf("triangle inequality violation: distance(%q, %q)=%d > distance(%q, %q)=%d + distance(%q, %q)=%d",
					a, c, distAC, a, b, distAB, b, c, distBC)
			}
		})
	}
}

func TestLevenshteinDistance_IdentityOfIndiscernibles(t *testing.T) {
	// property: distance(a, b) == 0 iff a == b
	strings := []string{
		"", "a", "hello", "日本語", "cafe", "café",
	}

	for _, s := range strings {
		t.Run("self_"+s, func(t *testing.T) {
			dist := LevenshteinDistance(s, s)
			if dist != 0 {
				t.Errorf("identity violation: distance(%q, %q) = %d; want 0", s, s, dist)
			}
		})
	}

	// also verify non-identical strings have distance > 0
	pairs := [][2]string{
		{"a", "b"},
		{"hello", "hallo"},
		{"", "x"},
	}
	for _, pair := range pairs {
		a, b := pair[0], pair[1]
		t.Run("non_self_"+a+"_"+b, func(t *testing.T) {
			dist := LevenshteinDistance(a, b)
			if dist == 0 {
				t.Errorf("identity violation: distance(%q, %q) = 0 but strings are different", a, b)
			}
		})
	}
}

func TestLevenshteinDistance_BoundedByMaxLength(t *testing.T) {
	// property: distance(a, b) <= max(len(a), len(b))
	pairs := [][2]string{
		{"", "abcdef"},
		{"abc", "xyz"},
		{"hello", "world"},
		{"a", "abcdefghij"},
		{"日本", "abc"},
	}

	for _, pair := range pairs {
		a, b := pair[0], pair[1]
		t.Run(a+"_"+b, func(t *testing.T) {
			dist := LevenshteinDistance(a, b)
			maxLen := len([]rune(a))
			if len([]rune(b)) > maxLen {
				maxLen = len([]rune(b))
			}
			if dist > maxLen {
				t.Errorf("bound violation: distance(%q, %q)=%d > max_rune_length=%d",
					a, b, dist, maxLen)
			}
		})
	}
}

func TestLevenshteinSuggester_Suggest(t *testing.T) {
	tests := []struct {
		name           string
		maxDistance    int
		input          string
		validOptions   []string
		wantNil        bool
		wantCorrected  string
		wantConfidence float64 // approximate, using delta comparison
	}{
		// basic suggestions
		{
			name:           "exact match",
			maxDistance:    2,
			input:          "status",
			validOptions:   []string{"status", "help", "version"},
			wantNil:        false,
			wantCorrected:  "status",
			wantConfidence: 1.0,
		},
		{
			name:           "single typo within distance",
			maxDistance:    2,
			input:          "statis",
			validOptions:   []string{"status", "help", "version"},
			wantNil:        false,
			wantCorrected:  "status",
			wantConfidence: 0.666, // 1.0 - 1/3
		},
		{
			name:           "two char typo within distance",
			maxDistance:    2,
			input:          "stauts",
			validOptions:   []string{"status", "help", "version"},
			wantNil:        false,
			wantCorrected:  "status",
			wantConfidence: 0.333, // 1.0 - 2/3
		},
		{
			name:           "typo beyond max distance",
			maxDistance:    1,
			input:          "stauts",
			validOptions:   []string{"status", "help", "version"},
			wantNil:        true,
			wantCorrected:  "",
			wantConfidence: 0,
		},

		// empty options
		{
			name:           "empty valid options",
			maxDistance:    2,
			input:          "anything",
			validOptions:   []string{},
			wantNil:        true,
			wantCorrected:  "",
			wantConfidence: 0,
		},
		{
			name:           "nil valid options",
			maxDistance:    2,
			input:          "anything",
			validOptions:   nil,
			wantNil:        true,
			wantCorrected:  "",
			wantConfidence: 0,
		},

		// picks closest match
		{
			name:           "picks closest among multiple",
			maxDistance:    3,
			input:          "agnet",
			validOptions:   []string{"agent", "init", "config", "agents"},
			wantNil:        false,
			wantCorrected:  "agent", // distance 2, closer than "agents" at 3
			wantConfidence: 0.5,     // 1.0 - 2/4
		},
		{
			name:           "first match wins on tie",
			maxDistance:    2,
			input:          "ab",
			validOptions:   []string{"ac", "ad", "ae"}, // all distance 1
			wantNil:        false,
			wantCorrected:  "ac", // first one found
			wantConfidence: 0.666,
		},

		// CLI realistic typos
		{
			name:           "help typo",
			maxDistance:    2,
			input:          "hepl",
			validOptions:   []string{"help", "version", "status"},
			wantNil:        false,
			wantCorrected:  "help",
			wantConfidence: 0.333,
		},
		{
			name:           "config typo",
			maxDistance:    3,
			input:          "cofnig",
			validOptions:   []string{"config", "init", "status"},
			wantNil:        false,
			wantCorrected:  "config",
			wantConfidence: 0.5, // distance 2, 1.0 - 2/4
		},
		{
			name:           "transcript typo",
			maxDistance:    3,
			input:          "transcirpt",
			validOptions:   []string{"transcript", "transfer", "translate"},
			wantNil:        false,
			wantCorrected:  "transcript",
			wantConfidence: 0.5, // distance 2
		},

		// no match within threshold
		{
			name:           "completely different word",
			maxDistance:    2,
			input:          "xyz",
			validOptions:   []string{"alpha", "beta", "gamma"},
			wantNil:        true,
			wantCorrected:  "",
			wantConfidence: 0,
		},

		// unicode handling
		{
			name:           "unicode option match",
			maxDistance:    2,
			input:          "日本",
			validOptions:   []string{"日本語", "中文", "english"},
			wantNil:        false,
			wantCorrected:  "日本語", // distance 1
			wantConfidence: 0.666,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			suggester := NewLevenshteinSuggester(tc.maxDistance)
			ctx := SuggestContext{
				ValidOptions: tc.validOptions,
			}

			suggestion := suggester.Suggest(tc.input, ctx)

			if tc.wantNil {
				if suggestion != nil {
					t.Errorf("expected nil suggestion, got %+v", suggestion)
				}
				return
			}

			if suggestion == nil {
				t.Fatal("expected non-nil suggestion, got nil")
			}

			if suggestion.Type != SuggestionLevenshtein {
				t.Errorf("Type = %q; want %q", suggestion.Type, SuggestionLevenshtein)
			}

			if suggestion.Original != tc.input {
				t.Errorf("Original = %q; want %q", suggestion.Original, tc.input)
			}

			if suggestion.Corrected != tc.wantCorrected {
				t.Errorf("Corrected = %q; want %q", suggestion.Corrected, tc.wantCorrected)
			}

			// confidence comparison with delta for floating point
			delta := 0.01
			if math.Abs(suggestion.Confidence-tc.wantConfidence) > delta {
				t.Errorf("Confidence = %f; want %f (delta %f)", suggestion.Confidence, tc.wantConfidence, delta)
			}
		})
	}
}

func TestLevenshteinSuggester_ConfidenceFormula(t *testing.T) {
	// verify confidence formula: 1.0 - distance/(maxDistance+1)
	tests := []struct {
		maxDistance int
		distance    int
		expected    float64
	}{
		{maxDistance: 2, distance: 0, expected: 1.0},      // exact match
		{maxDistance: 2, distance: 1, expected: 0.666667}, // 1 - 1/3
		{maxDistance: 2, distance: 2, expected: 0.333333}, // 1 - 2/3
		{maxDistance: 3, distance: 0, expected: 1.0},
		{maxDistance: 3, distance: 1, expected: 0.75}, // 1 - 1/4
		{maxDistance: 3, distance: 2, expected: 0.5},  // 1 - 2/4
		{maxDistance: 3, distance: 3, expected: 0.25}, // 1 - 3/4
		{maxDistance: 1, distance: 0, expected: 1.0},
		{maxDistance: 1, distance: 1, expected: 0.5}, // 1 - 1/2
	}

	for _, tc := range tests {
		// create input strings with exactly tc.distance edits apart
		// for simplicity, use strings where we know the distance
		var input, option string
		switch tc.distance {
		case 0:
			input, option = "test", "test"
		case 1:
			input, option = "test", "tests"
		case 2:
			input, option = "test", "toast"
		case 3:
			input, option = "test", "toast!"
		}

		suggester := NewLevenshteinSuggester(tc.maxDistance)
		ctx := SuggestContext{ValidOptions: []string{option}}
		suggestion := suggester.Suggest(input, ctx)

		if suggestion == nil {
			if tc.distance <= tc.maxDistance {
				t.Errorf("maxDistance=%d, distance=%d: expected suggestion, got nil",
					tc.maxDistance, tc.distance)
			}
			continue
		}

		// verify actual distance matches expected
		actualDist := LevenshteinDistance(input, option)
		if actualDist != tc.distance {
			t.Errorf("distance mismatch: got %d, expected %d for %q -> %q",
				actualDist, tc.distance, input, option)
			continue
		}

		delta := 0.001
		if math.Abs(suggestion.Confidence-tc.expected) > delta {
			t.Errorf("maxDistance=%d, distance=%d: Confidence = %f; want %f",
				tc.maxDistance, tc.distance, suggestion.Confidence, tc.expected)
		}
	}
}

func TestNewLevenshteinSuggester(t *testing.T) {
	tests := []struct {
		maxDist int
	}{
		{0},
		{1},
		{2},
		{3},
		{10},
	}

	for _, tc := range tests {
		suggester := NewLevenshteinSuggester(tc.maxDist)
		if suggester == nil {
			t.Errorf("NewLevenshteinSuggester(%d) returned nil", tc.maxDist)
			continue
		}
		if suggester.maxDistance != tc.maxDist {
			t.Errorf("maxDistance = %d; want %d", suggester.maxDistance, tc.maxDist)
		}
	}
}

func TestLevenshteinSuggester_ZeroMaxDistance(t *testing.T) {
	// with maxDistance=0, only exact matches should return suggestions
	suggester := NewLevenshteinSuggester(0)
	ctx := SuggestContext{ValidOptions: []string{"exact", "match"}}

	// exact match should work
	suggestion := suggester.Suggest("exact", ctx)
	if suggestion == nil {
		t.Error("expected suggestion for exact match with maxDistance=0")
	} else if suggestion.Corrected != "exact" {
		t.Errorf("Corrected = %q; want %q", suggestion.Corrected, "exact")
	} else if suggestion.Confidence != 1.0 {
		t.Errorf("Confidence = %f; want 1.0", suggestion.Confidence)
	}

	// single edit should not match
	suggestion = suggester.Suggest("exac", ctx)
	if suggestion != nil {
		t.Errorf("expected nil for 1-edit distance with maxDistance=0, got %+v", suggestion)
	}
}

// benchmarks

func BenchmarkLevenshteinDistance_Short(b *testing.B) {
	for i := 0; i < b.N; i++ {
		LevenshteinDistance("kitten", "sitting")
	}
}

func BenchmarkLevenshteinDistance_Medium(b *testing.B) {
	a := "levenshtein"
	c := "frankenstein"
	for i := 0; i < b.N; i++ {
		LevenshteinDistance(a, c)
	}
}

func BenchmarkLevenshteinDistance_Long(b *testing.B) {
	a := "the quick brown fox jumps over the lazy dog"
	c := "the lazy dog jumps over the quick brown fox"
	for i := 0; i < b.N; i++ {
		LevenshteinDistance(a, c)
	}
}

func BenchmarkLevenshteinDistance_Unicode(b *testing.B) {
	a := "日本語テキスト"
	c := "中文字テスト"
	for i := 0; i < b.N; i++ {
		LevenshteinDistance(a, c)
	}
}

func BenchmarkLevenshteinSuggester_SmallOptions(b *testing.B) {
	suggester := NewLevenshteinSuggester(2)
	ctx := SuggestContext{
		ValidOptions: []string{"status", "help", "version", "agent", "config"},
	}
	for i := 0; i < b.N; i++ {
		suggester.Suggest("stauts", ctx)
	}
}

func BenchmarkLevenshteinSuggester_ManyOptions(b *testing.B) {
	suggester := NewLevenshteinSuggester(2)
	ctx := SuggestContext{
		ValidOptions: []string{
			"status", "help", "version", "agent", "config", "init", "doctor",
			"login", "logout", "transcript", "transfer", "translate", "update",
			"upgrade", "verify", "validate", "watch", "wait", "work", "write",
		},
	}
	for i := 0; i < b.N; i++ {
		suggester.Suggest("stauts", ctx)
	}
}
