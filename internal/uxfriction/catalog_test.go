package uxfriction

import (
	"sync"
	"testing"
)

func TestNewFrictionCatalog(t *testing.T) {
	catalog := NewFrictionCatalog()

	if catalog == nil {
		t.Fatal("NewFrictionCatalog returned nil")
	}

	if catalog.version != "" {
		t.Errorf("expected empty version, got %q", catalog.version)
	}

	if catalog.commands == nil {
		t.Error("commands map is nil")
	}

	if catalog.tokens == nil {
		t.Error("tokens map is nil")
	}

	if len(catalog.commands) != 0 {
		t.Errorf("expected empty commands map, got %d entries", len(catalog.commands))
	}

	if len(catalog.tokens) != 0 {
		t.Errorf("expected empty tokens map, got %d entries", len(catalog.tokens))
	}
}

func TestCatalogUpdate(t *testing.T) {
	tests := []struct {
		name           string
		data           CatalogData
		wantVersion    string
		wantCmdCount   int
		wantTokenCount int
	}{
		{
			name:           "empty data",
			data:           CatalogData{},
			wantVersion:    "",
			wantCmdCount:   0,
			wantTokenCount: 0,
		},
		{
			name: "version only",
			data: CatalogData{
				Version: "1.0.0",
			},
			wantVersion:    "1.0.0",
			wantCmdCount:   0,
			wantTokenCount: 0,
		},
		{
			name: "commands only",
			data: CatalogData{
				Version: "1.1.0",
				Commands: []CommandMapping{
					{Pattern: "daemons list --every", Target: "daemons show --all", Count: 5, Confidence: 0.9},
					{Pattern: "agent ls", Target: "agent list", Count: 3, Confidence: 0.8},
				},
			},
			wantVersion:    "1.1.0",
			wantCmdCount:   2,
			wantTokenCount: 0,
		},
		{
			name: "tokens only",
			data: CatalogData{
				Version: "1.2.0",
				Tokens: []TokenMapping{
					{Pattern: "depliy", Target: "deploy", Kind: FailureUnknownCommand, Count: 10, Confidence: 0.95},
					{Pattern: "satuts", Target: "status", Kind: FailureUnknownCommand, Count: 7, Confidence: 0.85},
				},
			},
			wantVersion:    "1.2.0",
			wantCmdCount:   0,
			wantTokenCount: 2,
		},
		{
			name: "commands and tokens",
			data: CatalogData{
				Version: "2.0.0",
				Commands: []CommandMapping{
					{Pattern: "ox init --force", Target: "init --yes", Count: 15, Confidence: 0.92},
				},
				Tokens: []TokenMapping{
					{Pattern: "inti", Target: "init", Kind: FailureUnknownCommand, Count: 12, Confidence: 0.88},
					{Pattern: "--verbos", Target: "--verbose", Kind: FailureUnknownFlag, Count: 8, Confidence: 0.9},
				},
			},
			wantVersion:    "2.0.0",
			wantCmdCount:   1,
			wantTokenCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := NewFrictionCatalog()

			err := catalog.Update(tt.data)
			if err != nil {
				t.Fatalf("Update returned error: %v", err)
			}

			if catalog.Version() != tt.wantVersion {
				t.Errorf("version = %q, want %q", catalog.Version(), tt.wantVersion)
			}

			if len(catalog.commands) != tt.wantCmdCount {
				t.Errorf("commands count = %d, want %d", len(catalog.commands), tt.wantCmdCount)
			}

			if len(catalog.tokens) != tt.wantTokenCount {
				t.Errorf("tokens count = %d, want %d", len(catalog.tokens), tt.wantTokenCount)
			}
		})
	}
}

func TestCatalogUpdateOverwrites(t *testing.T) {
	catalog := NewFrictionCatalog()

	// first update
	err := catalog.Update(CatalogData{
		Version: "1.0.0",
		Commands: []CommandMapping{
			{Pattern: "old cmd", Target: "target1", Count: 1, Confidence: 0.5},
		},
		Tokens: []TokenMapping{
			{Pattern: "oldtypo", Target: "correct", Kind: FailureUnknownCommand, Count: 1, Confidence: 0.5},
		},
	})
	if err != nil {
		t.Fatalf("first Update failed: %v", err)
	}

	// second update should replace everything
	err = catalog.Update(CatalogData{
		Version: "2.0.0",
		Commands: []CommandMapping{
			{Pattern: "new cmd", Target: "target2", Count: 5, Confidence: 0.9},
		},
	})
	if err != nil {
		t.Fatalf("second Update failed: %v", err)
	}

	if catalog.Version() != "2.0.0" {
		t.Errorf("version = %q, want %q", catalog.Version(), "2.0.0")
	}

	// old command should be gone
	if catalog.LookupCommand("old cmd") != nil {
		t.Error("old command should not exist after update")
	}

	// new command should exist
	if catalog.LookupCommand("new cmd") == nil {
		t.Error("new command should exist after update")
	}

	// old token should be gone (tokens map was replaced with empty)
	if catalog.LookupToken("oldtypo", FailureUnknownCommand) != nil {
		t.Error("old token should not exist after update")
	}
}

func TestLookupCommand(t *testing.T) {
	catalog := NewFrictionCatalog()
	err := catalog.Update(CatalogData{
		Version: "1.0.0",
		Commands: []CommandMapping{
			{Pattern: "daemons list --every", Target: "daemons show --all", Count: 5, Confidence: 0.9, Description: "common mistake"},
			{Pattern: "agent ls -v", Target: "agent list --verbose", Count: 3, Confidence: 0.8},
			{Pattern: "status", Target: "doctor", Count: 2, Confidence: 0.7},
		},
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	tests := []struct {
		name       string
		input      string
		wantNil    bool
		wantTarget string
	}{
		{
			name:       "exact match",
			input:      "daemons list --every",
			wantTarget: "daemons show --all",
		},
		{
			name:       "with ox prefix",
			input:      "ox daemons list --every",
			wantTarget: "daemons show --all",
		},
		{
			name:       "flags reordered",
			input:      "agent -v ls",
			wantTarget: "agent list --verbose",
		},
		{
			name:       "simple command",
			input:      "status",
			wantTarget: "doctor",
		},
		{
			name:       "simple command with ox prefix",
			input:      "ox status",
			wantTarget: "doctor",
		},
		{
			name:    "no match",
			input:   "nonexistent command",
			wantNil: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantNil: true,
		},
		{
			name:    "only ox",
			input:   "ox",
			wantNil: true,
		},
		{
			name:    "partial match should fail",
			input:   "daemons list",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := catalog.LookupCommand(tt.input)

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.Target != tt.wantTarget {
				t.Errorf("target = %q, want %q", result.Target, tt.wantTarget)
			}
		})
	}
}

func TestLookupToken(t *testing.T) {
	catalog := NewFrictionCatalog()
	err := catalog.Update(CatalogData{
		Version: "1.0.0",
		Tokens: []TokenMapping{
			{Pattern: "depliy", Target: "deploy", Kind: FailureUnknownCommand, Count: 10, Confidence: 0.95},
			{Pattern: "satuts", Target: "status", Kind: FailureUnknownCommand, Count: 7, Confidence: 0.85},
			{Pattern: "--verbos", Target: "--verbose", Kind: FailureUnknownFlag, Count: 5, Confidence: 0.8},
			{Pattern: "--hlep", Target: "--help", Kind: FailureUnknownFlag, Count: 3, Confidence: 0.75},
		},
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	tests := []struct {
		name       string
		token      string
		kind       FailureKind
		wantNil    bool
		wantTarget string
	}{
		{
			name:       "command typo match",
			token:      "depliy",
			kind:       FailureUnknownCommand,
			wantTarget: "deploy",
		},
		{
			name:       "another command typo",
			token:      "satuts",
			kind:       FailureUnknownCommand,
			wantTarget: "status",
		},
		{
			name:       "flag typo match",
			token:      "--verbos",
			kind:       FailureUnknownFlag,
			wantTarget: "--verbose",
		},
		{
			name:       "case insensitive lookup",
			token:      "DEPLIY",
			kind:       FailureUnknownCommand,
			wantTarget: "deploy",
		},
		{
			name:       "mixed case lookup",
			token:      "DePLiy",
			kind:       FailureUnknownCommand,
			wantTarget: "deploy",
		},
		{
			name:    "wrong kind returns nil",
			token:   "depliy",
			kind:    FailureUnknownFlag,
			wantNil: true,
		},
		{
			name:    "nonexistent token",
			token:   "foobar",
			kind:    FailureUnknownCommand,
			wantNil: true,
		},
		{
			name:    "empty token",
			token:   "",
			kind:    FailureUnknownCommand,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := catalog.LookupToken(tt.token, tt.kind)

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.Target != tt.wantTarget {
				t.Errorf("target = %q, want %q", result.Target, tt.wantTarget)
			}
		})
	}
}

func TestVersion(t *testing.T) {
	catalog := NewFrictionCatalog()

	// initially empty
	if v := catalog.Version(); v != "" {
		t.Errorf("initial version = %q, want empty", v)
	}

	// after update
	err := catalog.Update(CatalogData{Version: "3.14.159"})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if v := catalog.Version(); v != "3.14.159" {
		t.Errorf("version = %q, want %q", v, "3.14.159")
	}

	// update to empty version
	err = catalog.Update(CatalogData{Version: ""})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if v := catalog.Version(); v != "" {
		t.Errorf("version = %q, want empty", v)
	}
}

func TestNormalizeCommand(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "whitespace only",
			input: "   ",
			want:  "",
		},
		{
			name:  "only ox",
			input: "ox",
			want:  "",
		},
		{
			name:  "ox with whitespace",
			input: "  ox  ",
			want:  "",
		},
		{
			name:  "simple command",
			input: "agent",
			want:  "agent",
		},
		{
			name:  "simple command with ox prefix",
			input: "ox agent",
			want:  "agent",
		},
		{
			name:  "command with subcommand",
			input: "agent list",
			want:  "agent list",
		},
		{
			name:  "command with ox prefix and subcommand",
			input: "ox agent list",
			want:  "agent list",
		},
		{
			name:  "command with single flag",
			input: "agent list --verbose",
			want:  "agent list --verbose",
		},
		{
			name:  "command with multiple flags already sorted",
			input: "agent list --verbose -a",
			want:  "agent list --verbose -a",
		},
		{
			name:  "command with multiple flags unsorted",
			input: "agent list -a --verbose",
			want:  "agent list --verbose -a",
		},
		{
			name:  "flags interleaved with positionals",
			input: "agent --verbose list -a",
			want:  "agent list --verbose -a",
		},
		{
			name:  "ox prefix with flags",
			input: "ox agent --verbose list -a",
			want:  "agent list --verbose -a",
		},
		{
			name:  "multiple positionals and flags",
			input: "agent show myagent --format json -q",
			want:  "agent show myagent json --format -q",
		},
		{
			name:  "long flags sorted alphabetically",
			input: "agent --zeta --alpha --beta",
			want:  "agent --alpha --beta --zeta",
		},
		{
			name:  "short flags sorted",
			input: "agent -z -a -b",
			want:  "agent -a -b -z",
		},
		{
			name:  "mixed short and long flags sorted",
			input: "agent -z --alpha -a --beta",
			want:  "agent --alpha --beta -a -z",
		},
		{
			name:  "extra whitespace normalized",
			input: "  ox   agent   list   --verbose  ",
			want:  "agent list --verbose",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeCommand(tt.input)
			if got != tt.want {
				t.Errorf("normalizeCommand(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTokenKey(t *testing.T) {
	tests := []struct {
		name  string
		token string
		kind  FailureKind
		want  string
	}{
		{
			name:  "lowercase token",
			token: "deploy",
			kind:  FailureUnknownCommand,
			want:  "deploy:unknown-command",
		},
		{
			name:  "uppercase token gets lowercased",
			token: "DEPLOY",
			kind:  FailureUnknownCommand,
			want:  "deploy:unknown-command",
		},
		{
			name:  "mixed case token",
			token: "DePlOy",
			kind:  FailureUnknownFlag,
			want:  "deploy:unknown-flag",
		},
		{
			name:  "empty token",
			token: "",
			kind:  FailureInvalidArg,
			want:  ":invalid-arg",
		},
		{
			name:  "flag with dashes",
			token: "--verbose",
			kind:  FailureUnknownFlag,
			want:  "--verbose:unknown-flag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenKey(tt.token, tt.kind)
			if got != tt.want {
				t.Errorf("tokenKey(%q, %q) = %q, want %q", tt.token, tt.kind, got, tt.want)
			}
		})
	}
}

func TestConcurrentAccess(t *testing.T) {
	catalog := NewFrictionCatalog()

	// populate initial data
	err := catalog.Update(CatalogData{
		Version: "1.0.0",
		Commands: []CommandMapping{
			{Pattern: "test cmd", Target: "target", Count: 1, Confidence: 0.9},
		},
		Tokens: []TokenMapping{
			{Pattern: "typo", Target: "correct", Kind: FailureUnknownCommand, Count: 1, Confidence: 0.9},
		},
	})
	if err != nil {
		t.Fatalf("initial Update failed: %v", err)
	}

	var wg sync.WaitGroup
	done := make(chan struct{})

	// concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					_ = catalog.LookupCommand("test cmd")
					_ = catalog.LookupToken("typo", FailureUnknownCommand)
					_ = catalog.Version()
				}
			}
		}()
	}

	// concurrent writers
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(version int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				select {
				case <-done:
					return
				default:
					_ = catalog.Update(CatalogData{
						Version: "1.0." + string(rune('0'+version)),
						Commands: []CommandMapping{
							{Pattern: "test cmd", Target: "target", Count: j, Confidence: 0.9},
						},
					})
				}
			}
		}(i)
	}

	// let it run for a bit
	go func() {
		wg.Wait()
	}()

	// stop after some iterations
	for i := 0; i < 300; i++ {
		_ = catalog.Version()
	}
	close(done)

	// wait for goroutines to finish
	wg.Wait()

	// if we got here without race detector complaints, concurrent access is safe
}

func TestCatalogImplementsInterface(t *testing.T) {
	// compile-time check that FrictionCatalog implements Catalog
	var _ Catalog = (*FrictionCatalog)(nil)
}

func TestLookupCommandReturnsPointerToStoredMapping(t *testing.T) {
	catalog := NewFrictionCatalog()
	err := catalog.Update(CatalogData{
		Version: "1.0.0",
		Commands: []CommandMapping{
			{Pattern: "test", Target: "result", Count: 5, Confidence: 0.9, Description: "test mapping"},
		},
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	result := catalog.LookupCommand("test")
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// verify all fields are accessible
	if result.Pattern != "test" {
		t.Errorf("Pattern = %q, want %q", result.Pattern, "test")
	}
	if result.Target != "result" {
		t.Errorf("Target = %q, want %q", result.Target, "result")
	}
	if result.Count != 5 {
		t.Errorf("Count = %d, want %d", result.Count, 5)
	}
	if result.Confidence != 0.9 {
		t.Errorf("Confidence = %f, want %f", result.Confidence, 0.9)
	}
	if result.Description != "test mapping" {
		t.Errorf("Description = %q, want %q", result.Description, "test mapping")
	}
}

func TestRegexPatternMatching(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		target     string
		input      string
		wantNil    bool
		wantTarget string
	}{
		{
			name:       "simple regex match with capture",
			pattern:    `agent close ([a-zA-Z0-9-]+)`,
			target:     "agent $1 session stop",
			input:      "agent close Oxa7b3",
			wantTarget: "agent Oxa7b3 session stop",
		},
		{
			name:       "multiple capture groups",
			pattern:    `agent ([a-zA-Z0-9-]+) move ([a-zA-Z0-9-]+)`,
			target:     "agent $1 session move --dest=$2",
			input:      "agent Oxa7b3 move archive",
			wantTarget: "agent Oxa7b3 session move --dest=archive",
		},
		{
			name:    "no match",
			pattern: `agent close ([a-zA-Z0-9-]+)`,
			target:  "agent $1 session stop",
			input:   "agent list",
			wantNil: true,
		},
		{
			name:       "regex without capture groups",
			pattern:    `daemons (list|show) --every`,
			target:     "daemons show --all",
			input:      "daemons list --every",
			wantTarget: "daemons show --all",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := NewFrictionCatalog()
			err := catalog.Update(CatalogData{
				Version: "1.0.0",
				Commands: []CommandMapping{
					{
						Pattern:     tt.pattern,
						Target:      tt.target,
						HasRegex:    true,
						AutoExecute: true,
						Confidence:  0.95,
					},
				},
			})
			if err != nil {
				t.Fatalf("Update failed: %v", err)
			}

			result := catalog.LookupCommand(tt.input)

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			// apply the mapping to get the corrected command
			corrected, ok := result.ApplyMapping(tt.input)
			if !ok {
				t.Fatal("ApplyMapping returned false")
			}

			if corrected != tt.wantTarget {
				t.Errorf("corrected = %q, want %q", corrected, tt.wantTarget)
			}
		})
	}
}

func TestAutoExecuteField(t *testing.T) {
	catalog := NewFrictionCatalog()
	err := catalog.Update(CatalogData{
		Version: "1.0.0",
		Commands: []CommandMapping{
			{
				Pattern:     "agent prine",
				Target:      "agent prime",
				HasRegex:    false,
				AutoExecute: true,
				Confidence:  0.95,
				Description: "common typo",
			},
			{
				Pattern:     "agent xyz",
				Target:      "agent abc",
				HasRegex:    false,
				AutoExecute: false,
				Confidence:  0.7,
				Description: "low confidence, suggest only",
			},
		},
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// test auto-execute enabled
	result := catalog.LookupCommand("agent prine")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.AutoExecute {
		t.Error("expected AutoExecute = true")
	}

	// test auto-execute disabled
	result2 := catalog.LookupCommand("agent xyz")
	if result2 == nil {
		t.Fatal("expected non-nil result")
	}
	if result2.AutoExecute {
		t.Error("expected AutoExecute = false")
	}
}

func TestApplyMapping(t *testing.T) {
	tests := []struct {
		name     string
		mapping  CommandMapping
		input    string
		wantCorr string
		wantOk   bool
	}{
		{
			name: "literal pattern returns target as-is",
			mapping: CommandMapping{
				Pattern:  "agent prine",
				Target:   "agent prime",
				HasRegex: false,
			},
			input:    "agent prine",
			wantCorr: "agent prime",
			wantOk:   true,
		},
		{
			name: "regex with single capture",
			mapping: CommandMapping{
				Pattern:  `agent close ([a-zA-Z0-9-]+)`,
				Target:   "agent $1 session stop",
				HasRegex: true,
			},
			input:    "agent close Oxa7b3",
			wantCorr: "agent Oxa7b3 session stop",
			wantOk:   true,
		},
		{
			name: "regex with multiple captures",
			mapping: CommandMapping{
				Pattern:  `agent ([a-zA-Z0-9-]+) rename ([a-zA-Z0-9-]+)`,
				Target:   "agent $1 session rename --new-name=$2",
				HasRegex: true,
			},
			input:    "agent Oxa7b3 rename MyNewName",
			wantCorr: "agent Oxa7b3 session rename --new-name=MyNewName",
			wantOk:   true,
		},
		{
			name: "regex no match returns false",
			mapping: CommandMapping{
				Pattern:  `agent close ([a-zA-Z0-9-]+)`,
				Target:   "agent $1 session stop",
				HasRegex: true,
			},
			input:    "agent list",
			wantCorr: "",
			wantOk:   false,
		},
		{
			name: "invalid regex returns false",
			mapping: CommandMapping{
				Pattern:  `agent close ([`,
				Target:   "agent $1 session stop",
				HasRegex: true,
			},
			input:    "agent close test",
			wantCorr: "",
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			corrected, ok := tt.mapping.ApplyMapping(tt.input)

			if ok != tt.wantOk {
				t.Errorf("ok = %v, want %v", ok, tt.wantOk)
			}
			if corrected != tt.wantCorr {
				t.Errorf("corrected = %q, want %q", corrected, tt.wantCorr)
			}
		})
	}
}

func TestInvalidRegexSkipped(t *testing.T) {
	catalog := NewFrictionCatalog()
	err := catalog.Update(CatalogData{
		Version: "1.0.0",
		Commands: []CommandMapping{
			{
				Pattern:     `agent close ([`, // invalid regex
				Target:      "agent $1 session stop",
				HasRegex:    true,
				AutoExecute: true,
				Confidence:  0.95,
			},
			{
				Pattern:     "agent prine",
				Target:      "agent prime",
				HasRegex:    false,
				AutoExecute: true,
				Confidence:  0.95,
			},
		},
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// invalid regex should be skipped, valid literal should work
	result := catalog.LookupCommand("agent prine")
	if result == nil {
		t.Fatal("expected valid literal pattern to work")
	}
	if result.Target != "agent prime" {
		t.Errorf("Target = %q, want %q", result.Target, "agent prime")
	}
}

func TestLookupTokenReturnsPointerToStoredMapping(t *testing.T) {
	catalog := NewFrictionCatalog()
	err := catalog.Update(CatalogData{
		Version: "1.0.0",
		Tokens: []TokenMapping{
			{Pattern: "typo", Target: "correct", Kind: FailureUnknownCommand, Count: 8, Confidence: 0.85},
		},
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	result := catalog.LookupToken("typo", FailureUnknownCommand)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// verify all fields are accessible
	if result.Pattern != "typo" {
		t.Errorf("Pattern = %q, want %q", result.Pattern, "typo")
	}
	if result.Target != "correct" {
		t.Errorf("Target = %q, want %q", result.Target, "correct")
	}
	if result.Kind != FailureUnknownCommand {
		t.Errorf("Kind = %q, want %q", result.Kind, FailureUnknownCommand)
	}
	if result.Count != 8 {
		t.Errorf("Count = %d, want %d", result.Count, 8)
	}
	if result.Confidence != 0.85 {
		t.Errorf("Confidence = %f, want %f", result.Confidence, 0.85)
	}
}
