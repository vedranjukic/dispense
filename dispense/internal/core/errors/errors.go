package errors

import (
	"fmt"
)

// ErrorCode represents specific error types for programmatic handling
type ErrorCode string

const (
	// Configuration errors
	ErrCodeConfigInvalid       ErrorCode = "CONFIG_INVALID"
	ErrCodeAPIKeyMissing       ErrorCode = "API_KEY_MISSING"
	ErrCodeAPIKeyInvalid       ErrorCode = "API_KEY_INVALID"
	
	// Sandbox errors
	ErrCodeSandboxNotFound     ErrorCode = "SANDBOX_NOT_FOUND"
	ErrCodeSandboxExists       ErrorCode = "SANDBOX_EXISTS"
	ErrCodeSandboxCreateFailed ErrorCode = "SANDBOX_CREATE_FAILED"
	ErrCodeSandboxDeleteFailed ErrorCode = "SANDBOX_DELETE_FAILED"
	ErrCodeSandboxNotReady     ErrorCode = "SANDBOX_NOT_READY"
	
	// Provider errors
	ErrCodeProviderUnavailable ErrorCode = "PROVIDER_UNAVAILABLE"
	ErrCodeProviderAuthFailed  ErrorCode = "PROVIDER_AUTH_FAILED"
	
	// Task/Project errors
	ErrCodeTaskInvalid         ErrorCode = "TASK_INVALID"
	ErrCodeProjectNotFound     ErrorCode = "PROJECT_NOT_FOUND"
	ErrCodeGitOperationFailed  ErrorCode = "GIT_OPERATION_FAILED"
	
	// Validation errors
	ErrCodeValidationFailed    ErrorCode = "VALIDATION_FAILED"
	ErrCodeInputInvalid        ErrorCode = "INPUT_INVALID"
	
	// System errors
	ErrCodeSystemUnavailable   ErrorCode = "SYSTEM_UNAVAILABLE"
	ErrCodeDaemonUnavailable   ErrorCode = "DAEMON_UNAVAILABLE"
)

// DispenseError represents structured errors with context
type DispenseError struct {
	Code      ErrorCode              `json:"code"`
	Message   string                 `json:"message"`
	Details   string                 `json:"details,omitempty"`
	Cause     error                  `json:"-"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

func (e *DispenseError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *DispenseError) Unwrap() error {
	return e.Cause
}

// WithContext adds context information to the error
func (e *DispenseError) WithContext(key string, value interface{}) *DispenseError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// New creates a new DispenseError
func New(code ErrorCode, message string) *DispenseError {
	return &DispenseError{
		Code:    code,
		Message: message,
	}
}

// NewWithDetails creates a new DispenseError with details
func NewWithDetails(code ErrorCode, message, details string) *DispenseError {
	return &DispenseError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// Wrap wraps an existing error with a DispenseError
func Wrap(err error, code ErrorCode, message string) *DispenseError {
	return &DispenseError{
		Code:    code,
		Message: message,
		Cause:   err,
	}
}

// WrapWithDetails wraps an existing error with a DispenseError and details
func WrapWithDetails(err error, code ErrorCode, message, details string) *DispenseError {
	return &DispenseError{
		Code:    code,
		Message: message,
		Details: details,
		Cause:   err,
	}
}

// Is checks if the error matches the given code
func Is(err error, code ErrorCode) bool {
	if dispenseErr, ok := err.(*DispenseError); ok {
		return dispenseErr.Code == code
	}
	return false
}

// GetCode extracts the error code from an error, returns empty string if not a DispenseError
func GetCode(err error) ErrorCode {
	if dispenseErr, ok := err.(*DispenseError); ok {
		return dispenseErr.Code
	}
	return ""
}