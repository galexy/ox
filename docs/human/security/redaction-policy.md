<!-- doc-audience: human -->
# Redaction Policy

This document describes what sensitive data ox automatically redacts from sessions
and other captured agent data.

## Overview

When ox captures sessions or processes agent output, it automatically scans for
and redacts secrets, credentials, and other sensitive data. Redacted content is
replaced with descriptive tokens like `[REDACTED_AWS_KEY]` so you know what was
removed without exposing the actual secret.

## Why This Matters

AI coding agents see everything in your terminal and files. When ox captures
sessions for learning or review, it **must not** leak:

- API keys and tokens
- Database credentials
- Private keys
- Connection strings with embedded passwords
- Authentication headers

## What Gets Redacted

| Category | Examples | Replacement Token |
|----------|----------|-------------------|
| **AWS** | `AKIA...` access keys, secret keys | `[REDACTED_AWS_KEY]`, `[REDACTED_AWS_SECRET]` |
| **GitHub** | `ghp_`, `gho_`, `ghs_`, `ghr_`, `ghu_` tokens | `[REDACTED_GITHUB_TOKEN]` |
| **GitHub PAT** | `github_pat_...` fine-grained tokens | `[REDACTED_GITHUB_PAT]` |
| **GitLab** | `glpat-...` tokens | `[REDACTED_GITLAB_TOKEN]` |
| **Slack** | `xoxb-`, `xoxp-`, `xoxa-` tokens | `[REDACTED_SLACK_TOKEN]` |
| **Stripe** | `sk_live_`, `sk_test_`, `pk_*` keys | `[REDACTED_STRIPE_KEY]` |
| **Twilio** | `SK...` API keys | `[REDACTED_TWILIO_KEY]` |
| **SendGrid** | `SG....` API keys | `[REDACTED_SENDGRID_KEY]` |
| **Mailchimp** | `...-us##` API keys | `[REDACTED_MAILCHIMP_KEY]` |
| **NPM** | `npm_...` tokens | `[REDACTED_NPM_TOKEN]` |
| **PyPI** | `pypi-...` tokens | `[REDACTED_PYPI_TOKEN]` |
| **UUIDs** | Heroku keys, other UUID secrets | `[REDACTED_UUID]` |
| **Private Keys** | RSA, DSA, EC, OpenSSH, PGP | `[REDACTED_PRIVATE_KEY]` |
| **Connection Strings** | `postgres://user:pass@host` | `[REDACTED_CONNECTION_STRING]` |
| **Bearer Tokens** | `Authorization: Bearer ...` | `[REDACTED_BEARER_TOKEN]` |
| **Basic Auth** | `Authorization: Basic ...` | `[REDACTED_BASIC_AUTH]` |
| **JWT Tokens** | `eyJ...` three-part tokens | `[REDACTED_JWT]` |
| **Generic Secrets** | `api_key=`, `secret=`, `password=` | `[REDACTED_API_KEY]`, `[REDACTED_SECRET]`, `[REDACTED_PASSWORD]` |
| **Exported Env Vars** | `export AWS_SECRET_ACCESS_KEY=...` | `[REDACTED_EXPORT]` |

## Signature Verification

The redaction patterns are compiled into the ox binary and **cryptographically signed**
during the release process. This prevents tampering:

1. **At build time**: A deterministic manifest of all patterns is signed with Ed25519
2. **At runtime**: `ox redaction verify` re-generates the manifest and verifies the signature
3. **Public key**: Embedded in the binary, verifiable against SageOx releases

Run `ox redaction verify` to confirm your binary hasn't been tampered with.

## Inspecting the Policy

```bash
# View the complete pattern list
ox redaction policy

# Verify signature integrity
ox redaction verify

# JSON output for automation
ox redaction policy --json
```

## Adding Custom Patterns

Custom redaction patterns can be added per-project (coming soon). For now,
the default patterns cover most common secrets.

If you find a secret type that should be redacted but isn't, please
[open an issue](https://github.com/sageox/ox/issues).

## Security Team Review

This policy is designed for security team review:

1. **Transparent**: All patterns are documented and inspectable
2. **Deterministic**: Same patterns always produce same manifest hash
3. **Signed**: Tamper-evident via Ed25519 signatures
4. **Auditable**: `ox redaction policy --json` outputs machine-readable format

The canonical pattern definitions live in `internal/session/secrets.go`.
