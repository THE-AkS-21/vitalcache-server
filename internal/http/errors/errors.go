package errors

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrBody struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}
type EnvelopeBody struct {
	Error ErrBody `json:"error"`
}

func Envelope(code, msg string, details interface{}) EnvelopeBody {
	return EnvelopeBody{Error: ErrBody{Code: code, Message: msg, Details: details}}
}

func Write(c *gin.Context, status int, code, msg string, details interface{}) {
	c.JSON(status, Envelope(code, msg, details))
}

func WriteBadRequest(c *gin.Context, msg string, details interface{}) {
	Write(c, http.StatusBadRequest, "bad_request", msg, details)
}
func WriteUnauthorized(c *gin.Context, msg string, details interface{}) {
	Write(c, http.StatusUnauthorized, "unauthorized", msg, details)
}
func WriteForbidden(c *gin.Context, msg string, details interface{}) {
	Write(c, http.StatusForbidden, "forbidden", msg, details)
}
func WriteNotFound(c *gin.Context, msg string, details interface{}) {
	Write(c, http.StatusNotFound, "not_found", msg, details)
}
func WriteConflict(c *gin.Context, msg string, details interface{}) {
	Write(c, http.StatusConflict, "conflict", msg, details)
}
func WriteInternal(c *gin.Context, err error) {
	Write(c, http.StatusInternalServerError, "internal_error", "Something went wrong", map[string]string{"detail": err.Error()})
}
