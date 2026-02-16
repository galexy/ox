#!/bin/bash
# generate-ldflags.sh - Generate ldflags for manifest signing
#
# This script generates the Ed25519 signature ldflags for embedding
# in the ox binary. Run this before goreleaser to get the flags.
#
# Usage:
#   export REDACTION_SIGNING_KEY=<base64-private-key>
#   source scripts/sign-manifest/generate-ldflags.sh
#   # MANIFEST_LDFLAGS is now set
#
# Or for CI:
#   eval $(go run ./scripts/sign-manifest)

set -e

if [ -z "$REDACTION_SIGNING_KEY" ]; then
    echo "Error: REDACTION_SIGNING_KEY not set" >&2
    echo "Skipping manifest signing (development build)" >&2
    export MANIFEST_LDFLAGS=""
    exit 0
fi

# generate ldflags using the signing tool
output=$(go run ./scripts/sign-manifest 2>&1) || {
    echo "Error: Failed to generate manifest signature" >&2
    echo "$output" >&2
    exit 1
}

# extract the MANIFEST_LDFLAGS line
ldflags=$(echo "$output" | grep "^MANIFEST_LDFLAGS=" | cut -d= -f2-)

if [ -z "$ldflags" ]; then
    echo "Error: Failed to extract MANIFEST_LDFLAGS from output" >&2
    echo "$output" >&2
    exit 1
fi

export MANIFEST_LDFLAGS="$ldflags"
echo "Manifest signature generated successfully" >&2
echo "$ldflags"
