// Package pagination provides reusable cursor/offset pagination helpers for Gin handlers.
package pagination

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// Params holds the resolved pagination values for a single request.
type Params struct {
	Limit  int
	Offset int
}

// FromContext extracts ?limit= and ?offset= from the gin.Context query string,
// clamping to safe defaults. Max limit is 100.
func FromContext(c *gin.Context) Params {
	return Params{
		Limit:  parseInt(c.DefaultQuery("limit", "20"), DefaultLimit, 1, MaxLimit),
		Offset: parseInt(c.DefaultQuery("offset", "0"), 0, 0, 1_000_000),
	}
}

func parseInt(s string, def, min, max int) int {
	n, err := strconv.Atoi(s)
	if err != nil || n < min {
		return def
	}
	if n > max {
		return max
	}
	return n
}
