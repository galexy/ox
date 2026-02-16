package auth

import (
	"context"
	"fmt"
	"time"
)

// LoginResult contains the result of a login attempt
type LoginResult struct {
	Success  bool
	UserInfo *UserInfo
	Error    string
}

// DeviceAuthPendingError indicates the user hasn't authorized yet (not an error, used for control flow)
type DeviceAuthPendingError struct{}

func (e *DeviceAuthPendingError) Error() string {
	return "authorization_pending"
}

// DeviceAuthSlowDownError indicates the server requested slower polling
type DeviceAuthSlowDownError struct{}

func (e *DeviceAuthSlowDownError) Error() string {
	return "slow_down"
}

// DeviceAuthError represents a device authorization failure
type DeviceAuthError struct {
	Message string
}

func (e *DeviceAuthError) Error() string {
	return e.Message
}

// Login performs the full OAuth Device Authorization Flow with LoginResult return type
// This is a wrapper around the existing Login function that returns LoginResult
// instead of error for simpler handling in CLI code
func LoginWithResult(timeout time.Duration, onStatus func(string)) *LoginResult {
	// request device code
	deviceResp, err := RequestDeviceCode()
	if err != nil {
		return &LoginResult{
			Success: false,
			Error:   fmt.Sprintf("failed to request device code: %v", err),
		}
	}

	// create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// perform login
	err = Login(ctx, deviceResp, onStatus)
	if err != nil {
		return &LoginResult{
			Success: false,
			Error:   err.Error(),
		}
	}

	// get token to extract user info
	token, err := GetToken()
	if err != nil {
		return &LoginResult{
			Success: false,
			Error:   fmt.Sprintf("failed to get token after login: %v", err),
		}
	}

	if token == nil {
		return &LoginResult{
			Success: false,
			Error:   "no token found after successful login",
		}
	}

	return &LoginResult{
		Success:  true,
		UserInfo: &token.UserInfo,
	}
}
