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
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/validator"
)

// --- Service Implementation ---

// service implements the Service interface for the medicines catalog.
// It interfaces directly with the MongoDB Repository to perform CRUD operations
// on the medicines collection.
type service struct {
	repo Repository
}

// NewService creates a new medicines Service.
//
// Args:
//
//	repo (Repository): The database repository implementation.
//
// Returns:
//
//	Service: The implemented service interface.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// SearchMedicines performs a text-based search across the medicines collection.
// It uses MongoDB indexes to search by medicine name, generic name, or manufacturer.
//
// Args:
//
//	ctx (context.Context): The request context.
//	query (string): The search string.
//	limit (int): Maximum number of results to return (max 100).
//	offset (int): Pagination offset.
//
// Returns:
//
//	[]Medicine: A slice of medicines matching the search query.
//	int64: The total number of matches found.
//	error: An internal error if the database query fails.
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

// CreateMedicine adds a new medicine to the catalog database.
//
// Args:
//
//	ctx (context.Context): The request context.
//	req (CreateMedicineReq): The structured payload for the new medicine.
//
// Returns:
//
//	*Medicine: The newly created medicine entity.
//	error: An internal error if the database insert fails.
func (s *service) CreateMedicine(ctx context.Context, req CreateMedicineReq) (*Medicine, error) {
	med := &Medicine{
		Name:         req.Name,
		GenericName:  req.GenericName,
		Manufacturer: req.Manufacturer,
		Tags:         req.Tags,
	}
	if err := s.repo.Create(ctx, med); err != nil {
		return nil, apperr.Internal(err)
	}
	return med, nil
}

// --- Handler Implementation ---

// Handler implements the Gin HTTP endpoints for the medicines service.
// It utilizes an injected Redis client to perform cache-aside caching for search results.
type Handler struct {
	svc Service
	rdb *redis.Client
}

// NewHandler creates a new HTTP handler for medicines.
//
// Args:
//
//	svc (Service): The business logic service layer.
//	rdb (*redis.Client): Redis client for caching responses.
//
// Returns:
//
//	*Handler: The initialized HTTP handler.
func NewHandler(svc Service, rdb *redis.Client) *Handler {
	return &Handler{svc: svc, rdb: rdb}
}

// Search is an HTTP handler for retrieving paginated and cached medicine lists.
// It intercepts the query parameters `q`, `limit`, and `offset`, checks the Redis
// cache, and if missing, falls back to the database, subsequently caching the result.
//
// Args:
//
//	c (*gin.Context): The Gin HTTP request context.
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

// Create is an HTTP handler for creating a new medicine.
// It parses the JSON payload, validates the structural constraints, and creates the record.
//
// Args:
//
//	c (*gin.Context): The Gin HTTP request context.
func (h *Handler) Create(c *gin.Context) {
	var req CreateMedicineReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Abort(c, apperr.BadRequest("invalid JSON payload"))
		return
	}
	if err := validator.Check(&req); err != nil {
		apperr.Abort(c, err)
		return
	}

	med, err := h.svc.CreateMedicine(c.Request.Context(), req)
	if err != nil {
		apperr.Abort(c, err)
		return
	}

	apperr.WriteOK(c, http.StatusCreated, med)
}
