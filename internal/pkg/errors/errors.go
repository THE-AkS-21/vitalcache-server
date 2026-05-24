package errors

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	TraceID string `json:"trace_id,omitempty"`
	Err     error  `json:"-"` // Internal error for logging, not sent to client
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func New(code, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// --- Helpers expected by handlers and services ---

func BadRequest(message string) *AppError {
	return &AppError{Code: "BAD_REQUEST", Message: message}
}

func Unauthorized(message string) *AppError {
	return &AppError{Code: "UNAUTHORIZED", Message: message}
}

func Internal(err error) *AppError {
	return &AppError{Code: "INTERNAL_ERROR", Message: "An unexpected server error occurred", Err: err}
}

func Conflict(message string, err error) *AppError {
	return &AppError{Code: "CONFLICT", Message: message, Err: err}
}

func Forbidden(message string) *AppError {
	return &AppError{Code: "FORBIDDEN", Message: message}
}

func NotFound(resource string, err error) *AppError {
	msg := resource + " not found"
	return &AppError{Code: "RESOURCE_NOT_FOUND", Message: msg, Err: err}
}

// --- HTTP Responders ---

// Abort standardizes error JSON responses across the app
func Abort(c *gin.Context, err error) {
	var appErr *AppError
	if e, ok := err.(*AppError); ok {
		appErr = e
	} else {
		appErr = Internal(err)
	}

	status := http.StatusInternalServerError
	switch appErr.Code {
	case "BAD_REQUEST", "VALIDATION_ERROR":
		status = http.StatusBadRequest
	case "UNAUTHORIZED":
		status = http.StatusUnauthorized
	case "FORBIDDEN":
		status = http.StatusForbidden
	case "RESOURCE_NOT_FOUND":
		status = http.StatusNotFound
	case "CONFLICT":
		status = http.StatusConflict
	}

	// Attach trace ID if present from middleware
	if traceID, exists := c.Get("request_id"); exists {
		appErr.TraceID = traceID.(string)
	}

	c.AbortWithStatusJSON(status, gin.H{
		"success": false,
		"error":   appErr,
	})
}

// WriteOK standardizes successful JSON responses
func WriteOK(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{"success": true, "data": data})
}

// PaginationMeta is the standard pagination envelope.
type PaginationMeta struct {
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
	Total  int64 `json:"total"`
}

// WritePaginated sends a standardized paginated list response.
func WritePaginated(c *gin.Context, status int, data any, meta PaginationMeta) {
	c.JSON(status, gin.H{
		"success": true,
		"data":    data,
		"meta":    meta,
	})
}
