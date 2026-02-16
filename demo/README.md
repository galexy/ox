# The AI-Native Team Context Demo

## What You're About to Witness

This isn't just another CLI tool demo. This is **the moment software engineering becomes multiplayer**.

Watch as:
1. A real production codebase gets analyzed in seconds
2. An AI coworker generates actionable findings
3. Those findings become tracked tasks—automatically
4. Multiple AI coworkers can now collaborate on fixing them

**This is team context meeting AI-as-teammate.**

---

## Why This Changes Everything

### Before: Engineering Knowledge Was Tribal

- Senior engineers hoarded patterns in their heads
- New team members spent months learning "how we do things here"
- Best practices lived in wikis nobody read
- Reviews caught issues after the damage was done

### After: Team Context Is Ambient

```bash
ox prime     # Every AI session starts with your team context
ox review    # Instant checklist against team standards
ox learn     # Extract patterns from existing codebases
```

Your team conventions are no longer documentation—they're **active participants** in every coding session.

---

## The Multiplayer Final Boss Level

Here's where it gets revolutionary. Picture an enterprise team working on the same codebase:

```
┌─────────────────────────────────────────────────────────────────┐
│                     ACME Corp Engineering                       │
│                     (same codebase, different features)         │
│                                                                 │
│      Maya               Jordan              Riley               │
│   (Auth Team)        (Payments Team)     (Platform Team)        │
│       │                    │                    │               │
│       └────────────────────┼────────────────────┘               │
│                            │                                    │
│                     ┌──────▼──────┐                             │
│                     │   SageOx    │                             │
│                     │  Guidance   │                             │
│                     └──────┬──────┘                             │
│                            │                                    │
│       ┌────────────────────┼────────────────────┐               │
│       │                    │                    │               │
│       ▼                    ▼                    ▼               │
│  Coding Agent        Coding Agent        Coding Agent           │
│  (via Cursor)        (via Claude)        (via Windsurf)         │
│       │                    │                    │               │
│       └────────────────────┼────────────────────┘               │
│                            │                                    │
│                     ┌──────▼──────┐                             │
│                     │   Beads     │                             │
│                     │   Tasks     │                             │
│                     └─────────────┘                             │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Three engineers. Three different AI coding agents. One source of truth.**

Maya uses Cursor, Jordan prefers Claude Code, Riley swears by Windsurf—but they all:
- Start every session with the same infrastructure context (`ox prime`), which happens automatically in supported coding agents.
- Generate findings that become tracked tasks (`bd create`)
- Sync tasks across the team via git (`bd sync`)
- See work queued by teammates' AI agents (`bd list`)

This isn't AI replacing engineers. This is **AI amplifying an entire engineering organization**.

---

## The Demo

### What We're Analyzing

[inbox-zero](https://github.com/elie222/inbox-zero) - A production Next.js application with:
- 5,000+ GitHub stars
- Real infrastructure patterns to discover
- Active development with complex deployment needs

### What Happens

1. **`ox init`** - SageOx injects team context into the repo
2. **`ox prime`** - Any AI coding agent loads your team context
3. **`ox review`** - Generate a comprehensive review against team standards
4. **Coding Agent** - AI reads the review and identifies improvements
5. **`bd create`** - AI files actionable tasks in beads
6. **`bd list`** - See the work queue ready for humans OR other AI agents

### The Groundbreaking Part

The tasks Maya's Cursor agent creates can be picked up by:
- Jordan (human) reviewing the backlog
- Riley's Windsurf agent in a different session
- An entirely different AI agent on CI
- An automated remediation pipeline

**Infrastructure improvements discovered by any AI, tracked for everyone, implementable by anyone (or anything).**

---

## Running the Demo

### Prerequisites

```bash
# VHS for recording
brew install vhs

# SOPS for credential encryption
brew install sops yq

# Node.js 18+ for browser automation
node --version  # Should be 18+
```

### Authentication Setup

The demo uses `you@yourcompany.com` - a dedicated demo account.

**Option 1: SOPS-encrypted credentials (recommended)**

1. Get SOPS age key from 1Password (search "SageOx SOPS Age Key")
2. Save to `~/.config/sops/age/keys.txt` with `chmod 600`
3. Create `demo/credentials.sops.yaml`:

```bash
# Copy template
cp demo/credentials.example.yaml demo/credentials.yaml
# Edit with real password
vim demo/credentials.yaml
# Encrypt
cd demo && sops -e credentials.yaml > credentials.sops.yaml
# Remove plaintext
rm credentials.yaml
```

**Option 2: Environment variables**

```bash
export DEMO_EMAIL="you@yourcompany.com"
export DEMO_PASSWORD="your-password"
```

### Setup & Record

```bash
# Full setup: build ox, authenticate, create demo environment
./demo/setup.sh

# Record the demo
vhs demo/demo.tape
```

### Options

```bash
# Just authenticate (skip repo setup)
./demo/setup.sh --auth-only

# Skip authentication (use existing session)
SKIP_AUTH=1 ./demo/setup.sh

# Show browser during login (debugging)
HEADLESS=false ./demo/setup.sh

# Clean up
./demo/setup.sh --clean
```

---

## Files

```
demo/
├── setup.sh              # Setup script (build, auth, environment)
├── demo.tape             # VHS tape for recording
├── demo.gif              # Output (generated)
├── .sops.yaml            # SOPS encryption config
├── credentials.sops.yaml # Encrypted credentials (not in git)
├── credentials.example.yaml # Credentials template
├── playwright/           # Browser automation
│   ├── package.json
│   ├── tsconfig.json
│   └── login.ts          # Automated login script
├── claude-demo.tape      # Extended demo with AI agent
└── README.md             # This file
```

---

## The Vision

Today: One AI coworker reviews one codebase and files tasks.

Tomorrow: **Fleets of AI coworkers** across your organization, all speaking the same language, all contributing to a shared backlog, all guided by your team conventions.

This is the foundation for:
- AI-driven code audits at scale
- Automated quality checking
- Cross-repo pattern enforcement
- Self-improving engineering pipelines

**Welcome to multiplayer AI-native engineering.**

---

## Learn More

- [SageOx Documentation](https://sageox.ai/docs)
- [Beads Task Tracking](https://sageox.ai/get/beads)
- [Enterprise Patterns](https://sageox.ai/enterprise)
