package api

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/fatih/color"
)

const (
	// HeaderMinVersion is returned by the server to indicate the minimum CLI version required
	HeaderMinVersion = "X-SageOx-Min-Version"

	// HeaderDeprecated is returned by the server to warn of upcoming deprecation
	HeaderDeprecated = "X-SageOx-Deprecated"
)

var (
	deprecationShown sync.Once // only show deprecation warning once per session
)

// CheckVersionResponse inspects HTTP response for version deprecation signals.
// Returns true if the CLI should abort due to unsupported version (HTTP 426).
// For successful responses, checks for soft deprecation warning header.
func CheckVersionResponse(resp *http.Response) bool {
	// hard block: 426 Upgrade Required
	if resp.StatusCode == http.StatusUpgradeRequired {
		minVersion := resp.Header.Get(HeaderMinVersion)
		PrintUpgradeRequired(minVersion)
		return true
	}

	// soft warning: deprecated but still functional
	if deprecated := resp.Header.Get(HeaderDeprecated); deprecated != "" {
		deprecationShown.Do(func() {
			PrintDeprecationWarning(deprecated)
		})
	}

	return false
}

// PrintUpgradeRequired displays a message indicating the CLI version is no longer supported
// Uses red color semantically indicating a blocking error
func PrintUpgradeRequired(minVersion string) {
	red := color.New(color.FgRed, color.Bold)
	redDim := color.New(color.FgRed)

	fmt.Fprintln(os.Stderr)
	red.Fprintln(os.Stderr, "  ✗ CLI Version No Longer Supported")
	fmt.Fprintln(os.Stderr)
	if minVersion != "" {
		redDim.Fprintf(os.Stderr, "  Minimum required version: %s\n", minVersion)
	}
	redDim.Fprintln(os.Stderr, "  Please upgrade: brew upgrade sageox/tap/ox")
	fmt.Fprintln(os.Stderr)
}

// PrintDeprecationWarning displays a warning that the CLI version is deprecated
// Uses yellow color semantically indicating a non-blocking warning
func PrintDeprecationWarning(message string) {
	yellow := color.New(color.FgYellow)
	if message != "" {
		yellow.Fprintf(os.Stderr, "  ⚠ Deprecation warning: %s\n", message)
	} else {
		yellow.Fprintln(os.Stderr, "  ⚠ This CLI version is deprecated. Please upgrade soon.")
	}
}
