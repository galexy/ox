package agentinstance

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewStoreForUser(t *testing.T) {
	tmpDir := t.TempDir()
	sageoxDir := filepath.Join(tmpDir, ".sageox")
	if err := os.MkdirAll(sageoxDir, 0755); err != nil {
		t.Fatalf("failed to create .sageox dir: %v", err)
	}

	store, err := NewStoreForUser(tmpDir, "testuser")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}

	// verify instance directory was created
	expectedPath := filepath.Join(tmpDir, ".sageox", "agent_instances", "testuser")
	info, err := os.Stat(expectedPath)
	if err != nil {
		t.Fatalf("instance directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected instance path to be a directory")
	}
}

func TestNewStoreEmptyProjectRoot(t *testing.T) {
	_, err := NewStoreForUser("", "testuser")
	if err == nil {
		t.Error("expected error for empty project root")
	}
}

func TestNewStoreEmptyUserSlug(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewStoreForUser(tmpDir, "")
	if err != nil {
		t.Fatalf("should not error for empty user slug: %v", err)
	}
	if store.userSlug != "anonymous" {
		t.Errorf("expected userSlug 'anonymous', got %q", store.userSlug)
	}
}

func TestAddAndGet(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewStoreForUser(tmpDir, "testuser")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	inst := &Instance{
		AgentID:         "OxA1b2",
		ServerSessionID: "oxsid_01KCJECKEGETGX6HC80NRYVZ3P",
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(24 * time.Hour),
		AgentType:       "claude-code",
		Model:           "claude-opus-4-5",
	}

	if err := store.Add(inst); err != nil {
		t.Fatalf("failed to add instance: %v", err)
	}

	got, err := store.Get("OxA1b2")
	if err != nil {
		t.Fatalf("failed to get instance: %v", err)
	}

	if got.AgentID != inst.AgentID {
		t.Errorf("AgentID mismatch: got %q, want %q", got.AgentID, inst.AgentID)
	}
	if got.ServerSessionID != inst.ServerSessionID {
		t.Errorf("ServerSessionID mismatch: got %q, want %q", got.ServerSessionID, inst.ServerSessionID)
	}
	if got.AgentType != inst.AgentType {
		t.Errorf("AgentType mismatch: got %q, want %q", got.AgentType, inst.AgentType)
	}
}

func TestGetNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewStoreForUser(tmpDir, "testuser")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	_, err = store.Get("OxNoSh")
	if err == nil {
		t.Error("expected error for non-existent instance")
	}
}

func TestGetExpired(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewStoreForUser(tmpDir, "testuser")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	inst := &Instance{
		AgentID:         "OxOld1",
		ServerSessionID: "oxsid_01KCJECKEGETGX6HC80NRYVZ3P",
		CreatedAt:       time.Now().Add(-48 * time.Hour),
		ExpiresAt:       time.Now().Add(-24 * time.Hour), // expired
		AgentType:       "claude-code",
	}

	if err := store.Add(inst); err != nil {
		t.Fatalf("failed to add instance: %v", err)
	}

	_, err = store.Get("OxOld1")
	if err == nil {
		t.Error("expected error for expired instance")
	}

	// Get triggers background compaction via go s.Prune(); wait for it
	// to finish so t.TempDir() cleanup doesn't race with it.
	time.Sleep(200 * time.Millisecond)
}

func TestList(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewStoreForUser(tmpDir, "testuser")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	instances := []*Instance{
		{
			AgentID:         "OxAbc1",
			ServerSessionID: "oxsid_01KCJECKEGETGX6HC80NRYVZ3P",
			CreatedAt:       time.Now(),
			ExpiresAt:       time.Now().Add(24 * time.Hour),
		},
		{
			AgentID:         "OxAbc2",
			ServerSessionID: "oxsid_01KCJECKEGETGX6HC80NRYVZ3Q",
			CreatedAt:       time.Now(),
			ExpiresAt:       time.Now().Add(24 * time.Hour),
		},
	}

	for _, inst := range instances {
		if err := store.Add(inst); err != nil {
			t.Fatalf("failed to add instance: %v", err)
		}
	}

	list, err := store.List()
	if err != nil {
		t.Fatalf("failed to list instances: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 instances, got %d", len(list))
	}
}

func TestCount(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewStoreForUser(tmpDir, "testuser")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	if store.Count() != 0 {
		t.Errorf("expected 0 instances initially, got %d", store.Count())
	}

	inst := &Instance{
		AgentID:         "OxTest",
		ServerSessionID: "oxsid_01KCJECKEGETGX6HC80NRYVZ3P",
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(24 * time.Hour),
	}

	if err := store.Add(inst); err != nil {
		t.Fatalf("failed to add instance: %v", err)
	}

	if store.Count() != 1 {
		t.Errorf("expected 1 instance, got %d", store.Count())
	}
}

func TestPrune(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewStoreForUser(tmpDir, "testuser")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// add expired instance
	expired := &Instance{
		AgentID:         "OxExp1",
		ServerSessionID: "oxsid_expired",
		CreatedAt:       time.Now().Add(-48 * time.Hour),
		ExpiresAt:       time.Now().Add(-24 * time.Hour),
	}
	if err := store.Add(expired); err != nil {
		t.Fatalf("failed to add expired instance: %v", err)
	}

	// add active instance
	active := &Instance{
		AgentID:         "OxAct1",
		ServerSessionID: "oxsid_active",
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(24 * time.Hour),
	}
	if err := store.Add(active); err != nil {
		t.Fatalf("failed to add active instance: %v", err)
	}

	pruned, err := store.Prune()
	if err != nil {
		t.Fatalf("failed to prune: %v", err)
	}

	if pruned != 1 {
		t.Errorf("expected 1 pruned, got %d", pruned)
	}

	// verify active instance still exists
	_, err = store.Get("OxAct1")
	if err != nil {
		t.Error("active instance should still exist after prune")
	}
}

func TestInstanceIsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		expected  bool
	}{
		{"future", time.Now().Add(1 * time.Hour), false},
		{"past", time.Now().Add(-1 * time.Hour), true},
		{"way past", time.Now().Add(-24 * time.Hour), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := &Instance{ExpiresAt: tt.expiresAt}
			if got := inst.IsExpired(); got != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestInstanceIsPrimeExcessive(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		expected bool
	}{
		{"zero", 0, false},
		{"one", 1, false},
		{"at threshold", ExcessivePrimeThreshold, false},
		{"above threshold", ExcessivePrimeThreshold + 1, true},
		{"way above", ExcessivePrimeThreshold + 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := &Instance{PrimeCallCount: tt.count}
			if got := inst.IsPrimeExcessive(); got != tt.expected {
				t.Errorf("IsPrimeExcessive() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewStoreForUser(tmpDir, "testuser")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	inst := &Instance{
		AgentID:         "OxUpd1",
		ServerSessionID: "oxsid_01KCJECKEGETGX6HC80NRYVZ3P",
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(24 * time.Hour),
		AgentType:       "claude-code",
		PrimeCallCount:  1,
	}

	if err := store.Add(inst); err != nil {
		t.Fatalf("failed to add instance: %v", err)
	}

	// update instance
	inst.PrimeCallCount = 5
	inst.Model = "claude-sonnet"
	if err := store.Update(inst); err != nil {
		t.Fatalf("failed to update instance: %v", err)
	}

	// verify update
	got, err := store.Get("OxUpd1")
	if err != nil {
		t.Fatalf("failed to get instance: %v", err)
	}

	if got.PrimeCallCount != 5 {
		t.Errorf("PrimeCallCount = %d, want 5", got.PrimeCallCount)
	}
	if got.Model != "claude-sonnet" {
		t.Errorf("Model = %q, want 'claude-sonnet'", got.Model)
	}
}

func TestIncrementPrimeCallCount(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewStoreForUser(tmpDir, "testuser")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	inst := &Instance{
		AgentID:         "OxInc1",
		ServerSessionID: "oxsid_01KCJECKEGETGX6HC80NRYVZ3P",
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(24 * time.Hour),
		PrimeCallCount:  0,
	}

	if err := store.Add(inst); err != nil {
		t.Fatalf("failed to add instance: %v", err)
	}

	// increment
	updated, isExcessive, err := store.IncrementPrimeCallCount("OxInc1")
	if err != nil {
		t.Fatalf("failed to increment: %v", err)
	}

	if updated.PrimeCallCount != 1 {
		t.Errorf("PrimeCallCount = %d, want 1", updated.PrimeCallCount)
	}
	if isExcessive {
		t.Error("should not be excessive after 1 increment")
	}

	// increment to threshold
	for i := 1; i <= ExcessivePrimeThreshold; i++ {
		_, _, err = store.IncrementPrimeCallCount("OxInc1")
		if err != nil {
			t.Fatalf("failed to increment: %v", err)
		}
	}

	// one more should be excessive
	_, isExcessive, err = store.IncrementPrimeCallCount("OxInc1")
	if err != nil {
		t.Fatalf("failed to increment: %v", err)
	}
	if !isExcessive {
		t.Errorf("should be excessive after %d increments", ExcessivePrimeThreshold+2)
	}
}

func TestGenerateAgentID(t *testing.T) {
	id, err := GenerateAgentID(nil)
	if err != nil {
		t.Fatalf("failed to generate ID: %v", err)
	}

	if !IsValidAgentID(id) {
		t.Errorf("generated ID is not valid: %q", id)
	}
}

func TestGenerateAgentIDAvoidCollisions(t *testing.T) {
	existing := []string{"OxAbc1", "OxAbc2", "OxAbc3"}

	id, err := GenerateAgentID(existing)
	if err != nil {
		t.Fatalf("failed to generate ID: %v", err)
	}

	for _, e := range existing {
		if id == e {
			t.Errorf("generated ID %q collides with existing", id)
		}
	}
}

func TestIsValidAgentID(t *testing.T) {
	tests := []struct {
		id       string
		expected bool
	}{
		{"OxA1b2", true},
		{"Ox1234", true},
		{"OxZzZz", true},
		{"OxAAAA", true},
		{"ox1234", false},  // lowercase prefix
		{"OX1234", false},  // uppercase X
		{"Ox123", false},   // too short
		{"Ox12345", false}, // too long
		{"Ox12#4", false},  // invalid character
		{"", false},
		{"random", false},
		{"Ox", false},
		{"O1234", false},
		{"xO1234", false},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			if got := IsValidAgentID(tt.id); got != tt.expected {
				t.Errorf("IsValidAgentID(%q) = %v, want %v", tt.id, got, tt.expected)
			}
		})
	}
}

func TestParseAgentID(t *testing.T) {
	tests := []struct {
		id       string
		suffix   string
		hasError bool
	}{
		{"OxA1b2", "A1b2", false},
		{"Ox1234", "1234", false},
		{"ox1234", "", true}, // invalid prefix
		{"Ox123", "", true},  // too short
		{"", "", true},       // empty
		{"random", "", true}, // no prefix
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			suffix, err := ParseAgentID(tt.id)
			if tt.hasError {
				if err == nil {
					t.Errorf("expected error for %q", tt.id)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for %q: %v", tt.id, err)
				}
				if suffix != tt.suffix {
					t.Errorf("suffix = %q, want %q", suffix, tt.suffix)
				}
			}
		})
	}
}
