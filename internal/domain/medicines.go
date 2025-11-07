package domain

import "time"

type Medicine struct {
	ID                uint
	Name              string
	Dose              string
	DurationDays      int
	Frequency         string
	RecommendedBrands string
	CreatedAt         time.Time
}
