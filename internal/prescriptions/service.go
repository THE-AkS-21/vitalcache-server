package prescriptions

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/cache"
	apperr "github.com/THE-AkS-21/vitalcache-server/internal/pkg/errors"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// service implements the Service interface for managing patient prescriptions.
// It handles creation of new prescriptions with asynchronous PDF generation
// via an Outbox pattern, and retrieves prescription data using a Redis-backed
// cache for performance optimization.
type service struct {
	repo Repository
	rdb  *redis.Client
	c    *cache.Cache
}

// NewService initializes a new Prescription service.
//
// Args:
//
//	repo (Repository): The MongoDB repository for data persistence.
//	rdb (*redis.Client): The Redis client for caching. Can be nil.
//
// Returns:
//
//	Service: The implemented interface for prescription business logic.
func NewService(repo Repository, rdb *redis.Client) Service {
	var c *cache.Cache
	if rdb != nil {
		c = cache.New(rdb)
	}
	return &service{repo: repo, rdb: rdb, c: c}
}

// CreatePrescription creates a new prescription for a patient.
// It generates a new UUID, saves the prescription to MongoDB, and pushes a job
// payload to the outbox table for asynchronous PDF generation.
//
// Args:
//
//	ctx (context.Context): The request context.
//	doctorID (string): The UUID of the doctor issuing the prescription.
//	req (CreatePrescriptionReq): The request payload containing patient and medication data.
//
// Returns:
//
//	*Prescription: The newly created prescription object.
//	error: An error if the creation process fails.
func (s *service) CreatePrescription(ctx context.Context, doctorID string, req CreatePrescriptionReq) (*Prescription, error) {
	p := &Prescription{
		PrescriptionID: uuid.New().String(),
		DoctorID:       doctorID,
		PatientID:      req.PatientID,
		HospitalID:     req.HospitalID,
		Medications:    req.Medications,
		Notes:          req.Notes,
		Status:         "ACTIVE",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	jobData := map[string]string{"prescription_id": p.PrescriptionID, "patient_id": p.PatientID}
	payloadBytes, _ := json.Marshal(jobData)
	job := map[string]string{"type": "generate_pdf", "payload": string(payloadBytes)}
	jobBytes, _ := json.Marshal(job)

	if err := s.repo.CreateWithOutbox(ctx, p, jobBytes); err != nil {
		return nil, apperr.Internal(err)
	}

	if s.c != nil {
		// Invalidate patient history cache pattern by scanning/deleting, or just deleting specific pages if known.
		// For simplicity we use a wildcard scan pattern via a goroutine or just delete the first few pages.
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			// Basic invalidation: ideally we'd store a set of cache keys for this patient or use pattern matching.
			// This clears the first page cache to ensure immediate updates are seen.
			s.c.Invalidate(bgCtx, fmt.Sprintf("prescriptions:patient:%s:l=20:o=0", req.PatientID))
		}()
	}

	return p, nil
}

// GetPrescription retrieves a specific prescription by its MongoDB ObjectID.
// It utilizes the Redis cache-aside pattern to quickly return frequently accessed prescriptions.
//
// Args:
//
//	ctx (context.Context): The request context.
//	id (string): The hexadecimal string representation of the MongoDB ObjectID.
//
// Returns:
//
//	*Prescription: The requested prescription data.
//	error: An apperr.New "NOT_FOUND" error if it does not exist, or a 500 error on failure.
func (s *service) GetPrescription(ctx context.Context, id string) (*Prescription, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperr.BadRequest("Invalid prescription ID format")
	}

	// We can optionally cache the single prescription here using cache.GetOrLoad
	var p *Prescription
	if s.c != nil {
		type result struct {
			P *Prescription
		}
		cacheKey := cache.KeyPrescription(id)
		res, loadErr := cache.GetOrLoad(ctx, s.c, cacheKey, cache.TTLPrescription, func() (result, error) {
			fetched, dbErr := s.repo.GetByID(ctx, objID)
			return result{P: fetched}, dbErr
		})
		if loadErr != nil {
			return nil, apperr.Internal(loadErr)
		}
		p = res.P
	} else {
		p, err = s.repo.GetByID(ctx, objID)
		if err != nil {
			return nil, apperr.Internal(err)
		}
	}

	if p == nil {
		return nil, apperr.New("NOT_FOUND", "Prescription not found")
	}

	return p, nil
}

// GetPatientHistory fetches a paginated list of all prescriptions for a specific patient.
// This method heavily utilizes Redis caching to speed up history retrieval.
//
// Args:
//
//	ctx (context.Context): The request context.
//	patientID (string): The UUID of the patient.
//	limit (int): The maximum number of records to return (defaults to 20, max 50).
//	offset (int): The pagination offset.
//
// Returns:
//
//	[]Prescription: A slice of prescription records.
//	int64: The total count of prescriptions for the patient (useful for pagination logic).
//	error: An error if the database or cache query fails.
func (s *service) GetPatientHistory(ctx context.Context, patientID string, limit, offset int) ([]Prescription, int64, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	type historyResult struct {
		List  []Prescription `json:"list"`
		Total int64          `json:"total"`
	}

	loadFn := func() (historyResult, error) {
		list, total, err := s.repo.ListByPatient(ctx, patientID, limit, offset)
		return historyResult{List: list, Total: total}, err
	}

	if s.c != nil {
		cacheKey := fmt.Sprintf("prescriptions:patient:%s:l=%d:o=%d", patientID, limit, offset)
		res, err := cache.GetOrLoad(ctx, s.c, cacheKey, cache.TTLPrescription, loadFn)
		if err != nil {
			return nil, 0, apperr.Internal(err)
		}
		return res.List, res.Total, nil
	}

	res, err := loadFn()
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	return res.List, res.Total, nil
}
