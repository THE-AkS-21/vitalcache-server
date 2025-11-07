package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/THE-AkS-21/vitalcache-server/internal/http/dto"
	"github.com/THE-AkS-21/vitalcache-server/internal/http/handlers/hdeps"
	"github.com/gin-gonic/gin"
)

func SendPrescription(d hdeps.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		patientIDStr := c.PostForm("patientId")
		file, err := c.FormFile("prescriptionFile")
		if err != nil || patientIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "patientId and prescriptionFile are required"})
			return
		}
		ext := filepath.Ext(file.Filename)
		switch ext {
		case ".pdf", ".png", ".jpg", ".jpeg":
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported file type"})
			return
		}
		tmp := filepath.Join(osTemp(), fmt.Sprintf("%d-%s", time.Now().UnixNano(), file.Filename))
		if err := c.SaveUploadedFile(file, tmp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
			return
		}

		// resolve doctorID from the authenticated user
		userIDVal, _ := c.Get("user_id")
		doc, err := d.Doctors.GetByUserID(uint(userIDVal.(uint)))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "doctor profile not found"})
			return
		}

		if err := d.Prescriptions.Queue(c.Request.Context(), patientIDStr, tmp, file.Filename, doc.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"message": "queued for sending"})
	}
}

func osTemp() string { return "/tmp" }

// GET /api/v1/patients/:id/prescriptions?start=&end=&limit=&offset=
func PatientPrescriptionHistory(d hdeps.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		// path id
		pidU64, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil || pidU64 == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid patient id"})
			return
		}
		patientID := uint(pidU64)

		// who is calling (doctor required by route guard)
		userIDVal, _ := c.Get("user_id")
		doc, err := d.Doctors.GetByUserID(uint(userIDVal.(uint)))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "doctor profile not found"})
			return
		}

		// query params
		var q dto.HistoryQuery
		if err := c.ShouldBindQuery(&q); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		startRFC3339 := normalizeDate(q.Start, true) // 00:00:00Z
		endRFC3339 := normalizeDate(q.End, false)    // 23:59:59Z

		items, err := d.Prescriptions.ListHistory(
			c.Request.Context(),
			patientID, doc.ID,
			startRFC3339, endRFC3339,
			q.Limit, q.Offset,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"patient_id": patientID,
			"count":      len(items),
			"items":      items,
		})
	}
}

// normalizeDate accepts "YYYY-MM-DD" or RFC3339 and returns RFC3339.
// if isStart=true sets time to 00:00:00Z when only a date is provided,
// else sets it to 23:59:59Z.
func normalizeDate(s string, isStart bool) string {
	if s == "" {
		return ""
	}
	// Try YYYY-MM-DD
	if t, err := time.Parse("2006-01-02", s); err == nil {
		if isStart {
			return t.UTC().Format(time.RFC3339)
		}
		// end of day
		return t.Add(23*time.Hour + 59*time.Minute + 59*time.Second).UTC().Format(time.RFC3339)
	}
	// Try RFC3339
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC().Format(time.RFC3339)
	}
	// Invalid: return empty to ignore
	return ""
}
