// Package history provides append-only clinical event tracking for VitalCache.
// Events are stored in MongoDB (patient_history collection) and are immutable —
// they are never deleted or updated, only appended.
package history

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"

	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/db"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/logger"
)

// EventType classifies a history entry.
type EventType string

const (
	EventVisit           EventType = "visit"
	EventPrescription    EventType = "prescription"
	EventLabResult       EventType = "lab_result"
	EventHospitalisation EventType = "hospitalisation"
	EventNote            EventType = "note"
)

// Event is an immutable timeline entry for a patient's clinical history.
type Event struct {
	ID          bson.ObjectID          `bson:"_id,omitempty" json:"id"`
	PatientID   int64                  `bson:"patient_id"    json:"patient_id"`
	DoctorID    int64                  `bson:"doctor_id"     json:"doctor_id"`
	Type        EventType              `bson:"type"          json:"type"`
	Title       string                 `bson:"title"         json:"title"`
	Description string                 `bson:"description"   json:"description"`
	Metadata    map[string]interface{} `bson:"metadata"      json:"metadata,omitempty"`
	CreatedAt   time.Time              `bson:"created_at"    json:"created_at"`
}

// AppendRequest is the payload for adding a history event.
type AppendRequest struct {
	Type        EventType              `json:"type"        validate:"required,oneof=visit prescription lab_result hospitalisation note"`
	Title       string                 `json:"title"       validate:"required,min=1,max=200"`
	Description string                 `json:"description" validate:"required"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Repository
// ─────────────────────────────────────────────────────────────────────────────

// MongoHistoryRepo is the MongoDB implementation of the history store.
type MongoHistoryRepo struct {
	coll *mongo.Collection
	log  *zap.Logger
}

// NewMongoHistoryRepo returns a production-ready history repository.
func NewMongoHistoryRepo(mongoClient *db.MongoClient) *MongoHistoryRepo {
	return &MongoHistoryRepo{
		coll: mongoClient.Collection(db.CollPatientHistory),
		log:  logger.Named("history.repo"),
	}
}

// Append inserts a new immutable event for a patient.
func (r *MongoHistoryRepo) Append(ctx context.Context, e Event) (*Event, error) {
	e.ID = bson.NewObjectID()
	e.CreatedAt = time.Now().UTC()

	if _, err := r.coll.InsertOne(ctx, e); err != nil {
		return nil, fmt.Errorf("history.Append: %w", err)
	}
	return &e, nil
}

// GetByPatient returns a paginated timeline for a patient (newest first).
func (r *MongoHistoryRepo) GetByPatient(ctx context.Context, patientID int64, limit, offset int) ([]Event, int64, error) {
	filter := bson.D{{Key: "patient_id", Value: patientID}}

	total, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("history.GetByPatient count: %w", err)
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("history.GetByPatient find: %w", err)
	}
	defer cursor.Close(ctx)

	var result []Event
	if err := cursor.All(ctx, &result); err != nil {
		return nil, 0, fmt.Errorf("history.GetByPatient decode: %w", err)
	}
	return result, total, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Service
// ─────────────────────────────────────────────────────────────────────────────

// Service is the history use-case layer.
type Service struct {
	repo *MongoHistoryRepo
	log  *zap.Logger
}

// NewService creates a history service.
func NewService(repo *MongoHistoryRepo) *Service {
	return &Service{repo: repo, log: logger.Named("history.service")}
}

// Append validates and records a new clinical event.
func (s *Service) Append(ctx context.Context, req AppendRequest, patientID, doctorID int64) (*Event, error) {
	e := Event{
		PatientID:   patientID,
		DoctorID:    doctorID,
		Type:        req.Type,
		Title:       req.Title,
		Description: req.Description,
		Metadata:    req.Metadata,
	}
	created, err := s.repo.Append(ctx, e)
	if err != nil {
		return nil, fmt.Errorf("history.Service.Append: %w", err)
	}
	s.log.Info("history event appended",
		zap.String("type", string(e.Type)),
		zap.Int64("patient_id", patientID),
	)
	return created, nil
}

// GetByPatient returns a patient's clinical timeline.
func (s *Service) GetByPatient(ctx context.Context, patientID int64, limit, offset int) ([]Event, int64, error) {
	return s.repo.GetByPatient(ctx, patientID, limit, offset)
}
