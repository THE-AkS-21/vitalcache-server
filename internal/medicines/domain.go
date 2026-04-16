package medicines

import (
	"context"
)

// --- Entities ---
type Medicine struct {
	ID           string   `json:"id" bson:"_id,omitempty"`
	Name         string   `json:"name" bson:"name"`
	GenericName  string   `json:"generic_name" bson:"generic_name"`
	Manufacturer string   `json:"manufacturer" bson:"manufacturer"`
	Tags         []string `json:"tags" bson:"tags"`
}

// --- Interfaces ---
type Repository interface {
	Search(ctx context.Context, query string, limit, offset int) ([]Medicine, int64, error)
}

type Service interface {
	SearchMedicines(ctx context.Context, query string, limit, offset int) ([]Medicine, int64, error)
}
