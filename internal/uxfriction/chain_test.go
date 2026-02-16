package uxfriction

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockCatalog implements Catalog for testing.
type mockCatalog struct {
	commandMappings map[string]*CommandMapping
	tokenMappings   map[string]*TokenMapping
	version         string
}

func newMockCatalog() *mockCatalog {
	return &mockCatalog{
		commandMappings: make(map[string]*CommandMapping),
		tokenMappings:   make(map[string]*TokenMapping),
		version:         "test-v1",
	}
}

func (m *mockCatalog) LookupCommand(input string) *CommandMapping {
	return m.commandMappings[normalizeCommand(input)]
}

func (m *mockCatalog) LookupToken(token string, kind FailureKind) *TokenMapping {
	key := tokenKey(token, kind)
	return m.tokenMappings[key]
}

func (m *mockCatalog) Update(data CatalogData) error {
	return nil
}

func (m *mockCatalog) Version() string {
	return m.version
}

func (m *mockCatalog) addCommand(pattern, target string, confidence float64, desc string) {
	mapping := &CommandMapping{
		Pattern:     pattern,
		Target:      target,
		Confidence:  confidence,
		Description: desc,
	}
	m.commandMappings[normalizeCommand(pattern)] = mapping
}

func (m *mockCatalog) addToken(pattern, target string, kind FailureKind, confidence float64) {
	mapping := &TokenMapping{
		Pattern:    pattern,
		Target:     target,
		Kind:       kind,
		Confidence: confidence,
	}
	key := tokenKey(pattern, kind)
	m.tokenMappings[key] = mapping
}

func TestNewSuggestionEngine(t *testing.T) {
	tests := []struct {
		name    string
		catalog Catalog
	}{
		{
			name:    "creates engine with catalog",
			catalog: newMockCatalog(),
		},
		{
			name:    "creates engine with nil catalog",
			catalog: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewSuggestionEngine(tt.catalog)

			assert.NotNil(t, engine)
			assert.Equal(t, tt.catalog, engine.catalog)
			assert.NotNil(t, engine.levenshtein)
		})
	}
}

func TestSuggestForCommand_CatalogCommandRemap(t *testing.T) {
	tests := []struct {
		name     string
		fullCmd  string
		ctx      SuggestContext
		setupCat func(*mockCatalog)
		wantType SuggestionType
		wantOrig string
		wantCorr string
		wantConf float64
		wantDesc string
		wantNil  bool
	}{
		{
			name:    "returns catalog command remap when found",
			fullCmd: "ox agent lsit",
			ctx:     SuggestContext{},
			setupCat: func(c *mockCatalog) {
				c.addCommand("agent lsit", "agent list", 0.95, "typo correction")
			},
			wantType: SuggestionCommandRemap,
			wantOrig: "ox agent lsit",
			wantCorr: "agent list",
			wantConf: 0.95,
			wantDesc: "typo correction",
		},
		{
			name:    "returns catalog command remap with flags",
			fullCmd: "ox daemons show --every",
			ctx:     SuggestContext{},
			setupCat: func(c *mockCatalog) {
				c.addCommand("daemons show --every", "daemons show --all", 0.9, "flag remap")
			},
			wantType: SuggestionCommandRemap,
			wantOrig: "ox daemons show --every",
			wantCorr: "daemons show --all",
			wantConf: 0.9,
			wantDesc: "flag remap",
		},
		{
			name:    "normalizes command before lookup",
			fullCmd: "ox agent list --verbose -a",
			ctx:     SuggestContext{},
			setupCat: func(c *mockCatalog) {
				// catalog stores normalized form (flags sorted)
				c.addCommand("agent list -a --verbose", "agent list --all --verbose", 0.85, "flag expansion")
			},
			wantType: SuggestionCommandRemap,
			wantOrig: "ox agent list --verbose -a",
			wantCorr: "agent list --all --verbose",
			wantConf: 0.85,
			wantDesc: "flag expansion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := newMockCatalog()
			tt.setupCat(catalog)
			engine := NewSuggestionEngine(catalog)

			suggestion := engine.SuggestForCommand(tt.fullCmd, tt.ctx)

			if tt.wantNil {
				assert.Nil(t, suggestion)
				return
			}

			assert.NotNil(t, suggestion)
			assert.Equal(t, tt.wantType, suggestion.Type)
			assert.Equal(t, tt.wantOrig, suggestion.Original)
			assert.Equal(t, tt.wantCorr, suggestion.Corrected)
			assert.Equal(t, tt.wantConf, suggestion.Confidence)
			assert.Equal(t, tt.wantDesc, suggestion.Description)
		})
	}
}

func TestSuggestForCommand_TokenLookup(t *testing.T) {
	tests := []struct {
		name     string
		fullCmd  string
		ctx      SuggestContext
		setupCat func(*mockCatalog)
		wantType SuggestionType
		wantOrig string
		wantCorr string
		wantConf float64
	}{
		{
			name:    "falls back to token lookup when command not found",
			fullCmd: "ox agent stattus",
			ctx: SuggestContext{
				Kind:     FailureUnknownCommand,
				BadToken: "stattus",
			},
			setupCat: func(c *mockCatalog) {
				c.addToken("stattus", "status", FailureUnknownCommand, 0.88)
			},
			wantType: SuggestionTokenFix,
			wantOrig: "stattus",
			wantCorr: "status",
			wantConf: 0.88,
		},
		{
			name:    "token lookup respects failure kind",
			fullCmd: "ox agent list --verboes",
			ctx: SuggestContext{
				Kind:     FailureUnknownFlag,
				BadToken: "verboes",
			},
			setupCat: func(c *mockCatalog) {
				c.addToken("verboes", "verbose", FailureUnknownFlag, 0.92)
			},
			wantType: SuggestionTokenFix,
			wantOrig: "verboes",
			wantCorr: "verbose",
			wantConf: 0.92,
		},
		{
			name:    "token lookup case insensitive",
			fullCmd: "ox agent Depliy",
			ctx: SuggestContext{
				Kind:     FailureUnknownCommand,
				BadToken: "Depliy",
			},
			setupCat: func(c *mockCatalog) {
				c.addToken("depliy", "deploy", FailureUnknownCommand, 0.9)
			},
			wantType: SuggestionTokenFix,
			wantOrig: "Depliy",
			wantCorr: "deploy",
			wantConf: 0.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := newMockCatalog()
			tt.setupCat(catalog)
			engine := NewSuggestionEngine(catalog)

			suggestion := engine.SuggestForCommand(tt.fullCmd, tt.ctx)

			assert.NotNil(t, suggestion)
			assert.Equal(t, tt.wantType, suggestion.Type)
			assert.Equal(t, tt.wantOrig, suggestion.Original)
			assert.Equal(t, tt.wantCorr, suggestion.Corrected)
			assert.Equal(t, tt.wantConf, suggestion.Confidence)
		})
	}
}

func TestSuggestForCommand_LevenshteinFallback(t *testing.T) {
	tests := []struct {
		name     string
		fullCmd  string
		ctx      SuggestContext
		wantOrig string
		wantCorr string
		wantNil  bool
	}{
		{
			name:    "falls back to levenshtein when catalog has no match",
			fullCmd: "ox agent statis",
			ctx: SuggestContext{
				Kind:         FailureUnknownCommand,
				BadToken:     "statis",
				ValidOptions: []string{"status", "start", "stop", "list"},
			},
			wantOrig: "statis",
			wantCorr: "status",
		},
		{
			name:    "levenshtein finds closest match",
			fullCmd: "ox agent lst",
			ctx: SuggestContext{
				Kind:         FailureUnknownCommand,
				BadToken:     "lst",
				ValidOptions: []string{"list", "status", "stop"},
			},
			wantOrig: "lst",
			wantCorr: "list",
		},
		{
			name:    "levenshtein returns nil when distance too large",
			fullCmd: "ox agent xyz",
			ctx: SuggestContext{
				Kind:         FailureUnknownCommand,
				BadToken:     "xyz",
				ValidOptions: []string{"list", "status", "stop"},
			},
			wantNil: true,
		},
		{
			name:    "levenshtein returns nil when no valid options",
			fullCmd: "ox agent statis",
			ctx: SuggestContext{
				Kind:         FailureUnknownCommand,
				BadToken:     "statis",
				ValidOptions: []string{},
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// empty catalog so we fall through to levenshtein
			catalog := newMockCatalog()
			engine := NewSuggestionEngine(catalog)

			suggestion := engine.SuggestForCommand(tt.fullCmd, tt.ctx)

			if tt.wantNil {
				assert.Nil(t, suggestion)
				return
			}

			assert.NotNil(t, suggestion)
			assert.Equal(t, SuggestionLevenshtein, suggestion.Type)
			assert.Equal(t, tt.wantOrig, suggestion.Original)
			assert.Equal(t, tt.wantCorr, suggestion.Corrected)
			assert.Greater(t, suggestion.Confidence, 0.0)
		})
	}
}

func TestSuggestForCommand_ChainPriority(t *testing.T) {
	tests := []struct {
		name       string
		fullCmd    string
		ctx        SuggestContext
		setupCat   func(*mockCatalog)
		wantType   SuggestionType
		wantCorr   string
		wantReason string
	}{
		{
			name:    "command remap takes priority over token lookup",
			fullCmd: "ox agent stattus",
			ctx: SuggestContext{
				Kind:         FailureUnknownCommand,
				BadToken:     "stattus",
				ValidOptions: []string{"status", "list"},
			},
			setupCat: func(c *mockCatalog) {
				c.addCommand("agent stattus", "agent status --verbose", 0.95, "common pattern")
				c.addToken("stattus", "status", FailureUnknownCommand, 0.88)
			},
			wantType:   SuggestionCommandRemap,
			wantCorr:   "agent status --verbose",
			wantReason: "command remap has priority over token lookup",
		},
		{
			name:    "token lookup takes priority over levenshtein",
			fullCmd: "ox agent stattus",
			ctx: SuggestContext{
				Kind:         FailureUnknownCommand,
				BadToken:     "stattus",
				ValidOptions: []string{"status", "list"},
			},
			setupCat: func(c *mockCatalog) {
				// no command remap, but token mapping exists
				c.addToken("stattus", "state", FailureUnknownCommand, 0.88)
			},
			wantType:   SuggestionTokenFix,
			wantCorr:   "state",
			wantReason: "token lookup has priority over levenshtein (which would suggest 'status')",
		},
		{
			name:    "levenshtein used when no catalog matches",
			fullCmd: "ox agent stattus",
			ctx: SuggestContext{
				Kind:         FailureUnknownCommand,
				BadToken:     "stattus",
				ValidOptions: []string{"status", "list"},
			},
			setupCat: func(c *mockCatalog) {
				// catalog is empty
			},
			wantType:   SuggestionLevenshtein,
			wantCorr:   "status",
			wantReason: "levenshtein fallback when catalog has no matches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := newMockCatalog()
			tt.setupCat(catalog)
			engine := NewSuggestionEngine(catalog)

			suggestion := engine.SuggestForCommand(tt.fullCmd, tt.ctx)

			assert.NotNil(t, suggestion, tt.wantReason)
			assert.Equal(t, tt.wantType, suggestion.Type, tt.wantReason)
			assert.Equal(t, tt.wantCorr, suggestion.Corrected, tt.wantReason)
		})
	}
}

func TestSuggestForCommand_NilCatalog(t *testing.T) {
	tests := []struct {
		name     string
		fullCmd  string
		ctx      SuggestContext
		wantNil  bool
		wantType SuggestionType
		wantCorr string
	}{
		{
			name:    "nil catalog skips to levenshtein",
			fullCmd: "ox agent statis",
			ctx: SuggestContext{
				Kind:         FailureUnknownCommand,
				BadToken:     "statis",
				ValidOptions: []string{"status", "list"},
			},
			wantType: SuggestionLevenshtein,
			wantCorr: "status",
		},
		{
			name:    "nil catalog returns nil when no valid options",
			fullCmd: "ox agent statis",
			ctx: SuggestContext{
				Kind:         FailureUnknownCommand,
				BadToken:     "statis",
				ValidOptions: []string{},
			},
			wantNil: true,
		},
		{
			name:    "nil catalog returns nil when no bad token",
			fullCmd: "ox agent statis",
			ctx: SuggestContext{
				Kind:         FailureUnknownCommand,
				BadToken:     "",
				ValidOptions: []string{"status", "list"},
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewSuggestionEngine(nil)

			suggestion := engine.SuggestForCommand(tt.fullCmd, tt.ctx)

			if tt.wantNil {
				assert.Nil(t, suggestion)
				return
			}

			assert.NotNil(t, suggestion)
			assert.Equal(t, tt.wantType, suggestion.Type)
			assert.Equal(t, tt.wantCorr, suggestion.Corrected)
		})
	}
}

func TestSuggestForCommand_ReturnsNil(t *testing.T) {
	tests := []struct {
		name     string
		fullCmd  string
		ctx      SuggestContext
		setupCat func(*mockCatalog)
	}{
		{
			name:    "no match in catalog and no bad token",
			fullCmd: "ox unknown command",
			ctx: SuggestContext{
				Kind:     FailureUnknownCommand,
				BadToken: "",
			},
			setupCat: func(c *mockCatalog) {},
		},
		{
			name:    "no match in catalog and empty valid options",
			fullCmd: "ox agent xyz",
			ctx: SuggestContext{
				Kind:         FailureUnknownCommand,
				BadToken:     "xyz",
				ValidOptions: []string{},
			},
			setupCat: func(c *mockCatalog) {},
		},
		{
			name:    "no match in catalog and levenshtein distance too large",
			fullCmd: "ox agent abcdefgh",
			ctx: SuggestContext{
				Kind:         FailureUnknownCommand,
				BadToken:     "abcdefgh",
				ValidOptions: []string{"list", "status"},
			},
			setupCat: func(c *mockCatalog) {},
		},
		{
			name:    "token mapping exists but wrong failure kind",
			fullCmd: "ox agent --verboes",
			ctx: SuggestContext{
				Kind:         FailureUnknownFlag,
				BadToken:     "verboes",
				ValidOptions: []string{},
			},
			setupCat: func(c *mockCatalog) {
				// mapping exists but for different kind
				c.addToken("verboes", "verbose", FailureUnknownCommand, 0.9)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := newMockCatalog()
			tt.setupCat(catalog)
			engine := NewSuggestionEngine(catalog)

			suggestion := engine.SuggestForCommand(tt.fullCmd, tt.ctx)

			assert.Nil(t, suggestion)
		})
	}
}

func TestSuggestForCommand_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		fullCmd string
		ctx     SuggestContext
	}{
		{
			name:    "empty command",
			fullCmd: "",
			ctx:     SuggestContext{},
		},
		{
			name:    "whitespace only command",
			fullCmd: "   ",
			ctx:     SuggestContext{},
		},
		{
			name:    "just ox prefix",
			fullCmd: "ox",
			ctx:     SuggestContext{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := newMockCatalog()
			engine := NewSuggestionEngine(catalog)

			// should not panic
			suggestion := engine.SuggestForCommand(tt.fullCmd, tt.ctx)

			// no match expected for these edge cases
			assert.Nil(t, suggestion)
		})
	}
}
