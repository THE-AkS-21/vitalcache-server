package appointments

import (
	"context"
	"fmt"
	"time"

	apperr "github.com/THE-AkS-21/vitalcache-server/internal/pkg/errors"
)

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateAppointment(ctx context.Context, doctorID string, req CreateAppointmentReq) (*Appointment, error) {
	a := &Appointment{
		DoctorID:        doctorID,
		PatientID:       req.PatientID,
		HospitalID:      req.HospitalID,
		AppointmentTime: req.AppointmentTime,
		Status:          StatusBooked,
		CreatedAt:       time.Now(),
	}

	if err := s.repo.Create(ctx, a); err != nil {
		return nil, apperr.Internal(fmt.Errorf("failed to create appointment: %w", err))
	}

	return a, nil
}

func (s *service) GetDoctorAppointments(ctx context.Context, doctorID string, limit, offset int) ([]Appointment, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	list, total, err := s.repo.ListByDoctor(ctx, doctorID, limit, offset)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	return list, total, nil
}

func (s *service) UpdateStatus(ctx context.Context, doctorID string, appointmentID string, req UpdateStatusReq) (*Appointment, error) {
	app, err := s.repo.GetByID(ctx, appointmentID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if app == nil {
		return nil, apperr.New("NOT_FOUND", "Appointment not found")
	}

	if app.DoctorID != doctorID {
		return nil, apperr.New("FORBIDDEN", "You do not have permission to modify this appointment")
	}

	if err := s.repo.UpdateStatus(ctx, appointmentID, req.Status); err != nil {
		return nil, apperr.Internal(err)
	}

	app.Status = req.Status
	return app, nil
}
