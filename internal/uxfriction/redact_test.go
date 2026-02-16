package uxfriction

import (
	"testing"
)

func TestRedactInput(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "empty args",
			args:     []string{},
			expected: "",
		},
		{
			name:     "single arg no secrets",
			args:     []string{"hello"},
			expected: "hello",
		},
		{
			name:     "multiple args no secrets",
			args:     []string{"git", "commit", "-m", "fix bug"},
			expected: "git commit -m fix bug",
		},
		{
			name:     "aws access key redacted",
			args:     []string{"export", "AWS_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE"},
			expected: "export AWS_ACCESS_KEY=[REDACTED_AWS_KEY]",
		},
		{
			name:     "github token redacted",
			args:     []string{"--token", "ghp_abcdefghijklmnopqrstuvwxyz1234567890"},
			expected: "--token [REDACTED_GITHUB_TOKEN]",
		},
		{
			name:     "github fine-grained pat redacted",
			args:     []string{"auth", "github_pat_abcdefghijklmnopqrstuv"},
			expected: "auth [REDACTED_GITHUB_PAT]",
		},
		{
			name:     "gitlab token redacted",
			args:     []string{"--gitlab-token", "glpat-abcdefghijklmnopqrstuv"},
			expected: "--gitlab-token [REDACTED_GITLAB_TOKEN]",
		},
		{
			name:     "slack token redacted",
			args:     []string{"slack-notify", "xoxb-1234567890-abcdefgh"},
			expected: "slack-notify [REDACTED_SLACK_TOKEN]",
		},
		{
			name:     "stripe key redacted",
			args:     []string{"--key", "sk_live_xxFAKEKEYxxxxxxxxxxxxxxx"},
			expected: "--key [REDACTED_STRIPE_KEY]",
		},
		{
			name:     "jwt token redacted",
			args:     []string{"--jwt", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"},
			expected: "--jwt [REDACTED_JWT]",
		},
		{
			name:     "api key pattern redacted",
			args:     []string{"curl", "-H", "api_key=abcdefghijklmnopqrstuvwxyz"},
			expected: "curl -H [REDACTED_API_KEY]",
		},
		{
			name:     "connection string redacted",
			args:     []string{"postgres://user:password@localhost:5432/mydb"},
			expected: "[REDACTED_CONNECTION_STRING]",
		},
		{
			name:     "multiple secrets redacted",
			args:     []string{"AKIAIOSFODNN7EXAMPLE", "ghp_abcdefghijklmnopqrstuvwxyz1234567890"},
			expected: "[REDACTED_AWS_KEY] [REDACTED_GITHUB_TOKEN]",
		},
		{
			name:     "mixed content with secrets",
			args:     []string{"deploy", "--token=ghp_abcdefghijklmnopqrstuvwxyz1234567890", "--env=prod"},
			expected: "deploy --token=[REDACTED_GITHUB_TOKEN] --env=prod",
		},
		{
			name:     "bearer token redacted",
			args:     []string{"authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"},
			expected: "[REDACTED_BEARER_TOKEN]",
		},
		{
			name:     "private key header redacted",
			args:     []string{"-----BEGIN RSA PRIVATE KEY-----"},
			expected: "[REDACTED_PRIVATE_KEY]",
		},
		{
			name:     "npm token redacted",
			args:     []string{"npm_abcdefghijklmnopqrstuvwxyz1234567890"},
			expected: "[REDACTED_NPM_TOKEN]",
		},
		{
			name:     "password pattern redacted",
			args:     []string{"password=\"supersecret123\""},
			expected: "[REDACTED_PASSWORD]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactInput(tt.args)
			if result != tt.expected {
				t.Errorf("RedactInput() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRedactError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		maxLen   int
		expected string
	}{
		{
			name:     "empty error message",
			errMsg:   "",
			maxLen:   100,
			expected: "",
		},
		{
			name:     "short message no secrets",
			errMsg:   "connection failed",
			maxLen:   100,
			expected: "connection failed",
		},
		{
			name:     "message with github token",
			errMsg:   "auth failed with token ghp_abcdefghijklmnopqrstuvwxyz1234567890",
			maxLen:   100,
			expected: "auth failed with token [REDACTED_GITHUB_TOKEN]",
		},
		{
			name:     "message with aws key",
			errMsg:   "invalid credentials: AKIAIOSFODNN7EXAMPLE",
			maxLen:   100,
			expected: "invalid credentials: [REDACTED_AWS_KEY]",
		},
		{
			name:     "truncation without secrets",
			errMsg:   "this is a very long error message that should be truncated",
			maxLen:   20,
			expected: "this is a very long ",
		},
		{
			name:     "truncation with secrets",
			errMsg:   "failed: AKIAIOSFODNN7EXAMPLE and more text here",
			maxLen:   32,
			expected: "failed: [REDACTED_AWS_KEY] and m",
		},
		{
			name:     "zero maxLen no truncation",
			errMsg:   "this message should not be truncated at all",
			maxLen:   0,
			expected: "this message should not be truncated at all",
		},
		{
			name:     "negative maxLen no truncation",
			errMsg:   "negative maxLen should not truncate",
			maxLen:   -1,
			expected: "negative maxLen should not truncate",
		},
		{
			name:     "message shorter than maxLen",
			errMsg:   "short",
			maxLen:   100,
			expected: "short",
		},
		{
			name:     "message exactly at maxLen",
			errMsg:   "exactly10!",
			maxLen:   10,
			expected: "exactly10!",
		},
		{
			name:     "connection string in error",
			errMsg:   "failed to connect: postgres://admin:secret@db.example.com:5432/prod",
			maxLen:   200,
			expected: "failed to connect: [REDACTED_CONNECTION_STRING]",
		},
		{
			name:     "jwt in error message",
			errMsg:   "token expired: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
			maxLen:   200,
			expected: "token expired: [REDACTED_JWT]",
		},
		{
			name:     "multiple secrets with truncation",
			errMsg:   "auth AKIAIOSFODNN7EXAMPLE and ghp_abcdefghijklmnopqrstuvwxyz1234567890 failed",
			maxLen:   50,
			expected: "auth [REDACTED_AWS_KEY] and [REDACTED_GITHUB_TOKEN",
		},
		{
			name:     "private key in error",
			errMsg:   "invalid key: -----BEGIN RSA PRIVATE KEY-----",
			maxLen:   100,
			expected: "invalid key: [REDACTED_PRIVATE_KEY]",
		},
		{
			name:     "stripe key in error",
			errMsg:   "payment failed with key sk_live_xxFAKEKEYxxxxxxxxxxxxxxx",
			maxLen:   200,
			expected: "payment failed with key [REDACTED_STRIPE_KEY]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactError(tt.errMsg, tt.maxLen)
			if result != tt.expected {
				t.Errorf("RedactError() = %q, want %q", result, tt.expected)
			}
		})
	}
}
