// Package validator wraps go-playground/validator/v10, returning structured
// *AppError values instead of raw validation errors, making handler code
// uniform and client error messages consistent.
package validator

import (
	"errors"
	"reflect"
	"strings"
	"sync"

	apperr "github.com/THE-AkS-21/vitalcache-server/internal/pkg/errors"
	lib "github.com/go-playground/validator/v10"
)

var (
	instance *lib.Validate
	once     sync.Once
)

// get returns the package-level singleton validator.
func get() *lib.Validate {
	once.Do(func() {
		instance = lib.New()

		// Use the JSON struct tag as the field name in error messages,
		// so clients see field names that match the payload they sent.
		instance.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})
	})
	return instance
}

// Check validates v (must be a struct or pointer-to-struct) and returns a
// *apperr.AppError with Code=VALIDATION_ERROR on failure, nil on success.
//
// Example:
//
//	type CreatePatientRequest struct {
//	    Name  string `json:"name"  validate:"required,min=2,max=100"`
//	    Phone string `json:"phone" validate:"required,e164"`
//	}
//
//	if err := validator.Check(req); err != nil {
//	    apperr.Abort(c, err)
//	    return
//	}
func Check(v any) error {
	if err := get().Struct(v); err != nil {
		var ve lib.ValidationErrors
		if errors.As(err, &ve) {
			msgs := make([]string, 0, len(ve))
			for _, fe := range ve {
				msgs = append(msgs, formatFieldError(fe))
			}
			// ✅ FIX: Use apperr.New() instead of apperr.Validation()
			return apperr.New("VALIDATION_ERROR", strings.Join(msgs, "; "))
		}
		// ✅ FIX: Use apperr.New() instead of apperr.BadRequest()
		return apperr.New("BAD_REQUEST", err.Error())
	}
	return nil
}

// formatFieldError converts a single field validation error into a
// human-readable string.
func formatFieldError(fe lib.FieldError) string {
	f := fe.Field()
	switch fe.Tag() {
	case "required":
		return f + " is required"
	case "email":
		return f + " must be a valid email address"
	case "min":
		return f + " must be at least " + fe.Param() + " characters"
	case "max":
		return f + " must be at most " + fe.Param() + " characters"
	case "oneof":
		return f + " must be one of: " + fe.Param()
	case "e164":
		return f + " must be a valid E.164 phone number (e.g. +919876543210)"
	case "uuid":
		return f + " must be a valid UUID"
	case "url":
		return f + " must be a valid URL"
	case "gte":
		return f + " must be greater than or equal to " + fe.Param()
	case "lte":
		return f + " must be less than or equal to " + fe.Param()
	default:
		return f + " failed validation (" + fe.Tag() + ")"
	}
}
