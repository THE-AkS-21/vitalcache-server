package controllers

import (
	"log/slog"
	"net/http"

	"github.com/THE-AkS-21/vitalcache-server/cmd/api/store"
	"github.com/gin-gonic/gin"
)

type MedicineController struct {
	store *store.MedicineStore
}

func NewMedicineController(store *store.MedicineStore) *MedicineController {
	return &MedicineController{store: store}
}

// New Handler: Get a list of all medicines
func (mc *MedicineController) GetAllMedicines(c *gin.Context) {
	medicines, err := mc.store.GetAll()
	if err != nil {
		slog.Error("Failed to get all medicines", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve medicine list"})
		return
	}

	c.JSON(http.StatusOK, medicines)
}
