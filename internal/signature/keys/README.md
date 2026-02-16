# Signature Keys

This directory contains documentation and development keys for SAGEOX.md signature verification.

## Key Format

Keys are embedded directly in `keys.go` as base64-encoded Ed25519 keys, indexed by key ID:

```go
var SageoxPublicKeys = map[string][]byte{
    "sageox-dev-2025-01": mustDecodeBase64("DIpRfK3WCVDc4A27r3WbIf+RmWxnRalISUaT55i0hdE="),
}
```

## Files

- `private.pem.dev` - Development private key (for testing only, matches `sageox-dev-2025-01`)
- `example_usage.md` - Example of how to sign and verify content

## Key ID Format

Key IDs follow the pattern: `sageox-{env}-{year}-{sequence}`

Examples:
- `sageox-dev-2025-01` - Development key, 2025, first key
- `sageox-prod-2025-01` - Production key, 2025, first key

## Development

The development private key matches the `sageox-dev-2025-01` public key embedded in `keys.go`.

To generate a new keypair for testing:
```bash
# generate keypair
openssl genpkey -algorithm Ed25519 -out private.pem
openssl pkey -in private.pem -pubout -outform DER | tail -c 32 | base64
```

## Key Rotation

To add a new public key:

1. Add the new key to `SageoxPublicKeys` map in `keys.go`
2. Use a new key ID (increment the sequence number)
3. Rebuild the binary
4. Server signs new content with the new key ID

Old keys remain in the map for verifying previously-signed content.

## Security Notes

- The private keys corresponding to production public keys are held securely by sageox.ai
- Never commit production private keys to version control
- The `.gitignore` is configured to exclude `*.pem.dev` files
