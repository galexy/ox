# Example Usage

This document demonstrates how to use the signature verification functions.

## Get the Public Key

```go
import "github.com/sageox/ox/internal/signature"

publicKey, err := signature.GetPublicKey()
if err != nil {
    log.Fatalf("Failed to get public key: %v", err)
}

fmt.Printf("Public key size: %d bytes\n", len(publicKey))
```

## Verify a Signature

```go
import "github.com/sageox/ox/internal/signature"

content := []byte("Hello, World!")
signature := []byte{...} // Ed25519 signature bytes

valid := signature.VerifySignature(content, signature)
if valid {
    fmt.Println("Signature is valid!")
} else {
    fmt.Println("Signature verification failed")
}
```

## Complete Example

```go
package main

import (
    "encoding/base64"
    "fmt"
    "log"

    "github.com/sageox/ox/internal/signature"
)

func main() {
    // content to verify
    content := []byte("This content was signed by sageox.ai")

    // signature from sageox.ai (base64 encoded)
    signatureB64 := "base64_encoded_signature_here"
    signatureBytes, err := base64.StdEncoding.DecodeString(signatureB64)
    if err != nil {
        log.Fatalf("Failed to decode signature: %v", err)
    }

    // verify the signature
    valid := signature.VerifySignature(content, signatureBytes)

    if valid {
        fmt.Println("Content verified: signature is valid")
    } else {
        fmt.Println("Warning: signature verification failed")
    }
}
```

## Notes

- The public key is embedded in the binary at build time
- Ed25519 signatures are 64 bytes
- The content should be the exact bytes that were signed (including whitespace)
- For production use, replace the development public key with the actual sageox.ai public key
