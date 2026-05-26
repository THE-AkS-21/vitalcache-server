package reports

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxRepo struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgxRepo{pool: pool}
}

func (r *pgxRepo) GetFormat(ctx context.Context, hospitalID string) (*ReportFormat, error) {
	query := `
		SELECT hospital_id, header_text, address_text, footer_text, logo_url
		FROM report_formats
		WHERE hospital_id = $1
	`
	var f ReportFormat
	var header, address, footer, logo *string
	err := r.pool.QueryRow(ctx, query, hospitalID).Scan(
		&f.HospitalID, &header, &address, &footer, &logo,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if header != nil {
		f.HeaderText = *header
	}
	if address != nil {
		f.AddressText = *address
	}
	if footer != nil {
		f.FooterText = *footer
	}
	if logo != nil {
		f.LogoURL = *logo
	}
	return &f, nil
}

func (r *pgxRepo) UpsertFormat(ctx context.Context, format *ReportFormat) error {
	query := `
		INSERT INTO report_formats (hospital_id, header_text, address_text, footer_text, logo_url)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (hospital_id) DO UPDATE SET
			header_text = EXCLUDED.header_text,
			address_text = EXCLUDED.address_text,
			footer_text = EXCLUDED.footer_text,
			logo_url = EXCLUDED.logo_url,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.pool.Exec(ctx, query, format.HospitalID, format.HeaderText, format.AddressText, format.FooterText, format.LogoURL)
	return err
}
