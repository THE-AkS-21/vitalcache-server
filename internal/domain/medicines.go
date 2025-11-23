package domain

import "time"

type Medicine struct {
	ID                uint      `json:"id"`
	Name              string    `json:"name"`
	Dose              string    `json:"dose"`
	DurationDays      int       `json:"duration_days"`
	Frequency         string    `json:"frequency"`
	RecommendedBrands string    `json:"recommended_brands"`
	CreatedAt         time.Time `json:"created_at"`
}
