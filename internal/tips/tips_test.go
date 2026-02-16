// internal/tips/tips_test.go
package tips

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTip_ReturnsContextualTip(t *testing.T) {
	tip := GetTip("login")

	assert.NotEmpty(t, tip, "expected a tip for login command")
}

func TestGetTip_ReturnsGeneralTipForUnknownCommand(t *testing.T) {
	tip := GetTip("unknown-command")

	assert.NotEmpty(t, tip, "expected a general tip for unknown command")
}

func TestShouldShow_ReturnsFalseWhenSuppressed(t *testing.T) {
	result := ShouldShow(AlwaysShow, true, true, false)
	assert.False(t, result, "expected false when quiet mode is enabled")

	result = ShouldShow(AlwaysShow, false, false, true)
	assert.False(t, result, "expected false when json mode is enabled")

	result = ShouldShow(AlwaysShow, false, true, false)
	assert.False(t, result, "expected false when tips are disabled")
}

func TestShouldShow_AlwaysShowReturnsTrue(t *testing.T) {
	result := ShouldShow(AlwaysShow, false, false, false)
	assert.True(t, result, "expected true for AlwaysShow mode")
}

func TestShouldShow_WhenMinimalReturnsTrue(t *testing.T) {
	result := ShouldShow(WhenMinimal, false, false, false)
	assert.True(t, result, "expected true for WhenMinimal mode")
}

func TestShouldShow_RandomChanceReturnsBoolean(t *testing.T) {
	// run multiple times to verify it returns boolean values
	for i := 0; i < 100; i++ {
		result := ShouldShow(RandomChance, false, false, false)
		// result should be a boolean (true or false)
		// this test verifies the function doesn't panic and returns a bool
		_ = result
	}
}

func TestGetRandomFromPool_EmptyPool(t *testing.T) {
	result := getRandomFromPool([]string{})
	assert.Empty(t, result, "expected empty string for empty pool")
}

func TestContextualTips_NotEmpty(t *testing.T) {
	// validate no empty tip arrays in humanContextualTips
	for command, tips := range humanContextualTips {
		assert.NotEmpty(t, tips, "human command %q has empty tips array", command)
		for i, tip := range tips {
			assert.NotEmpty(t, tip, "human command %q has empty tip at index %d", command, i)
		}
	}

	// validate no empty tip arrays in agentContextualTips
	for command, tips := range agentContextualTips {
		assert.NotEmpty(t, tips, "agent command %q has empty tips array", command)
		for i, tip := range tips {
			assert.NotEmpty(t, tip, "agent command %q has empty tip at index %d", command, i)
		}
	}
}

func TestIsAgentCommand(t *testing.T) {
	agentCmds := []string{"prime", "agent"}
	for _, cmd := range agentCmds {
		assert.True(t, isAgentCommand(cmd), "expected %q to be an agent command", cmd)
	}

	humanCmds := []string{"login", "status", "init", "doctor", "hooks"}
	for _, cmd := range humanCmds {
		assert.False(t, isAgentCommand(cmd), "expected %q to be a human command", cmd)
	}
}

func TestGetTip_AgentCommands(t *testing.T) {
	tip := GetTip("prime")
	assert.NotEmpty(t, tip, "expected a tip for prime command")

	tip = GetTip("agent")
	assert.NotEmpty(t, tip, "expected a tip for agent command")
}

func TestMaybeShow_IntegratesWithCLI(t *testing.T) {
	// verify it doesn't panic when called
	MaybeShow("login", AlwaysShow, false, false, false)
}
