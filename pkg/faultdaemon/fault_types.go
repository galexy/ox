// Package faultdaemon provides a generic fault injection daemon for testing IPC.
// It can simulate various failure modes like hangs, corrupt responses, partial writes,
// etc. without being tied to any specific protocol.
//
// Protocol-specific response generation is handled via the ResponseHandler callback.
// This allows the same fault injection infrastructure to be used across different
// projects that share similar IPC patterns.
package faultdaemon

import "time"

// Fault represents a specific failure mode that can be injected.
type Fault string

const (
	// FaultNone is the default - daemon behaves normally.
	FaultNone Fault = ""

	// FaultHangOnAccept accepts connections but never reads from them.
	// Simulates a daemon stuck before even reading the request.
	FaultHangOnAccept Fault = "hang_on_accept"

	// FaultHangBeforeResponse reads the request but never responds.
	// Simulates a handler that's stuck/deadlocked.
	FaultHangBeforeResponse Fault = "hang_before_response"

	// FaultSlowResponse responds after a configurable delay.
	// Tests timeout boundary conditions.
	FaultSlowResponse Fault = "slow_response"

	// FaultCorruptResponse sends garbage bytes instead of valid data.
	// Tests protocol error handling.
	FaultCorruptResponse Fault = "corrupt_response"

	// FaultPartialResponse sends an incomplete message (no newline).
	// Tests partial read handling.
	FaultPartialResponse Fault = "partial_response"

	// FaultCloseImmediately accepts then immediately closes the connection.
	// Simulates daemon crash mid-connection.
	FaultCloseImmediately Fault = "close_immediately"

	// FaultCloseAfterRead reads the request then closes without responding.
	// Simulates daemon crash after receiving request.
	FaultCloseAfterRead Fault = "close_after_read"

	// FaultDropConnection drops every Nth connection silently.
	// Tests retry logic.
	FaultDropConnection Fault = "drop_connection"

	// FaultPanicInHandler panics while handling the request.
	// Tests panic recovery.
	FaultPanicInHandler Fault = "panic_in_handler"

	// FaultDeadlock acquires a mutex and never releases it.
	// Simulates a real deadlock scenario.
	FaultDeadlock Fault = "deadlock"

	// FaultMultipleResponses sends two complete responses instead of one.
	// Tests protocol desync - client reads first, second corrupts next request.
	FaultMultipleResponses Fault = "multiple_responses"

	// FaultResponseWithoutNewline sends valid data but no trailing newline.
	// Client hangs waiting for delimiter (different from partial/corrupt data).
	FaultResponseWithoutNewline Fault = "response_without_newline"

	// FaultChunkedResponse sends response one byte at a time with delays.
	// Tests that buffered reader handles fragmented reads correctly.
	FaultChunkedResponse Fault = "chunked_response"

	// FaultSlowAccept accepts connection but waits before reading request.
	// Simulates daemon under load - accept succeeds but handler delayed.
	FaultSlowAccept Fault = "slow_accept"

	// FaultRefuseAfterAccept accepts then immediately closes (connection refused after accept).
	// Different from CloseImmediately - this simulates backlog exhaustion.
	FaultRefuseAfterAccept Fault = "refuse_after_accept"

	// FaultVerySlowResponse responds after 15+ seconds.
	// Tests that client enforces reasonable timeouts on long operations.
	FaultVerySlowResponse Fault = "very_slow_response"

	// FaultResponseTooLarge sends a response exceeding typical size limits (1MB+).
	// Tests client size limit enforcement.
	FaultResponseTooLarge Fault = "response_too_large"

	// FaultInvalidJSON sends syntactically invalid JSON (unescaped control chars).
	// Tests unmarshal error handling.
	FaultInvalidJSON Fault = "invalid_json"

	// FaultEmbeddedNewlines sends valid JSON with newlines in string values.
	// Tests that client uses JSON framing, not naive line splitting.
	FaultEmbeddedNewlines Fault = "embedded_newlines"

	// FaultWriteHalfThenHang writes half the response then hangs forever.
	// Tests partial write recovery / timeout during write phase.
	FaultWriteHalfThenHang Fault = "write_half_then_hang"
)

// ResponseHandler generates a response for a given request.
// The request is the raw bytes received from the client (including newline).
// Return the response bytes to send (should include trailing newline for NDJSON).
// Return nil to use the default behavior (echo the request back).
type ResponseHandler func(request []byte) []byte

// FaultMatcher determines if a fault should apply to a given request.
// Return true to apply the fault, false to respond normally.
// If nil, the fault applies to all requests.
type FaultMatcher func(request []byte) bool

// Config configures fault injection behavior.
type Config struct {
	// Fault is the primary fault mode to inject.
	Fault Fault

	// SlowResponseDelay is the delay for FaultSlowResponse and FaultSlowAccept.
	SlowResponseDelay time.Duration

	// DropEveryN drops every Nth connection for FaultDropConnection.
	DropEveryN int

	// ResponseHandler generates protocol-specific responses.
	// If nil, requests are echoed back.
	ResponseHandler ResponseHandler

	// FaultMatcher determines if the fault applies to a specific request.
	// If nil, the fault applies to all requests.
	FaultMatcher FaultMatcher

	// LargeSizeBytes is the size for FaultResponseTooLarge (default 2MB).
	LargeSizeBytes int
}

// RPCCall records a call made to the fault daemon.
type RPCCall struct {
	// Request is the raw request bytes received.
	Request []byte

	// Timestamp is when the call was received.
	Timestamp time.Time
}
