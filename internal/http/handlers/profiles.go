package handlers

import (
	"github.com/THE-AkS-21/vitalcache-server/internal/app/middlewares"
	"github.com/THE-AkS-21/vitalcache-server/internal/http/handlers/hdeps"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func MyProfile(d hdeps.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middlewares.GetUserID(c)
		role := middlewares.GetRole(c)

		switch role {
		case "doctor":
			data, _, err := d.DB.ForUser(userID, role).
				From("doctors").
				Select("*", "", false).
				Eq("user_id", strconv.Itoa(userID)).
				Single().
				Execute()
			if err != nil || len(data) == 0 {
				c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
				return
			}
			c.Data(http.StatusOK, "application/json", data)

		case "developer":
			data, _, err := d.DB.ForUser(userID, role).
				From("developers").
				Select("*", "", false).
				Eq("user_id", strconv.Itoa(userID)).
				Single().
				Execute()
			if err != nil || len(data) == 0 {
				c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
				return
			}
			c.Data(http.StatusOK, "application/json", data)
		default:
			c.JSON(http.StatusForbidden, gin.H{"error": "no profile for this role"})
		}
	}
}
