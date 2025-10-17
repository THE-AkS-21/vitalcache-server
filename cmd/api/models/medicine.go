package models

import "time"

type Medicine struct {
	ID                uint      `json:"id,omitempty"`
	Name              string    `json:"name"`
	Dose              string    `json:"dose,omitempty"`
	DurationDays      int       `json:"duration_days,omitempty"`
	Frequency         string    `json:"frequency,omitempty"`
	RecommendedBrands string    `json:"recommended_brands,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}
