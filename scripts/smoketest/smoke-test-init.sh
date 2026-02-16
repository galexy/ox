#!/usr/bin/env bash
# smoke-test-init.sh - Test ox init command
#
# Required environment variables (set by smoke-test.sh):
#   SAGEOX_ENDPOINT - API endpoint
#   SMOKE_TEST_WORKDIR - Temp directory for test
#   OX - Path to ox binary

set -euo pipefail

# validate required env vars
: "${SAGEOX_ENDPOINT:?SAGEOX_ENDPOINT is required}"
: "${SMOKE_TEST_WORKDIR:?SMOKE_TEST_WORKDIR is required}"
: "${OX:?OX is required}"

TEST_REPO="$SMOKE_TEST_WORKDIR/test-init-repo"

echo "Testing ox init..."
echo "Creating test git repository at $TEST_REPO"

# create a fresh git repo
mkdir -p "$TEST_REPO"
cd "$TEST_REPO"
git init -q
git config user.email "test@example.com"
git config user.name "Test User"

# create initial commit (required for ox init)
echo "# Test Repo" > README.md
git add README.md
git commit -q -m "initial commit"

echo "Running ox init..."
if ! $OX init --quiet; then
    echo "error: ox init failed"
    exit 1
fi

# verify .sageox directory was created
if [[ ! -d ".sageox" ]]; then
    echo "error: .sageox directory not created"
    exit 1
fi

echo ".sageox directory created successfully"

# verify config file exists
if [[ -f ".sageox/config.toml" ]]; then
    echo "config.toml created"
elif [[ -f ".sageox/config.local.toml" ]]; then
    echo "config.local.toml created"
else
    echo "warning: no config file found in .sageox/"
fi

# check for repo marker file
repo_marker=$(find .sageox -name ".repo_*" -type f 2>/dev/null | head -1)
if [[ -n "$repo_marker" ]]; then
    echo "Repo marker file created: $(basename "$repo_marker")"
else
    echo "warning: no repo marker file found"
fi

echo "ox init test passed"
exit 0
