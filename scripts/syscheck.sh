#!/bin/bash
# syscheck.sh - Quick system resource diagnostic for macOS
# Usage: ./syscheck.sh [--kill-idle-claude]

set -euo pipefail

RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BOLD='\033[1m'
NC='\033[0m'

warn() { echo -e "${YELLOW}⚠${NC}  $1"; }
bad() { echo -e "${RED}✗${NC}  $1"; }
good() { echo -e "${GREEN}✓${NC}  $1"; }
header() { echo -e "\n${BOLD}$1${NC}"; }

# Get core count for load average comparison
CORES=$(sysctl -n hw.ncpu)

header "System Overview"
echo "Cores: $CORES"

# Load average
LOAD=$(sysctl -n vm.loadavg | awk '{print $2}')
LOAD_INT=${LOAD%.*}
if (( LOAD_INT > CORES * 2 )); then
    bad "Load average: $LOAD (very high, >2x cores)"
elif (( LOAD_INT > CORES )); then
    warn "Load average: $LOAD (elevated, >cores)"
else
    good "Load average: $LOAD"
fi

# Memory
MEM_USED=$(vm_stat | awk '/Pages active/ {active=$3} /Pages wired/ {wired=$4} /compressor/ {comp=$5} END {printf "%.0f", (active+wired+comp)*16384/1024/1024/1024}')
MEM_FREE=$(vm_stat | awk '/Pages free/ {print int($3*16384/1024/1024)}')
if (( MEM_FREE < 500 )); then
    bad "Memory: ${MEM_USED}GB used, ${MEM_FREE}MB free (critical)"
elif (( MEM_FREE < 2000 )); then
    warn "Memory: ${MEM_USED}GB used, ${MEM_FREE}MB free (low)"
else
    good "Memory: ${MEM_USED}GB used, ${MEM_FREE}MB free"
fi

# Compressor pressure (pages occupied by compressor)
COMPRESSOR_PAGES=$(vm_stat | awk '/Pages occupied by compressor/ {gsub(/\./,"",$NF); print $NF}')
COMPRESSOR_GB=$(awk "BEGIN {printf \"%.1f\", ${COMPRESSOR_PAGES:-0} * 16384 / 1024 / 1024 / 1024}")
COMPRESSOR_INT=${COMPRESSOR_GB%.*}
if (( COMPRESSOR_INT > 4 )); then
    bad "Memory compressor: ${COMPRESSOR_GB}GB (heavy pressure)"
elif (( COMPRESSOR_INT > 1 )); then
    warn "Memory compressor: ${COMPRESSOR_GB}GB (moderate pressure)"
else
    good "Memory compressor: ${COMPRESSOR_GB}GB"
fi

header "Top CPU Consumers"
ps -Ao pid,pcpu,comm | sort -k2 -rn | head -8 | tail -7 | while read pid cpu comm; do
    name=$(basename "$comm")
    if (( ${cpu%.*} > 50 )); then
        bad "$name (PID $pid): ${cpu}%"
    elif (( ${cpu%.*} > 20 )); then
        warn "$name (PID $pid): ${cpu}%"
    else
        echo "   $name (PID $pid): ${cpu}%"
    fi
done

header "Top Memory Consumers"
ps -Ao pid,rss,comm | sort -k2 -rn | head -8 | tail -7 | while read pid rss comm; do
    name=$(basename "$comm")
    mb=$((rss / 1024))
    if (( mb > 500 )); then
        bad "$name (PID $pid): ${mb}MB"
    elif (( mb > 200 )); then
        warn "$name (PID $pid): ${mb}MB"
    else
        echo "   $name (PID $pid): ${mb}MB"
    fi
done

header "Process Counts"
CLAUDE_COUNT=$(pgrep -f "claude" 2>/dev/null | wc -l | tr -d ' ')
NODE_COUNT=$(pgrep -f "node" 2>/dev/null | wc -l | tr -d ' ')
DOCKER_RUNNING=$(docker ps -q 2>/dev/null | wc -l | tr -d ' ' || echo "0")

# Claude is the workload, not the problem
CLAUDE_MEM=$(ps -Ao rss,comm | grep claude | awk '{sum+=$1} END {print int(sum/1024)}')
good "Claude agents: $CLAUDE_COUNT (using ~${CLAUDE_MEM}MB)"

if (( NODE_COUNT > 20 )); then
    warn "Node processes: $NODE_COUNT (typical for dev, but high)"
else
    good "Node processes: $NODE_COUNT"
fi

if (( DOCKER_RUNNING > 0 )); then
    warn "Docker containers: $DOCKER_RUNNING running"
    docker ps --format "   {{.Names}}: {{.Status}}" 2>/dev/null || true
else
    good "Docker containers: none running"
fi

header "Disk Space"
DISK_USED=$(df -h / | tail -1 | awk '{print $5}' | tr -d '%')
if (( DISK_USED > 90 )); then
    bad "Root volume: ${DISK_USED}% used"
elif (( DISK_USED > 80 )); then
    warn "Root volume: ${DISK_USED}% used"
else
    good "Root volume: ${DISK_USED}% used"
fi

# Optional: kill idle claude processes
if [[ "${1:-}" == "--kill-idle-claude" ]]; then
    header "Killing idle Claude processes..."
    # This is aggressive - only kills claude processes not attached to current terminal
    CURRENT_TTY=$(tty 2>/dev/null | sed 's|/dev/||' || echo "")
    for pid in $(pgrep -f "claude" 2>/dev/null); do
        proc_tty=$(ps -o tty= -p "$pid" 2>/dev/null | tr -d ' ')
        if [[ "$proc_tty" != "$CURRENT_TTY" && "$proc_tty" != "??" ]]; then
            echo "Would kill PID $pid (tty: $proc_tty)"
            # Uncomment to actually kill: kill "$pid"
        fi
    done
    echo "(Dry run - uncomment kill line in script to actually terminate)"
fi

header "Claude Capacity Analysis"
# Estimate: each Claude agent uses ~300MB RAM
CLAUDE_AVG_MB=300
TOTAL_MEM_MB=$(sysctl -n hw.memsize | awk '{print int($1/1024/1024)}')
RESERVED_MB=2000  # leave 2GB for system
AVAILABLE_FOR_CLAUDE=$((TOTAL_MEM_MB - RESERVED_MB))
MAX_CLAUDE_THEORETICAL=$((AVAILABLE_FOR_CLAUDE / CLAUDE_AVG_MB))

# What's eating into Claude capacity?
DOCKER_MEM=$(ps -Ao rss,comm | grep -E "(docker|com.docker)" | awk '{sum+=$1} END {print int(sum/1024)}')
BROWSER_MEM=$(ps -Ao rss,comm | grep -iE "(brave|chrome|firefox|safari)" | awk '{sum+=$1} END {print int(sum/1024)}')
NODE_NON_CLAUDE=$(ps -Ao rss,comm | grep node | awk '{sum+=$1} END {print int(sum/1024)}')

echo "Total RAM: ${TOTAL_MEM_MB}MB | Reserved for system: ${RESERVED_MB}MB"
echo "Theoretical max Claude agents: ~$MAX_CLAUDE_THEORETICAL (at ${CLAUDE_AVG_MB}MB each)"
echo ""
echo "Resources competing with Claude:"
if (( DOCKER_MEM > 500 )); then
    warn "Docker: ${DOCKER_MEM}MB - stop containers to free memory"
else
    echo "   Docker: ${DOCKER_MEM}MB"
fi
if (( BROWSER_MEM > 500 )); then
    warn "Browser: ${BROWSER_MEM}MB - close tabs to free memory"
else
    echo "   Browser: ${BROWSER_MEM}MB"
fi
if (( NODE_NON_CLAUDE > 500 )); then
    warn "Node.js: ${NODE_NON_CLAUDE}MB - check VS Code extensions"
else
    echo "   Node.js: ${NODE_NON_CLAUDE}MB"
fi

# Potential freed capacity
POTENTIAL_FREE=$((DOCKER_MEM + BROWSER_MEM))
POTENTIAL_MORE_CLAUDE=$((POTENTIAL_FREE / CLAUDE_AVG_MB))
if (( POTENTIAL_MORE_CLAUDE > 0 )); then
    echo ""
    good "Stopping Docker + closing browser could free ~${POTENTIAL_FREE}MB (~${POTENTIAL_MORE_CLAUDE} more agents)"
fi

echo ""
