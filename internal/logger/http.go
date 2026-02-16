package logger

import (
	"time"
)

// LogHTTPRequest logs an outbound HTTP request at debug level
func LogHTTPRequest(method, url string, attrs ...any) {
	args := []any{"method", method, "url", url}
	args = append(args, attrs...)
	Debug("http request", args...)
}

// LogHTTPResponse logs an HTTP response with timing at debug level
func LogHTTPResponse(method, url string, status int, duration time.Duration, attrs ...any) {
	args := []any{
		"method", method,
		"url", url,
		"status", status,
		"duration_ms", duration.Milliseconds(),
	}
	args = append(args, attrs...)
	Debug("http response", args...)
}

// LogHTTPError logs an HTTP request error with timing
func LogHTTPError(method, url string, err error, duration time.Duration) {
	Debug("http error",
		"method", method,
		"url", url,
		"error", err.Error(),
		"duration_ms", duration.Milliseconds(),
	)
}

// LogHTTPRequestBody logs the request body at debug level
func LogHTTPRequestBody(body string) {
	Debug("http request body", "body", body)
}

// LogHTTPResponseBody logs the response body at debug level
func LogHTTPResponseBody(body string) {
	Debug("http response body", "body", body)
}
