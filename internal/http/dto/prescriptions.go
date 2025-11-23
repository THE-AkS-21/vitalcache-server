package dto

// HistoryQuery binds ?start=&end=&limit=&offset=
type HistoryQuery struct {
	Start  string `form:"start"` // optional: 2025-01-01 or RFC3339
	End    string `form:"end"`   // optional
	Limit  int    `form:"limit,default=50"`
	Offset int    `form:"offset,default=0"`
}

// CreatePrescriptionRequest – JSON payload used by handler; server sets DoctorID.
type CreatePrescriptionRequest struct {
	PatientID int     `json:"patient_id" binding:"required"`
	DoctorID  int     `json:"-"` // injected from token by handler; not accepted from client
	BundleID  *int    `json:"bundle_id,omitempty"`
	Notes     *string `json:"notes,omitempty"`
	FileURL   *string `json:"file_url,omitempty"`
}
