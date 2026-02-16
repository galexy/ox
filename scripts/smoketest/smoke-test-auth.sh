#!/usr/bin/env bash
# smoke-test-auth.sh - Authenticate with SageOx using test account credentials
#
# Required environment variables (set by smoke-test.sh):
#   SAGEOX_ENDPOINT - API endpoint
#   SAGEOX_CI_EMAIL - Test account email
#   SAGEOX_CI_PASSWORD - Test account password

set -euo pipefail

# validate required env vars
: "${SAGEOX_ENDPOINT:?SAGEOX_ENDPOINT is required}"
: "${SAGEOX_CI_EMAIL:?SAGEOX_CI_EMAIL is required}"
: "${SAGEOX_CI_PASSWORD:?SAGEOX_CI_PASSWORD is required}"

echo "Authenticating as $SAGEOX_CI_EMAIL against $SAGEOX_ENDPOINT..."

# login via API
response=$(curl -s -w "\n%{http_code}" -X POST "$SAGEOX_ENDPOINT/api/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\": \"$SAGEOX_CI_EMAIL\", \"password\": \"$SAGEOX_CI_PASSWORD\"}")

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [[ "$http_code" != "200" ]]; then
    echo "error: login failed with status $http_code"
    echo "response: $body"
    exit 1
fi

# extract token
access_token=$(echo "$body" | jq -r '.access_token // empty')
if [[ -z "$access_token" ]]; then
    echo "error: no access_token in response"
    echo "response: $body"
    exit 1
fi

# calculate expiry (1 hour from now)
# handle both GNU date and BSD date
if date -d '+1 hour' +%Y-%m-%dT%H:%M:%SZ &>/dev/null; then
    expires_at=$(date -u -d '+1 hour' +%Y-%m-%dT%H:%M:%SZ)
else
    expires_at=$(date -u -v+1H +%Y-%m-%dT%H:%M:%SZ)
fi

# create auth directory
mkdir -p ~/.config/sageox

# write auth.json
cat > ~/.config/sageox/auth.json << EOF
{
  "tokens": {
    "$SAGEOX_ENDPOINT": {
      "access_token": "$access_token",
      "expires_at": "$expires_at"
    }
  }
}
EOF

echo "Authentication successful"
echo "Token stored in ~/.config/sageox/auth.json"

# verify auth works
if [[ -n "${OX:-}" ]]; then
    echo "Verifying authentication with ox..."
    if $OX status &>/dev/null; then
        echo "Verification successful"
    else
        echo "Note: ox status verification failed"
    fi
fi

exit 0
