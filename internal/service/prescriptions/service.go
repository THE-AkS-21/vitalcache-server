package prescriptions

import (
	_ "bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/THE-AkS-21/vitalcache-server/internal/domain"
	"github.com/THE-AkS-21/vitalcache-server/internal/queue"
	"github.com/THE-AkS-21/vitalcache-server/internal/store/supabase"
	supa "github.com/supabase-community/supabase-go"
)

type Service struct {
	patients      *supabase.PatientsStore
	prescriptions *supabase.PrescriptionsStore
	db            *supa.Client
	q             queue.Client
}

func NewService(ps *supabase.PatientsStore, db *supa.Client, q queue.Client) *Service {
	return &Service{
		patients:      ps,
		prescriptions: supabase.NewPrescriptionsStore(db),
		db:            db,
		q:             q,
	}
}

// Queue: uploads file to storage (best-effort), persist a DB row, then enqueue background job.
// doctorID is required for RBAC/audit; handler should resolve it from the token via DoctorsStore.
func (s *Service) Queue(ctx context.Context, patientIDStr, tempPath, fileName string, doctorID uint) error {
	pid, err := parseUint(patientIDStr)
	if err != nil {
		return fmt.Errorf("invalid patientId")
	}

	// Guard on file existence + size
	fi, err := os.Stat(tempPath)
	if err != nil {
		return fmt.Errorf("temp file not found")
	}
	if fi.Size() > 5*1024*1024 {
		return fmt.Errorf("file too large")
	}

	// Upload to Supabase Storage (best effort)
	body, readErr := os.ReadFile(tempPath)
	var storagePath string
	if readErr == nil {
		storagePath = filepath.Join(strconv.FormatUint(uint64(pid), 10), fmt.Sprintf("%d-%s", time.Now().Unix(), fileName))
		_, _ = s.db.Storage.UploadFile("prescriptions", storagePath, bytesReader(body))
		// (You can later switch to signed/public URLs; for now we store the storage path)
	}

	// Persist prescription row (always; even if upload failed we store metadata)
	_, _ = s.prescriptions.Create(ctx, domain.Prescription{
		PatientID: pid,
		DoctorID:  doctorID,
		FileURL:   storagePath, // may be "", still useful to track that a file was sent
		SentAt:    time.Now().UTC(),
	})

	// Enqueue job (in-memory for now; worker just logs & deletes temp file)
	return s.q.EnqueuePrescription(ctx, pid, tempPath, fileName)
}

// ---- helpers ----

func parseUint(s string) (uint, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	return uint(v), err
}

type byteReader []byte

func (b byteReader) Read(p []byte) (int, error) {
	n := copy(p, b)
	if n < len(b) {
		return n, nil
	}
	return n, io.EOF
}
func bytesReader(b []byte) byteReader { return byteReader(b) }

// ListHistory returns a patient's prescriptions, filtered by optional date range.
func (s *Service) ListHistory(
	ctx context.Context,
	patientID, doctorID uint,
	startRFC3339, endRFC3339 string,
	limit, offset int,
) ([]domain.Prescription, error) {
	// App-level check optional (RLS should already restrict); keep light.
	return s.prescriptions.ListByPatientVisibleToDoctorRange(ctx, patientID, doctorID, startRFC3339, endRFC3339, limit, offset)
}
