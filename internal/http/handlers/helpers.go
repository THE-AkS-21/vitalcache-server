package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func clampLimit(c *gin.Context) int {
	n, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if n <= 0 {
		n = 50
	}
	if n > 100 {
		n = 100
	}
	return n
}

func clampOffset(c *gin.Context) int {
	n, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if n < 0 {
		n = 0
	}
	return n
}
