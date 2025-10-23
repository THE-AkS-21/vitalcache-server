package services

import (
	"bytes" // Import the bytes package
	"fmt"
	"log/slog"
	"os"

	"github.com/THE-AkS-21/vitalcache-server/cmd/api/store"
	supa "github.com/supabase-community/supabase-go"
)

type PrescriptionService struct {
	patientStore *store.PatientStore
	emailService *EmailService
	db           *supa.Client
}

func NewPrescriptionService(ps *store.PatientStore, es *EmailService, db *supa.Client) *PrescriptionService {
	return &PrescriptionService{patientStore: ps, emailService: es, db: db}
}

// ProcessAndSendPrescription now correctly handles file upload and email queuing.
func (s *PrescriptionService) ProcessAndSendPrescription(patientID, doctorID uint, filePath, fileName string) error {
	// Step 1: Get patient details to find their email
	patient, err := s.patientStore.GetByID(patientID, doctorID)
	if err != nil {
		return fmt.Errorf("could not find patient: %v", err)
	}
	if patient.Email == "" {
		return fmt.Errorf("patient does not have an email address on file")
	}

	// Step 2 (Optional): Upload the file to Supabase Storage for record-keeping
	fileBody, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("could not read file for upload: %v", err)
	}

	bucketName := "prescriptions" // Ensure this bucket exists in your Supabase project
	storagePath := fmt.Sprintf("%d/%s", patientID, fileName)

	// Corrected: Convert []byte to io.Reader for the UploadFile function
	_, err = s.db.Storage.UploadFile(bucketName, storagePath, bytes.NewReader(fileBody))
	if err != nil {
		slog.Warn("Failed to upload prescription to Supabase Storage", "error", err, "path", storagePath)
		// We can continue even if storage upload fails, as email is the primary goal.
	} else {
		slog.Info("Successfully uploaded prescription to storage", "path", storagePath)
	}

	// Step 3: Create the email content
	emailSubject := "Your Prescription from VitalCache"
	emailBody := "Dear " + patient.Name + ",<br><br>Please find your prescription attached to this email.<br><br>Sincerely,<br>Your Doctor at VitalCache"

	// Step 4: Create a job and push it to the queue
	job := EmailJob{
		To:             patient.Email,
		Subject:        emailSubject,
		Body:           emailBody,
		AttachmentPath: filePath,
		Retries:        0,
	}

	EmailJobQueue <- job

	return nil
}
