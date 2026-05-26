package reports

import "context"

type ReportFormat struct {
	HospitalID  string `json:"hospital_id"`
	HeaderText  string `json:"header_text"`
	AddressText string `json:"address_text"`
	FooterText  string `json:"footer_text"`
	LogoURL     string `json:"logo_url"`
}

type UpdateFormatReq struct {
	HeaderText  string `json:"header_text"`
	AddressText string `json:"address_text"`
	FooterText  string `json:"footer_text"`
	LogoURL     string `json:"logo_url"`
}

type Repository interface {
	GetFormat(ctx context.Context, hospitalID string) (*ReportFormat, error)
	UpsertFormat(ctx context.Context, format *ReportFormat) error
}

type Service interface {
	GetReportFormat(ctx context.Context, hospitalID string) (*ReportFormat, error)
	UpdateReportFormat(ctx context.Context, hospitalID string, req UpdateFormatReq) (*ReportFormat, error)
}
