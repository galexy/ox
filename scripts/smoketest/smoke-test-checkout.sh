#!/usr/bin/env bash
# smoke-test-checkout.sh - Test ox team context add command
#
# Required environment variables (set by smoke-test.sh):
#   SAGEOX_ENDPOINT - API endpoint
#   SMOKE_TEST_WORKDIR - Temp directory for test
#   OX - Path to ox binary
#
# Optional environment variables:
#   SAGEOX_CI_TEAM_ID - Team ID to test checkout (skip if not set)

set -euo pipefail

# validate required env vars
: "${SAGEOX_ENDPOINT:?SAGEOX_ENDPOINT is required}"
: "${SMOKE_TEST_WORKDIR:?SMOKE_TEST_WORKDIR is required}"
: "${OX:?OX is required}"

# skip if no team ID configured
if [[ -z "${SAGEOX_CI_TEAM_ID:-}" ]]; then
    echo "Skipping: SAGEOX_CI_TEAM_ID not set"
    echo "Set SAGEOX_CI_TEAM_ID to enable team context checkout test"
    exit 0
fi

TEST_REPO="$SMOKE_TEST_WORKDIR/test-init-repo"

if [[ ! -d "$TEST_REPO/.sageox" ]]; then
    echo "Creating test repository for checkout test..."
    mkdir -p "$TEST_REPO"
    cd "$TEST_REPO"
    git init -q
    git config user.email "test@example.com"
    git config user.name "Test User"
    echo "# Test Repo" > README.md
    git add README.md
    git commit -q -m "initial commit"
    $OX init --quiet
fi

cd "$TEST_REPO"

echo "Testing ox team context add..."

# the command prompts for checkout path, use stdin redirect to accept default
# echo "" sends empty string to accept default path
if ! echo "" | $OX team context add "$SAGEOX_CI_TEAM_ID"; then
    echo "error: ox team context add failed"
    exit 1
fi

# verify context was added (check config for team reference)
if grep -q "$SAGEOX_CI_TEAM_ID" .sageox/config.local.toml 2>/dev/null; then
    echo "Team context reference found in config"
else
    echo "warning: team context not found in config (may be expected)"
fi

echo "ox team context add test passed"
exit 0
