package uxfriction

import (
	"strings"

	"github.com/sageox/ox/internal/session"
)

// RedactInput joins command arguments and redacts secrets.
func RedactInput(args []string) string {
	input := strings.Join(args, " ")
	redactor := session.NewRedactor()
	redacted, _ := redactor.RedactString(input)
	return redacted
}

// RedactError redacts secrets from an error message and truncates to maxLen.
func RedactError(errMsg string, maxLen int) string {
	if errMsg == "" {
		return ""
	}

	redactor := session.NewRedactor()
	redacted, _ := redactor.RedactString(errMsg)

	if maxLen > 0 && len(redacted) > maxLen {
		return redacted[:maxLen]
	}

	return redacted
}
