#!/usr/bin/env bash
#
# SageOx Demo Setup Script
#
# Prepares environment for recording the ox CLI demo with VHS.
# Handles authentication via SOPS-encrypted credentials.
#
# Usage:
#   ./demo/setup.sh              # Full setup
#   ./demo/setup.sh --clean      # Remove demo artifacts
#   ./demo/setup.sh --auth-only  # Just authenticate
#
# Environment variables (override SOPS credentials):
#   DEMO_EMAIL     - Demo account email
#   DEMO_PASSWORD  - Demo account password
#   SKIP_AUTH      - Set to 1 to skip authentication
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DEMO_DIR="/tmp/sageox-demo"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}→${NC} $1"; }
log_success() { echo -e "${GREEN}✓${NC} $1"; }
log_warn() { echo -e "${YELLOW}⚠${NC} $1"; }
log_error() { echo -e "${RED}✗${NC} $1"; }

# Load credentials from SOPS or environment
load_credentials() {
    # Environment variables take precedence
    if [[ -n "${DEMO_EMAIL:-}" ]] && [[ -n "${DEMO_PASSWORD:-}" ]]; then
        log_info "Using credentials from environment"
        return 0
    fi

    # Try SOPS-encrypted credentials
    local creds_file="$SCRIPT_DIR/credentials.sops.yaml"
    if [[ -f "$creds_file" ]]; then
        log_info "Decrypting credentials from SOPS..."
        if ! command -v sops &>/dev/null; then
            log_error "sops not installed. Install with: brew install sops"
            return 1
        fi

        # Extract credentials using sops + yq
        if ! command -v yq &>/dev/null; then
            log_error "yq not installed. Install with: brew install yq"
            return 1
        fi

        local decrypted
        decrypted=$(sops -d "$creds_file" 2>/dev/null) || {
            log_error "Failed to decrypt credentials. Check SOPS key setup."
            log_info "See: https://sageox.ai/docs/setup/sops"
            return 1
        }

        export DEMO_EMAIL=$(echo "$decrypted" | yq -r '.demo.email')
        export DEMO_PASSWORD=$(echo "$decrypted" | yq -r '.demo.password')
        log_success "Credentials loaded from SOPS"
        return 0
    fi

    log_error "No credentials found"
    log_info "Either:"
    log_info "  1. Set DEMO_EMAIL and DEMO_PASSWORD environment variables"
    log_info "  2. Create demo/credentials.sops.yaml (see credentials.example.yaml)"
    return 1
}

# Build ox from source
build_ox() {
    log_info "Building ox from source..."
    (cd "$PROJECT_ROOT" && go build -o "$DEMO_DIR/bin/ox" ./cmd/ox)
    export PATH="$DEMO_DIR/bin:$PATH"
    log_success "Built ox to $DEMO_DIR/bin/ox"
}

# Setup Playwright for browser automation
setup_playwright() {
    log_info "Setting up Playwright..."
    local pw_dir="$SCRIPT_DIR/playwright"

    if [[ ! -d "$pw_dir/node_modules" ]]; then
        (cd "$pw_dir" && npm install --silent)
        (cd "$pw_dir" && npx playwright install chromium --with-deps 2>/dev/null || npx playwright install chromium)
    fi
    log_success "Playwright ready"
}

# Perform login with browser automation
perform_login() {
    if [[ "${SKIP_AUTH:-}" == "1" ]]; then
        log_info "Skipping authentication (SKIP_AUTH=1)"
        return 0
    fi

    log_info "Starting ox login..."

    # Start ox login in background, capture output for verification URL
    local login_output
    login_output=$(mktemp)

    # Run ox login with SKIP_BROWSER=1 to prevent auto-opening browser
    SKIP_BROWSER=1 "$DEMO_DIR/bin/ox" login 2>&1 | tee "$login_output" &
    local ox_pid=$!

    # Wait for URL to appear in output
    local url=""
    local attempts=0
    while [[ -z "$url" ]] && [[ $attempts -lt 30 ]]; do
        sleep 1
        url=$(grep -oE 'https://[^ ]+device[^ ]*' "$login_output" 2>/dev/null | head -1 || true)
        ((attempts++))
    done

    if [[ -z "$url" ]]; then
        log_error "Could not find verification URL"
        kill $ox_pid 2>/dev/null || true
        rm -f "$login_output"
        return 1
    fi

    log_info "Found verification URL: $url"
    log_info "Automating browser login..."

    # Run Playwright to complete login
    (cd "$SCRIPT_DIR/playwright" && \
        DEMO_EMAIL="$DEMO_EMAIL" \
        DEMO_PASSWORD="$DEMO_PASSWORD" \
        HEADLESS="${HEADLESS:-true}" \
        npx ts-node login.ts "$url")

    # Wait for ox login to complete
    wait $ox_pid || {
        log_error "ox login failed"
        rm -f "$login_output"
        return 1
    }

    rm -f "$login_output"
    log_success "Authentication complete"
}

# Create demo directory
create_demo_dir() {
    log_info "Creating demo directory..."
    rm -rf "$DEMO_DIR"
    mkdir -p "$DEMO_DIR/bin"
    mkdir -p "$DEMO_DIR/repo"

    # Initialize a git repo for demo
    (cd "$DEMO_DIR/repo" && git init -q && git commit --allow-empty -m "initial" -q)
    log_success "Demo directory ready: $DEMO_DIR"
}

# Clean up
cleanup() {
    log_info "Cleaning up..."
    rm -rf "$DEMO_DIR"
    log_success "Cleaned up demo artifacts"
}

# Print environment for VHS tape
print_env() {
    echo ""
    echo "═══════════════════════════════════════════════════════════════"
    echo "  Demo Environment Ready"
    echo "═══════════════════════════════════════════════════════════════"
    echo ""
    echo "  Demo directory: $DEMO_DIR"
    echo "  ox binary:      $DEMO_DIR/bin/ox"
    echo ""
    echo "  For VHS recording:"
    echo "    export PATH=\"$DEMO_DIR/bin:\$PATH\""
    echo "    cd $DEMO_DIR/repo"
    echo "    vhs demo/demo.tape"
    echo ""
    echo "═══════════════════════════════════════════════════════════════"
}

# Main
main() {
    echo ""
    echo "SageOx Demo Setup"
    echo "─────────────────"
    echo ""

    case "${1:-}" in
        --clean)
            cleanup
            exit 0
            ;;
        --auth-only)
            load_credentials
            setup_playwright
            build_ox
            perform_login
            exit 0
            ;;
        *)
            create_demo_dir
            build_ox
            load_credentials
            setup_playwright
            perform_login
            print_env
            ;;
    esac
}

main "$@"
