#!/usr/bin/env bash
# smoke-test-doctor.sh - Test ox doctor cloud command
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

# use the test repo created by init test, or create a new one
TEST_REPO="$SMOKE_TEST_WORKDIR/test-init-repo"

if [[ ! -d "$TEST_REPO/.sageox" ]]; then
    echo "Creating test repository for doctor test..."
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

echo "Testing ox doctor..."

# run doctor and capture output
# doctor may return non-zero if there are warnings, which is okay
set +e
output=$($OX doctor 2>&1)
exit_code=$?
set -e

echo "$output"

# check for critical errors (not warnings)
if echo "$output" | grep -qi "error.*failed\|fatal\|panic"; then
    echo "error: ox doctor reported critical errors"
    exit 1
fi

# doctor returning 0 or 1 (warnings) is acceptable
if [[ $exit_code -gt 1 ]]; then
    echo "error: ox doctor failed with exit code $exit_code"
    exit 1
fi

echo "ox doctor test passed"
exit 0
