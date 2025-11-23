package handlers

import (
	"net/http"

	"github.com/THE-AkS-21/vitalcache-server/internal/app/middlewares"
	"github.com/THE-AkS-21/vitalcache-server/internal/http/errors"
	"github.com/THE-AkS-21/vitalcache-server/internal/http/handlers/hdeps"
	"github.com/gin-gonic/gin"
)

func ListMedicines(d hdeps.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		println("DEBUG [MedicinesHandler]: ListMedicines request received")
		// 1. Setup pagination defaults
		limit := 50
		offset := 0
		// (Optional: You can parse c.Query("limit") here if you want dynamic page size)

		// 2. Call Real Store
		token := middlewares.GetRawToken(c)
		println("DEBUG [MedicinesHandler]: Calling Medicines.List with limit:", limit, "offset:", offset)
		medicines, err := d.Medicines.List(c.Request.Context(), token, limit, offset)
		if err != nil {
			println("DEBUG [MedicinesHandler]: Error from store:", err.Error())
			errors.WriteInternal(c, err)
			return
		}
		println("DEBUG [MedicinesHandler]: Retrieved", len(medicines), "medicines")

		c.JSON(http.StatusOK, medicines)
		println("DEBUG [MedicinesHandler]: Response sent successfully")
	}
}
