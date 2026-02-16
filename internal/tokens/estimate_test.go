package tokens

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantMin int
		wantMax int
	}{
		{
			name:    "empty string",
			text:    "",
			wantMin: 0,
			wantMax: 0,
		},
		{
			name:    "short text",
			text:    "Hello, world!",
			wantMin: 3,
			wantMax: 4,
		},
		{
			name:    "typical sentence",
			text:    "This is a test sentence with multiple words.",
			wantMin: 10,
			wantMax: 12,
		},
		{
			name:    "200 characters",
			text:    "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris.",
			wantMin: 48,
			wantMax: 52,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.text)
			assert.GreaterOrEqual(t, got, tt.wantMin, "EstimateTokens() = %v, want >= %v", got, tt.wantMin)
			assert.LessOrEqual(t, got, tt.wantMax, "EstimateTokens() = %v, want <= %v", got, tt.wantMax)
		})
	}
}

func TestEstimateTokensWithChildren(t *testing.T) {
	tests := []struct {
		name              string
		content           string
		childDescriptions map[string]string
		wantMin           int
		wantMax           int
	}{
		{
			name:              "no children",
			content:           "Main content here",
			childDescriptions: map[string]string{},
			wantMin:           4,
			wantMax:           5,
		},
		{
			name:    "with children",
			content: "Main content here",
			childDescriptions: map[string]string{
				"child1": "First child description",
				"child2": "Second child description",
			},
			wantMin: 15,
			wantMax: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokensWithChildren(tt.content, tt.childDescriptions)
			assert.GreaterOrEqual(t, got, tt.wantMin, "EstimateTokensWithChildren() = %v, want >= %v", got, tt.wantMin)
			assert.LessOrEqual(t, got, tt.wantMax, "EstimateTokensWithChildren() = %v, want <= %v", got, tt.wantMax)
		})
	}
}
