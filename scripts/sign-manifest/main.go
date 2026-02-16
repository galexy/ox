// sign-manifest signs ox artifacts and updates the release-signatures.json file.
//
// Usage:
//
//	# Sign artifacts and update .github/release-signatures.json
//	SAGEOX_CLI_SIGNING_KEY=<base64-private-key> go run ./scripts/sign-manifest
//
//	# Generate a new key pair
//	go run ./scripts/sign-manifest --generate-keys
//
//	# List registered artifacts
//	go run ./scripts/sign-manifest --list
//
// The signatures file is checked into the repo and embedded at build time.
package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sageox/ox/internal/signing"
	// import session to register the redaction artifact
	_ "github.com/sageox/ox/internal/session"
)

const signaturesFile = ".github/release-signatures.json"

// SignaturesFile is the JSON structure for the checked-in signatures file.
type SignaturesFile struct {
	Comment       string            `json:"_comment"`
	SchemaVersion string            `json:"schema_version"`
	PublicKey     string            `json:"public_key"`
	Signatures    map[string]string `json:"signatures"`
}

func main() {
	generateKeys := flag.Bool("generate-keys", false, "Generate a new Ed25519 key pair")
	list := flag.Bool("list", false, "List registered artifacts and their hashes")
	verify := flag.Bool("verify", false, "Verify signatures in the file match current artifacts")
	flag.Parse()

	if *generateKeys {
		generateKeyPair()
		return
	}

	if *list {
		listArtifacts()
		return
	}

	if *verify {
		verifySignatures()
		return
	}

	signAndUpdate()
}

func generateKeyPair() {
	pub, priv, err := signing.GenerateKeyPair()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to generate key pair: %v\n", err)
		os.Exit(1)
	}

	pubB64 := base64.StdEncoding.EncodeToString(pub)
	privB64 := base64.StdEncoding.EncodeToString(priv)

	fmt.Println("# SageOx CLI Release Signing Key Pair")
	fmt.Println("#")
	fmt.Println("# PUBLIC KEY (checked into repo in release-signatures.json):")
	fmt.Printf("SAGEOX_CLI_PUBLIC_KEY=%s\n", pubB64)
	fmt.Println()
	fmt.Println("# PRIVATE KEY (keep secret, use to sign releases):")
	fmt.Printf("SAGEOX_CLI_SIGNING_KEY=%s\n", privB64)
	fmt.Println()
	fmt.Println("# Store the private key securely. Only maintainers need it.")
}

func listArtifacts() {
	artifacts := signing.ListArtifacts()
	sort.Strings(artifacts)

	fmt.Printf("Registered artifacts (%d):\n\n", len(artifacts))
	for _, name := range artifacts {
		result := signing.Verify(name)
		fmt.Printf("  %s\n", name)
		if result.Hash != "" {
			fmt.Printf("    hash: %s\n", result.Hash)
		}
		if result.Error != nil {
			fmt.Printf("    error: %v\n", result.Error)
		}
		fmt.Println()
	}
}

func verifySignatures() {
	// read current signatures file
	data, err := os.ReadFile(signaturesFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot read %s: %v\n", signaturesFile, err)
		os.Exit(1)
	}

	var sigFile SignaturesFile
	if err := json.Unmarshal(data, &sigFile); err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid JSON in %s: %v\n", signaturesFile, err)
		os.Exit(1)
	}

	if sigFile.PublicKey == "" {
		fmt.Println("No signatures configured yet.")
		fmt.Println("Run with SAGEOX_CLI_SIGNING_KEY to sign artifacts.")
		return
	}

	// temporarily set embedded values for verification
	oldPub := signing.EmbeddedPublicKey
	oldSigs := signing.EmbeddedSignatures
	defer func() {
		signing.EmbeddedPublicKey = oldPub
		signing.EmbeddedSignatures = oldSigs
	}()

	signing.EmbeddedPublicKey = sigFile.PublicKey

	// convert map to semicolon format
	var sigPairs []string
	for name, sig := range sigFile.Signatures {
		sigPairs = append(sigPairs, name+":"+sig)
	}
	sort.Strings(sigPairs)
	signing.EmbeddedSignatures = joinStrings(sigPairs, ";")

	// verify each artifact
	artifacts := signing.ListArtifacts()
	sort.Strings(artifacts)

	allValid := true
	for _, name := range artifacts {
		result := signing.Verify(name)
		switch result.Status {
		case signing.StatusValid:
			fmt.Printf("  %s: valid\n", name)
		case signing.StatusInvalid:
			fmt.Printf("  %s: INVALID (signature mismatch)\n", name)
			allValid = false
		case signing.StatusMissing:
			fmt.Printf("  %s: MISSING (no signature in file)\n", name)
			allValid = false
		default:
			fmt.Printf("  %s: error - %v\n", name, result.Error)
			allValid = false
		}
	}

	if !allValid {
		fmt.Println()
		fmt.Println("Some signatures are invalid or missing.")
		fmt.Println("Run with SAGEOX_CLI_SIGNING_KEY to re-sign.")
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("All signatures valid.")
}

func signAndUpdate() {
	// get private key from environment
	privKeyB64 := os.Getenv("SAGEOX_CLI_SIGNING_KEY")
	if privKeyB64 == "" {
		fmt.Fprintln(os.Stderr, "error: SAGEOX_CLI_SIGNING_KEY not set")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Generate a key pair with: go run ./scripts/sign-manifest --generate-keys")
		os.Exit(1)
	}

	// decode private key
	privKeyBytes, err := base64.StdEncoding.DecodeString(privKeyB64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid private key: %v\n", err)
		os.Exit(1)
	}

	if len(privKeyBytes) != ed25519.PrivateKeySize {
		fmt.Fprintf(os.Stderr, "error: invalid private key size\n")
		os.Exit(1)
	}

	privateKey := ed25519.PrivateKey(privKeyBytes)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	pubKeyB64 := base64.StdEncoding.EncodeToString(publicKey)

	// get all registered artifacts
	artifactNames := signing.ListArtifacts()
	sort.Strings(artifactNames)

	if len(artifactNames) == 0 {
		fmt.Fprintln(os.Stderr, "error: no artifacts registered")
		os.Exit(1)
	}

	// sign each artifact
	signatures := make(map[string]string)
	fmt.Printf("Signing %d artifact(s):\n\n", len(artifactNames))

	for _, name := range artifactNames {
		manifestBytes, err := signing.GetManifestBytes(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s: %v\n", name, err)
			os.Exit(1)
		}

		sig := signing.Sign(privateKey, manifestBytes)
		sigB64 := base64.StdEncoding.EncodeToString(sig)
		signatures[name] = sigB64

		fmt.Printf("  %s\n", name)
		fmt.Printf("    hash: %s\n", signing.Hash(manifestBytes))
		fmt.Println()
	}

	// write signatures file
	sigFile := SignaturesFile{
		Comment:       "SageOx CLI release signatures - regenerate with: SAGEOX_CLI_SIGNING_KEY=... go run ./scripts/sign-manifest",
		SchemaVersion: "1",
		PublicKey:     pubKeyB64,
		Signatures:    signatures,
	}

	data, err := json.MarshalIndent(sigFile, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to marshal JSON: %v\n", err)
		os.Exit(1)
	}

	// ensure directory exists
	dir := filepath.Dir(signaturesFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to create directory: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(signaturesFile, append(data, '\n'), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to write %s: %v\n", signaturesFile, err)
		os.Exit(1)
	}

	fmt.Printf("Updated %s\n", signaturesFile)
	fmt.Println()
	fmt.Println("Commit this file to the repo. The signatures will be embedded at build time.")
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}
	var b strings.Builder
	b.WriteString(strs[0])
	for i := 1; i < len(strs); i++ {
		b.WriteString(sep)
		b.WriteString(strs[i])
	}
	return b.String()
}
