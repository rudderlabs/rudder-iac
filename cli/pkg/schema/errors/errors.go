package errors

import (
	"fmt"
	"time"
)

// ErrorType represents the category of error
type ErrorType string

const (
	// Fetch errors
	ErrorTypeFetchConnection ErrorType = "fetch_connection"
	ErrorTypeFetchAPI        ErrorType = "fetch_api"
	ErrorTypeFetchParsing    ErrorType = "fetch_parsing"
	ErrorTypeFetchTimeout    ErrorType = "fetch_timeout"

	// Processing errors
	ErrorTypeProcessValidation ErrorType = "process_validation"
	ErrorTypeProcessJSONPath   ErrorType = "process_jsonpath"
	ErrorTypeProcessTransform  ErrorType = "process_transform"

	// Conversion errors
	ErrorTypeConvertAnalysis   ErrorType = "convert_analysis"
	ErrorTypeConvertYAML       ErrorType = "convert_yaml"
	ErrorTypeConvertIO         ErrorType = "convert_io"
	ErrorTypeConvertValidation ErrorType = "convert_validation"

	// General errors
	ErrorTypeConfiguration ErrorType = "configuration"
	ErrorTypeInternal      ErrorType = "internal"
)

// ErrorContext provides additional context for errors
type ErrorContext struct {
	Operation string            `json:"operation"`
	Component string            `json:"component"`
	Timestamp time.Time         `json:"timestamp"`
	UserID    string            `json:"user_id,omitempty"`
	WriteKey  string            `json:"write_key,omitempty"`
	SchemaUID string            `json:"schema_uid,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// SchemaError represents a structured error with context and tracing
type SchemaError struct {
	Type      ErrorType    `json:"type"`
	Message   string       `json:"message"`
	Context   ErrorContext `json:"context"`
	Cause     error        `json:"cause,omitempty"`
	Code      string       `json:"code,omitempty"`
	Retryable bool         `json:"retryable"`
	UserError bool         `json:"user_error"` // Whether this is a user-actionable error
}

// Error implements the error interface
func (e *SchemaError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying cause
func (e *SchemaError) Unwrap() error {
	return e.Cause
}

// IsRetryable returns whether the error is retryable
func (e *SchemaError) IsRetryable() bool {
	return e.Retryable
}

// IsUserError returns whether this is a user-actionable error
func (e *SchemaError) IsUserError() bool {
	return e.UserError
}

// WithContext adds context to the error
func (e *SchemaError) WithContext(key, value string) *SchemaError {
	if e.Context.Metadata == nil {
		e.Context.Metadata = make(map[string]string)
	}
	e.Context.Metadata[key] = value
	return e
}

// NewSchemaError creates a new structured error
func NewSchemaError(errorType ErrorType, message string, options ...ErrorOption) *SchemaError {
	err := &SchemaError{
		Type:    errorType,
		Message: message,
		Context: ErrorContext{
			Timestamp: time.Now(),
			Metadata:  make(map[string]string),
		},
		Retryable: false,
		UserError: false,
	}

	for _, opt := range options {
		opt(err)
	}

	return err
}

// ErrorOption represents an option for configuring errors
type ErrorOption func(*SchemaError)

// WithCause adds a causing error
func WithCause(cause error) ErrorOption {
	return func(e *SchemaError) {
		e.Cause = cause
	}
}

// WithCode adds an error code
func WithCode(code string) ErrorOption {
	return func(e *SchemaError) {
		e.Code = code
	}
}

// WithOperation sets the operation context
func WithOperation(operation string) ErrorOption {
	return func(e *SchemaError) {
		e.Context.Operation = operation
	}
}

// WithComponent sets the component context
func WithComponent(component string) ErrorOption {
	return func(e *SchemaError) {
		e.Context.Component = component
	}
}

// WithWriteKey sets the write key context
func WithWriteKey(writeKey string) ErrorOption {
	return func(e *SchemaError) {
		e.Context.WriteKey = writeKey
	}
}

// WithSchemaUID sets the schema UID context
func WithSchemaUID(uid string) ErrorOption {
	return func(e *SchemaError) {
		e.Context.SchemaUID = uid
	}
}

// WithMetadata adds metadata to the error
func WithMetadata(key, value string) ErrorOption {
	return func(e *SchemaError) {
		if e.Context.Metadata == nil {
			e.Context.Metadata = make(map[string]string)
		}
		e.Context.Metadata[key] = value
	}
}

// AsRetryable marks the error as retryable
func AsRetryable() ErrorOption {
	return func(e *SchemaError) {
		e.Retryable = true
	}
}

// AsUserError marks the error as user-actionable
func AsUserError() ErrorOption {
	return func(e *SchemaError) {
		e.UserError = true
	}
}

// Common error constructors

// NewFetchError creates a fetch-related error
func NewFetchError(subType ErrorType, message string, options ...ErrorOption) *SchemaError {
	opts := append([]ErrorOption{WithComponent("fetch")}, options...)
	return NewSchemaError(subType, message, opts...)
}

// NewProcessError creates a processing-related error
func NewProcessError(subType ErrorType, message string, options ...ErrorOption) *SchemaError {
	opts := append([]ErrorOption{WithComponent("process")}, options...)
	return NewSchemaError(subType, message, opts...)
}

// NewConvertError creates a conversion-related error
func NewConvertError(subType ErrorType, message string, options ...ErrorOption) *SchemaError {
	opts := append([]ErrorOption{WithComponent("convert")}, options...)
	return NewSchemaError(subType, message, opts...)
}

// Error recovery helpers

// ShouldRetry determines if an operation should be retried based on the error
func ShouldRetry(err error) bool {
	if schemaErr, ok := err.(*SchemaError); ok {
		return schemaErr.IsRetryable()
	}
	return false
}

// GetUserMessage returns a user-friendly error message
func GetUserMessage(err error) string {
	if schemaErr, ok := err.(*SchemaError); ok {
		if schemaErr.IsUserError() {
			return schemaErr.Message
		}
		// For non-user errors, provide generic guidance
		switch schemaErr.Type {
		case ErrorTypeFetchConnection:
			return "Unable to connect to the API. Please check your internet connection and try again."
		case ErrorTypeFetchAPI:
			return "API request failed. Please verify your access token and try again."
		case ErrorTypeProcessValidation:
			return "Invalid input data. Please check your schema file and try again."
		case ErrorTypeConvertIO:
			return "File operation failed. Please check file permissions and try again."
		default:
			return "An unexpected error occurred. Please try again or contact support."
		}
	}
	return err.Error()
}
