package medicines

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	apperr "github.com/THE-AkS-21/vitalcache-server/internal/pkg/errors"
)

// --- Service Implementation ---
type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) SearchMedicines(ctx context.Context, query string, limit, offset int) ([]Medicine, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	list, total, err := s.repo.Search(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	return list, total, nil
}

// --- Handler Implementation ---
type Handler struct {
	svc Service
	rdb *redis.Client
}

func NewHandler(svc Service, rdb *redis.Client) *Handler {
	return &Handler{svc: svc, rdb: rdb}
}

func (h *Handler) Search(c *gin.Context) {
	query := c.Query("q")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// 1. Redis Cache-Aside Pattern
	cacheKey := fmt.Sprintf("medicines:search:q=%s:l=%d:o=%d", query, limit, offset)

	if h.rdb != nil {
		if cached, err := h.rdb.Get(c.Request.Context(), cacheKey).Result(); err == nil {
			c.Header("X-Cache", "HIT")
			c.Data(http.StatusOK, "application/json", []byte(cached))
			return
		}
	}

	// 2. Fetch from MongoDB
	list, total, err := h.svc.SearchMedicines(c.Request.Context(), query, limit, offset)
	if err != nil {
		apperr.Abort(c, err)
		return
	}

	response := gin.H{
		"data":  list,
		"total": total,
	}

	// 3. Save to Redis (Background goroutine so we don't block the HTTP response)
	if h.rdb != nil {
		respBytes, _ := json.Marshal(response)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			h.rdb.Set(ctx, cacheKey, respBytes, 15*time.Minute)
		}()
	}

	c.Header("X-Cache", "MISS")
	apperr.WriteOK(c, http.StatusOK, response)
}
