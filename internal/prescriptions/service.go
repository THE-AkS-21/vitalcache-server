package prescriptions

import (
	"context"
	"encoding/json"
	"time"

	apperr "github.com/THE-AkS-21/vitalcache-server/internal/pkg/errors"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type service struct {
	repo Repository
	rdb  *redis.Client
}

func NewService(repo Repository, rdb *redis.Client) Service {
	return &service{repo: repo, rdb: rdb}
}

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

	return p, nil
}

func (s *service) GetPrescription(ctx context.Context, id string) (*Prescription, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperr.BadRequest("Invalid prescription ID format")
	}

	p, err := s.repo.GetByID(ctx, objID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if p == nil {
		return nil, apperr.New("NOT_FOUND", "Prescription not found")
	}

	return p, nil
}

func (s *service) GetPatientHistory(ctx context.Context, patientID string, limit, offset int) ([]Prescription, int64, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	list, total, err := s.repo.ListByPatient(ctx, patientID, limit, offset)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}

	return list, total, nil
}
