#!/usr/bin/env bash
# test-friction.sh - Generate friction events and verify they're sent to SageOx servers
#
# Usage:
#   ./scripts/test-friction.sh           # Run full test
#   ./scripts/test-friction.sh --quick   # Generate events only (no wait)
#   ./scripts/test-friction.sh --help    # Show help
#
# Prerequisites:
#   - ox CLI built and in PATH (or run from repo root)
#   - Network access to project endpoint (or SAGEOX_FRICTION_ENDPOINT set)

set -euo pipefail

# colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # no color

# config
FLUSH_THRESHOLD=20
FLUSH_INTERVAL=900

log_info() { echo -e "${BLUE}ℹ${NC} $*"; }
log_ok() { echo -e "${GREEN}✓${NC} $*"; }
log_warn() { echo -e "${YELLOW}⚠${NC} $*"; }
log_err() { echo -e "${RED}✗${NC} $*"; }

show_help() {
    cat <<EOF
test-friction.sh - Generate friction events and verify server upload

USAGE:
    ./scripts/test-friction.sh [OPTIONS]

OPTIONS:
    --quick     Generate events only, don't wait for flush
    --count N   Number of events to generate (default: 25)
    --endpoint  Show configured friction endpoint
    --help      Show this help

EXAMPLES:
    # Full test with wait
    ./scripts/test-friction.sh

    # Quick burst of events
    ./scripts/test-friction.sh --quick --count 50

    # Check endpoint configuration
    ./scripts/test-friction.sh --endpoint

ENVIRONMENT:
    SAGEOX_FRICTION_ENDPOINT  Override friction API endpoint
    DO_NOT_TRACK=1            Disable friction collection
    SAGEOX_FRICTION=false     Disable friction collection
EOF
}

check_ox() {
    if ! command -v ox &>/dev/null; then
        # try local build
        if [[ -f "./ox" ]]; then
            OX="./ox"
        elif [[ -f "./bin/ox" ]]; then
            OX="./bin/ox"
        else
            log_err "ox CLI not found. Build with: make build"
            exit 1
        fi
    else
        OX="ox"
    fi
    log_ok "Using ox: $($OX version 2>/dev/null || echo 'unknown')"
}

check_daemon() {
    log_info "Checking daemon status..."

    if $OX daemon status &>/dev/null; then
        log_ok "Daemon is running"

        # show friction stats if available
        if friction_stats=$($OX daemon status --json 2>/dev/null | jq -r '.friction // empty' 2>/dev/null); then
            if [[ -n "$friction_stats" ]]; then
                buffer_count=$(echo "$friction_stats" | jq -r '.buffer_count // 0')
                sample_rate=$(echo "$friction_stats" | jq -r '.sample_rate // 1')
                log_info "Current buffer: $buffer_count events, sample_rate: $sample_rate"
            fi
        fi
        return 0
    else
        log_warn "Daemon not running. Starting..."
        $OX daemon start
        sleep 1

        if $OX daemon status &>/dev/null; then
            log_ok "Daemon started"
            return 0
        else
            log_err "Failed to start daemon"
            exit 1
        fi
    fi
}

show_endpoint() {
    endpoint="${SAGEOX_FRICTION_ENDPOINT:-<project endpoint>}"
    log_info "Friction endpoint: $endpoint/api/v1/cli/friction"
    log_info "(Uses project's configured endpoint; override with SAGEOX_FRICTION_ENDPOINT)"

    if [[ "${DO_NOT_TRACK:-}" == "1" ]]; then
        log_warn "DO_NOT_TRACK=1 is set - friction is DISABLED"
    fi
    if [[ "${SAGEOX_FRICTION:-}" == "false" ]]; then
        log_warn "SAGEOX_FRICTION=false is set - friction is DISABLED"
    fi
}

generate_events() {
    local count="${1:-25}"
    log_info "Generating $count friction events..."

    local generated=0
    local errors=(
        # unknown commands (typos)
        "initt"
        "statuss"
        "agnt"
        "sesion"
        "loginn"
        "confg"
        "daemn"
        "versoin"
        "helpp"
        "doctorr"
        # completely unknown
        "foo"
        "bar"
        "baz"
        "xyz"
        "unknown"
        # unknown flags
        "init --verbos"
        "status --detaled"
        "agent --forec"
        "login --quiett"
        "daemon --debugg"
        # nested command typos
        "agent prim"
        "agent statuss"
        "session startt"
        "session stopp"
        "config sett"
    )

    while [[ $generated -lt $count ]]; do
        for cmd in "${errors[@]}"; do
            if [[ $generated -ge $count ]]; then
                break
            fi

            # run command, suppress output
            $OX $cmd &>/dev/null || true
            ((generated++))

            # progress indicator
            if ((generated % 10 == 0)); then
                echo -ne "\r  Generated $generated/$count events..."
            fi
        done
    done

    echo -e "\r  Generated $generated events                    "
    log_ok "Generated $generated friction events"
}

wait_for_flush() {
    log_info "Waiting for daemon to flush events to server..."
    log_info "(Events flush every ${FLUSH_INTERVAL}s or when ${FLUSH_THRESHOLD}+ buffered)"

    # get initial buffer count
    initial_count=$($OX daemon status --json 2>/dev/null | jq -r '.friction.buffer_count // 0' 2>/dev/null || echo "0")

    if [[ "$initial_count" == "0" ]]; then
        log_ok "Buffer already empty - events were sent!"
        return 0
    fi

    log_info "Buffer has $initial_count events, waiting for flush..."

    # wait up to 60 seconds for flush
    for i in {1..12}; do
        sleep 5
        current_count=$($OX daemon status --json 2>/dev/null | jq -r '.friction.buffer_count // 0' 2>/dev/null || echo "0")

        if [[ "$current_count" == "0" ]]; then
            log_ok "Events flushed to server!"
            return 0
        fi

        echo -ne "\r  Waiting... ($current_count events buffered, ${i}0s elapsed)"
    done

    echo ""
    log_warn "Events still buffered after 60s. They will be sent eventually."
    log_info "Tip: Run '$OX daemon stop && $OX daemon start' to force flush"
}

force_flush() {
    log_info "Forcing flush by restarting daemon..."
    $OX daemon stop 2>/dev/null || true
    sleep 1
    $OX daemon start
    log_ok "Daemon restarted - final flush completed"
}

show_stats() {
    log_info "Current friction stats:"
    if stats=$($OX daemon status --json 2>/dev/null); then
        echo "$stats" | jq '.friction // "friction stats not available"'
    else
        log_warn "Could not get daemon status"
    fi
}

# --- main ---

QUICK=false
COUNT=25
SHOW_ENDPOINT=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --quick)
            QUICK=true
            shift
            ;;
        --count)
            COUNT="$2"
            shift 2
            ;;
        --endpoint)
            SHOW_ENDPOINT=true
            shift
            ;;
        --help|-h)
            show_help
            exit 0
            ;;
        *)
            log_err "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

echo ""
echo "╔════════════════════════════════════════╗"
echo "║     Friction Events Test Script        ║"
echo "╚════════════════════════════════════════╝"
echo ""
echo -e "${YELLOW}TIP:${NC} For best debugging, run the daemon in foreground in another terminal:"
echo ""
echo -e "  ${GREEN}SAGEOX_LOG_LEVEL=debug ox daemon start --foreground -v${NC}"
echo ""
echo "Then run this script to see friction events being sent in real-time."
echo ""
echo "────────────────────────────────────────"
echo ""

if $SHOW_ENDPOINT; then
    show_endpoint
    exit 0
fi

check_ox
show_endpoint
echo ""

check_daemon
echo ""

generate_events "$COUNT"
echo ""

show_stats
echo ""

if $QUICK; then
    log_info "Quick mode - not waiting for flush"
    log_info "Events will be sent within ${FLUSH_INTERVAL}s"
else
    # if we generated enough to trigger early flush, check immediately
    if [[ $COUNT -ge $FLUSH_THRESHOLD ]]; then
        sleep 2  # give daemon a moment
        show_stats
        echo ""
    fi

    wait_for_flush
    echo ""
    show_stats
fi

echo ""
log_ok "Done! Check server logs or database for received events."
echo ""
echo "To verify in database:"
echo "  SELECT ts, kind, input_redacted FROM friction_events ORDER BY created_at DESC LIMIT 10;"
echo ""
