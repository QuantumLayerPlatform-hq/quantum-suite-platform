package errors

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

// ErrorType represents the category of error
type ErrorType string

const (
	// Request errors
	ErrorTypeValidation     ErrorType = "validation_error"
	ErrorTypeAuthentication ErrorType = "authentication_error"
	ErrorTypeAuthorization  ErrorType = "authorization_error"
	ErrorTypeNotFound       ErrorType = "not_found"
	ErrorTypeConflict      ErrorType = "conflict"
	ErrorTypeTooManyRequests ErrorType = "too_many_requests"
	
	// Business logic errors
	ErrorTypeBusiness      ErrorType = "business_error"
	ErrorTypeQuotaExceeded ErrorType = "quota_exceeded"
	ErrorTypeBudgetExceeded ErrorType = "budget_exceeded"
	ErrorTypeProviderLimit ErrorType = "provider_limit"
	
	// System errors
	ErrorTypeInternal       ErrorType = "internal_error"
	ErrorTypeConfiguration  ErrorType = "configuration_error"
	ErrorTypeTimeout        ErrorType = "timeout"
	ErrorTypeUnavailable    ErrorType = "service_unavailable"
	ErrorTypeExternal       ErrorType = "external_service_error"
	
	// Provider errors
	ErrorTypeProviderError       ErrorType = "provider_error"
	ErrorTypeProviderUnavailable ErrorType = "provider_unavailable"
	ErrorTypeModelUnavailable    ErrorType = "model_unavailable"
	ErrorTypeInvalidModel        ErrorType = "invalid_model"
)

// ErrorSeverity indicates how critical the error is
type ErrorSeverity string

const (
	ErrorSeverityLow      ErrorSeverity = "low"
	ErrorSeverityMedium   ErrorSeverity = "medium"
	ErrorSeverityHigh     ErrorSeverity = "high"
	ErrorSeverityCritical ErrorSeverity = "critical"
	
	// Legacy aliases for backwards compatibility
	SeverityLow      = ErrorSeverityLow
	SeverityMedium   = ErrorSeverityMedium
	SeverityHigh     = ErrorSeverityHigh
	SeverityCritical = ErrorSeverityCritical
)

// QLensError represents a structured error in the QLens system
type QLensError struct {
	// Public fields - safe to expose to clients
	Code       string                 `json:"code"`
	Type       ErrorType              `json:"type"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	RequestID  string                 `json:"request_id,omitempty"`
	
	// Metadata for internal use
	Severity   ErrorSeverity          `json:"-"`
	Retryable  bool                   `json:"-"`
	StatusCode int                    `json:"-"`
	Internal   error                  `json:"-"` // Never exposed to clients
	Context    map[string]interface{} `json:"-"` // Internal context
	
	// Tracing information
	Service   string `json:"-"`
	Operation string `json:"-"`
	TenantID  string `json:"-"`
	UserID    string `json:"-"`
}

// Error implements the error interface
func (e *QLensError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %s (internal: %s)", e.Type, e.Message, e.Internal.Error())
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap provides access to the underlying error
func (e *QLensError) Unwrap() error {
	return e.Internal
}

// Is checks if the error matches the target
func (e *QLensError) Is(target error) bool {
	if t, ok := target.(*QLensError); ok {
		return e.Type == t.Type && e.Code == t.Code
	}
	return errors.Is(e.Internal, target)
}

// PublicError returns a sanitized version safe for client exposure
func (e *QLensError) PublicError() *QLensError {
	public := &QLensError{
		Code:      e.Code,
		Type:      e.Type,
		Message:   e.Message,
		Details:   make(map[string]interface{}),
		Timestamp: e.Timestamp,
		RequestID: e.RequestID,
	}
	
	// Only include safe details
	if e.Details != nil {
		for key, value := range e.Details {
			switch key {
			case "field", "parameter", "model", "provider", "tenant_id", "validation_errors":
				public.Details[key] = value
			}
		}
	}
	
	return public
}

// HTTPStatusCode returns the appropriate HTTP status code
func (e *QLensError) HTTPStatusCode() int {
	if e.StatusCode != 0 {
		return e.StatusCode
	}
	
	switch e.Type {
	case ErrorTypeValidation:
		return http.StatusBadRequest
	case ErrorTypeAuthentication:
		return http.StatusUnauthorized
	case ErrorTypeAuthorization:
		return http.StatusForbidden
	case ErrorTypeNotFound:
		return http.StatusNotFound
	case ErrorTypeConflict:
		return http.StatusConflict
	case ErrorTypeTooManyRequests, ErrorTypeQuotaExceeded:
		return http.StatusTooManyRequests
	case ErrorTypeTimeout:
		return http.StatusRequestTimeout
	case ErrorTypeUnavailable, ErrorTypeProviderUnavailable:
		return http.StatusServiceUnavailable
	case ErrorTypeExternal, ErrorTypeProviderError:
		return http.StatusBadGateway
	case ErrorTypeBudgetExceeded, ErrorTypeProviderLimit:
		return http.StatusPaymentRequired
	default:
		return http.StatusInternalServerError
	}
}

// Builder for creating errors with fluent interface
type ErrorBuilder struct {
	err *QLensError
}

// NewError creates a new error builder
func NewError(errorType ErrorType, message string) *ErrorBuilder {
	return &ErrorBuilder{
		err: &QLensError{
			Type:      errorType,
			Message:   message,
			Timestamp: time.Now().UTC(),
			Details:   make(map[string]interface{}),
			Context:   make(map[string]interface{}),
			Severity:  SeverityMedium, // Default severity
		},
	}
}

// WithCode sets the error code
func (b *ErrorBuilder) WithCode(code string) *ErrorBuilder {
	b.err.Code = code
	return b
}

// WithDetail adds a detail field
func (b *ErrorBuilder) WithDetail(key string, value interface{}) *ErrorBuilder {
	if b.err.Details == nil {
		b.err.Details = make(map[string]interface{})
	}
	b.err.Details[key] = value
	return b
}

// WithDetails adds multiple detail fields
func (b *ErrorBuilder) WithDetails(details map[string]interface{}) *ErrorBuilder {
	if b.err.Details == nil {
		b.err.Details = make(map[string]interface{})
	}
	for key, value := range details {
		b.err.Details[key] = value
	}
	return b
}

// WithInternal sets the internal error (not exposed to clients)
func (b *ErrorBuilder) WithInternal(err error) *ErrorBuilder {
	b.err.Internal = err
	return b
}

// WithSeverity sets the error severity
func (b *ErrorBuilder) WithSeverity(severity ErrorSeverity) *ErrorBuilder {
	b.err.Severity = severity
	return b
}

// WithRetryable marks the error as retryable
func (b *ErrorBuilder) WithRetryable(retryable bool) *ErrorBuilder {
	b.err.Retryable = retryable
	return b
}

// WithStatusCode sets a custom HTTP status code
func (b *ErrorBuilder) WithStatusCode(code int) *ErrorBuilder {
	b.err.StatusCode = code
	return b
}

// WithRequestID sets the request ID for tracing
func (b *ErrorBuilder) WithRequestID(requestID string) *ErrorBuilder {
	b.err.RequestID = requestID
	return b
}

// WithService sets the service context
func (b *ErrorBuilder) WithService(service string) *ErrorBuilder {
	b.err.Service = service
	return b
}

// WithOperation sets the operation context
func (b *ErrorBuilder) WithOperation(operation string) *ErrorBuilder {
	b.err.Operation = operation
	return b
}

// WithTenant sets the tenant context
func (b *ErrorBuilder) WithTenant(tenantID string) *ErrorBuilder {
	b.err.TenantID = tenantID
	return b
}

// WithUser sets the user context
func (b *ErrorBuilder) WithUser(userID string) *ErrorBuilder {
	b.err.UserID = userID
	return b
}

// WithContext adds internal context (not exposed)
func (b *ErrorBuilder) WithContext(key string, value interface{}) *ErrorBuilder {
	if b.err.Context == nil {
		b.err.Context = make(map[string]interface{})
	}
	b.err.Context[key] = value
	return b
}

// Build returns the constructed error
func (b *ErrorBuilder) Build() *QLensError {
	// Generate code if not set
	if b.err.Code == "" {
		b.err.Code = generateErrorCode(b.err.Type)
	}
	return b.err
}

// Predefined error constructors

// ValidationError creates a validation error
func ValidationError(message string, field string) *QLensError {
	return NewError(ErrorTypeValidation, message).
		WithDetail("field", field).
		WithSeverity(SeverityLow).
		WithRetryable(false).
		Build()
}

// AuthenticationError creates an authentication error
func AuthenticationError(message string) *QLensError {
	return NewError(ErrorTypeAuthentication, message).
		WithSeverity(SeverityMedium).
		WithRetryable(false).
		Build()
}

// AuthorizationError creates an authorization error
func AuthorizationError(message string) *QLensError {
	return NewError(ErrorTypeAuthorization, message).
		WithSeverity(SeverityMedium).
		WithRetryable(false).
		Build()
}

// NotFoundError creates a not found error
func NotFoundError(resource string, id string) *QLensError {
	return NewError(ErrorTypeNotFound, fmt.Sprintf("%s not found", resource)).
		WithDetail("resource", resource).
		WithDetail("id", id).
		WithSeverity(SeverityLow).
		WithRetryable(false).
		Build()
}

// RateLimitError creates a rate limit error
func RateLimitError(limit int, resetTime time.Time) *QLensError {
	return NewError(ErrorTypeTooManyRequests, "Rate limit exceeded").
		WithDetail("limit", limit).
		WithDetail("reset_time", resetTime).
		WithSeverity(SeverityMedium).
		WithRetryable(true).
		Build()
}

// QuotaExceededError creates a quota exceeded error
func QuotaExceededError(quota int, used int, resetTime time.Time) *QLensError {
	return NewError(ErrorTypeQuotaExceeded, "Quota exceeded").
		WithDetail("quota", quota).
		WithDetail("used", used).
		WithDetail("reset_time", resetTime).
		WithSeverity(SeverityHigh).
		WithRetryable(true).
		Build()
}

// BudgetExceededError creates a budget exceeded error
func BudgetExceededError(budget float64, spent float64) *QLensError {
	return NewError(ErrorTypeBudgetExceeded, "Budget exceeded").
		WithDetail("budget", budget).
		WithDetail("spent", spent).
		WithSeverity(SeverityHigh).
		WithRetryable(false).
		Build()
}

// ProviderError creates a provider error
func ProviderError(provider string, message string, err error) *QLensError {
	return NewError(ErrorTypeProviderError, message).
		WithDetail("provider", provider).
		WithInternal(err).
		WithSeverity(SeverityHigh).
		WithRetryable(true).
		Build()
}

// ProviderUnavailableError creates a provider unavailable error
func ProviderUnavailableError(provider string) *QLensError {
	return NewError(ErrorTypeProviderUnavailable, fmt.Sprintf("Provider %s is unavailable", provider)).
		WithDetail("provider", provider).
		WithSeverity(SeverityHigh).
		WithRetryable(true).
		Build()
}

// ModelUnavailableError creates a model unavailable error
func ModelUnavailableError(model string, provider string) *QLensError {
	return NewError(ErrorTypeModelUnavailable, fmt.Sprintf("Model %s is unavailable", model)).
		WithDetail("model", model).
		WithDetail("provider", provider).
		WithSeverity(SeverityMedium).
		WithRetryable(true).
		Build()
}

// TimeoutError creates a timeout error
func TimeoutError(operation string, timeout time.Duration) *QLensError {
	return NewError(ErrorTypeTimeout, fmt.Sprintf("Operation %s timed out", operation)).
		WithDetail("operation", operation).
		WithDetail("timeout_ms", timeout.Milliseconds()).
		WithSeverity(SeverityHigh).
		WithRetryable(true).
		Build()
}

// InternalError creates an internal server error
func InternalError(message string, err error) *QLensError {
	return NewError(ErrorTypeInternal, message).
		WithInternal(err).
		WithSeverity(SeverityCritical).
		WithRetryable(false).
		Build()
}

// Error utilities

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	var qlensErr *QLensError
	if errors.As(err, &qlensErr) {
		return qlensErr.Retryable
	}
	return false
}

// IsType checks if an error is of a specific type
func IsType(err error, errorType ErrorType) bool {
	var qlensErr *QLensError
	if errors.As(err, &qlensErr) {
		return qlensErr.Type == errorType
	}
	return false
}

// GetSeverity returns the error severity
func GetSeverity(err error) ErrorSeverity {
	var qlensErr *QLensError
	if errors.As(err, &qlensErr) {
		return qlensErr.Severity
	}
	return SeverityMedium
}

// WrapError wraps an existing error with QLens error structure
func WrapError(err error, errorType ErrorType, message string) *QLensError {
	return NewError(errorType, message).
		WithInternal(err).
		Build()
}

// FromError converts any error to QLensError
func FromError(err error) *QLensError {
	var qlensErr *QLensError
	if errors.As(err, &qlensErr) {
		return qlensErr
	}
	
	return NewError(ErrorTypeInternal, "Internal server error").
		WithInternal(err).
		WithSeverity(SeverityCritical).
		Build()
}

// Multi-error handling

// MultiError represents multiple errors
type MultiError struct {
	Errors []error `json:"errors"`
}

// Error implements error interface
func (m *MultiError) Error() string {
	if len(m.Errors) == 0 {
		return "no errors"
	}
	if len(m.Errors) == 1 {
		return m.Errors[0].Error()
	}
	return fmt.Sprintf("%s and %d more errors", m.Errors[0].Error(), len(m.Errors)-1)
}

// Add adds an error to the multi-error
func (m *MultiError) Add(err error) {
	if err != nil {
		m.Errors = append(m.Errors, err)
	}
}

// HasErrors returns true if there are any errors
func (m *MultiError) HasErrors() bool {
	return len(m.Errors) > 0
}

// ToError returns nil if no errors, otherwise returns itself
func (m *MultiError) ToError() error {
	if !m.HasErrors() {
		return nil
	}
	return m
}

// NewMultiError creates a new multi-error
func NewMultiError() *MultiError {
	return &MultiError{
		Errors: make([]error, 0),
	}
}

// Helper functions

func generateErrorCode(errorType ErrorType) string {
	// Generate a unique error code based on type and timestamp
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s_%d", string(errorType), timestamp%10000)
}

// Error context helpers

// WithErrorContext adds context to an error chain
func WithErrorContext(err error, context map[string]interface{}) error {
	if err == nil {
		return nil
	}
	
	var qlensErr *QLensError
	if errors.As(err, &qlensErr) {
		// Add to existing context
		for key, value := range context {
			qlensErr.Context[key] = value
		}
		return qlensErr
	}
	
	// Wrap non-QLens error
	builder := NewError(ErrorTypeInternal, err.Error()).WithInternal(err)
	for key, value := range context {
		builder = builder.WithContext(key, value)
	}
	return builder.Build()
}

// Logging integration helper
func LogError(logger interface{}, err error) {
	// This would integrate with the logger package
	// Implementation depends on logger interface
	if err == nil {
		return
	}
	
	var qlensErr *QLensError
	if errors.As(err, &qlensErr) {
		// Log with all context
		fmt.Printf("ERROR: %+v\n", qlensErr)
	} else {
		fmt.Printf("ERROR: %v\n", err)
	}
}
// ConfigurationError creates a configuration error
func ConfigurationError(message string) *QLensError {
	return NewError(ErrorTypeConfiguration, message).
		WithCode("CONFIGURATION_ERROR").
		WithSeverity(ErrorSeverityHigh).
		WithRetryable(false).
		Build()
}
