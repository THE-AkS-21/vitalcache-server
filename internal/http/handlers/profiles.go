package handlers

import (
	"net/http"

	"github.com/THE-AkS-21/vitalcache-server/internal/http/handlers/hdeps"
	"github.com/gin-gonic/gin"
)

func MyProfile(d hdeps.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		role, _ := c.Get("user_role")

		switch role {
		case "doctor":
			prof, err := d.Doctors.GetByUserID(uint(userID.(uint)))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
				return
			}
			c.JSON(http.StatusOK, prof)
		case "developer":
			prof, err := d.Developers.GetByUserID(uint(userID.(uint)))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
				return
			}
			c.JSON(http.StatusOK, prof)
		default:
			c.JSON(http.StatusForbidden, gin.H{"error": "no profile for this role"})
		}
	}
}
