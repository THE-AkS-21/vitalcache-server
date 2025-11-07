package policies

import (
	"context"
	"fmt"
	"log/slog"
)

// DoctorCanAccessPatient checks if a doctor is treating a patient.
//
// It first tries the (optional) link table patient_doctors (patient has multiple treating doctors).
// If that table isn't present, it falls back to patients.doctor_id = doctorID.
//
// The two callbacks keep this package decoupled from concrete stores.
type PatientDoctorLookup func(ctx context.Context, patientID, doctorID uint) (bool, error)
type PatientOwnerLookup func(ctx context.Context, patientID, doctorID uint) (bool, error)

func DoctorCanAccessPatient(ctx context.Context,
	doctorID, patientID uint,
	checkLinked PatientDoctorLookup,
	checkOwner PatientOwnerLookup,
) (bool, error) {

	// Try link-table (multi-doctor care)
	if checkLinked != nil {
		ok, err := checkLinked(ctx, patientID, doctorID)
		if err == nil {
			if ok {
				return true, nil
			}
		} else {
			// If the table doesn't exist yet, log and continue to fallback.
			slog.Debug("patient_doctors lookup failed (falling back to patients.doctor_id)", "err", err)
		}
	}

	// Fallback to single owner doctor_id on patients row
	if checkOwner != nil {
		ok, err := checkOwner(ctx, patientID, doctorID)
		if err != nil {
			return false, fmt.Errorf("patient owner lookup: %w", err)
		}
		return ok, nil
	}

	return false, nil
}
