package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/THE-AkS-21/vitalcache-server/internal/domain"
	"github.com/THE-AkS-21/vitalcache-server/internal/http/dto"
	"github.com/THE-AkS-21/vitalcache-server/internal/http/handlers"
	"github.com/THE-AkS-21/vitalcache-server/internal/http/handlers/hdeps"
	"github.com/gin-gonic/gin"
)

// ---------- Mocks ----------

type mockPatientsService struct {
	GetByIDFunc        func(ctx context.Context, token string, id int) (*domain.Patient, error)
	CreateFunc         func(ctx context.Context, token string, req dto.CreatePatientRequest) (domain.Patient, error)
	UpdatePartialFunc  func(ctx context.Context, token string, id int, req dto.UpdatePatientRequest) (domain.Patient, error)
	SearchByMobileFunc func(ctx context.Context, token string, mobile string, limit, offset int) ([]domain.Patient, error)
}

func (m *mockPatientsService) GetByID(ctx context.Context, token string, id int) (*domain.Patient, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, token, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockPatientsService) Create(ctx context.Context, token string, req dto.CreatePatientRequest) (domain.Patient, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, token, req)
	}
	return domain.Patient{}, errors.New("not implemented")
}

func (m *mockPatientsService) UpdatePartial(ctx context.Context, token string, id int, req dto.UpdatePatientRequest) (domain.Patient, error) {
	if m.UpdatePartialFunc != nil {
		return m.UpdatePartialFunc(ctx, token, id, req)
	}
	return domain.Patient{}, errors.New("not implemented")
}

func (m *mockPatientsService) SearchByMobile(ctx context.Context, token string, mobile string, limit, offset int) ([]domain.Patient, error) {
	if m.SearchByMobileFunc != nil {
		return m.SearchByMobileFunc(ctx, token, mobile, limit, offset)
	}
	return nil, errors.New("not implemented")
}

type mockPrescriptionsService struct {
	CreateFunc        func(ctx context.Context, token string, req dto.CreatePrescriptionRequest) (domain.Prescription, error)
	ListByPatientFunc func(ctx context.Context, token string, patientID int, start, end *time.Time, limit, offset int) ([]domain.Prescription, error)
}

func (m *mockPrescriptionsService) Create(ctx context.Context, token string, req dto.CreatePrescriptionRequest) (domain.Prescription, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, token, req)
	}
	return domain.Prescription{}, errors.New("not implemented")
}

func (m *mockPrescriptionsService) ListByPatient(ctx context.Context, token string, patientID int, start, end *time.Time, limit, offset int) ([]domain.Prescription, error) {
	if m.ListByPatientFunc != nil {
		return m.ListByPatientFunc(ctx, token, patientID, start, end, limit, offset)
	}
	return nil, errors.New("not implemented")
}

type mockMedicinesStore struct {
	ListFunc func(ctx context.Context, limit, offset int) ([]domain.Medicine, error)
}

func (m *mockMedicinesStore) List(ctx context.Context, limit, offset int) ([]domain.Medicine, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, limit, offset)
	}
	return nil, errors.New("not implemented")
}

// ---------- Tests ----------

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestGetPatientByID_InvalidID(t *testing.T) {
	r := setupRouter()
	deps := hdeps.Deps{
		Patients: &mockPatientsService{},
	}
	r.GET("/patients/:id", handlers.GetPatientByID(deps))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/patients/abc", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGetPatientByID_Success(t *testing.T) {
	r := setupRouter()
	mockSvc := &mockPatientsService{
		GetByIDFunc: func(ctx context.Context, token string, id int) (*domain.Patient, error) {
			return &domain.Patient{ID: 1, Name: "John Doe"}, nil
		},
	}
	deps := hdeps.Deps{
		Patients: mockSvc,
	}
	r.GET("/patients/:id", handlers.GetPatientByID(deps))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/patients/1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	var p domain.Patient
	json.Unmarshal(w.Body.Bytes(), &p)
	if p.Name != "John Doe" {
		t.Errorf("expected name John Doe, got %s", p.Name)
	}
}

func TestUpdatePatient_InvalidID(t *testing.T) {
	r := setupRouter()
	deps := hdeps.Deps{
		Patients: &mockPatientsService{},
	}
	r.PATCH("/patients/:id", handlers.UpdatePatient(deps))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/patients/abc", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestPatientPrescriptionHistory_InvalidID(t *testing.T) {
	r := setupRouter()
	deps := hdeps.Deps{
		Prescriptions: &mockPrescriptionsService{},
	}
	r.GET("/patients/:id/prescriptions", handlers.PatientPrescriptionHistory(deps))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/patients/abc/prescriptions", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestListMedicines_Success(t *testing.T) {
	r := setupRouter()
	mockStore := &mockMedicinesStore{
		ListFunc: func(ctx context.Context, limit, offset int) ([]domain.Medicine, error) {
			return []domain.Medicine{{ID: 1, Name: "Paracetamol"}}, nil
		},
	}
	deps := hdeps.Deps{
		Medicines: mockStore,
	}
	r.GET("/medicines", handlers.ListMedicines(deps))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/medicines", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	var items []domain.Medicine
	json.Unmarshal(w.Body.Bytes(), &items)
	if len(items) != 1 || items[0].Name != "Paracetamol" {
		t.Errorf("expected Paracetamol, got %v", items)
	}
}
