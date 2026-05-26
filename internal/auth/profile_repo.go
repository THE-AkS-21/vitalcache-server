package auth

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type profileRepo struct {
	db *pgxpool.Pool
}

func NewProfileRepository(db *pgxpool.Pool) ProfileRepository {
	return &profileRepo{db: db}
}

func (r *profileRepo) GetProfileAndPermissions(ctx context.Context, userID string) (*Profile, error) {
	profile := &Profile{UserID: userID, Permissions: []string{}}

	// 1. Get Role and Designation names
	queryRole := `
		SELECT r.name, d.name 
		FROM user_roles ur
		JOIN roles r ON ur.role_id = r.id
		JOIN designations d ON ur.designation_id = d.id
		WHERE ur.user_id = $1 LIMIT 1
	`
	_ = r.db.QueryRow(ctx, queryRole, userID).Scan(&profile.Role, &profile.Designation)

	// 2. Aggregate Permissions (UNION of role_permissions and designation_permissions)
	queryPerms := `
		SELECT p.name FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN user_roles ur ON rp.role_id = ur.role_id
		WHERE ur.user_id = $1
		UNION
		SELECT p.name FROM permissions p
		JOIN designation_permissions dp ON p.id = dp.permission_id
		JOIN user_roles ur ON dp.designation_id = ur.designation_id
		WHERE ur.user_id = $1
	`
	rows, err := r.db.Query(ctx, queryPerms, userID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var perm string
			if rows.Scan(&perm) == nil {
				profile.Permissions = append(profile.Permissions, perm)
			}
		}
	}

	// 3. Fetch specific entity IDs if applicable
	if profile.Role == "DOCTOR" {
		_ = r.db.QueryRow(ctx, "SELECT id FROM doctors WHERE user_id = $1", userID).Scan(&profile.DoctorID)
	} else if profile.Role == "PATIENT" {
		_ = r.db.QueryRow(ctx, "SELECT id FROM patients WHERE user_id = $1", userID).Scan(&profile.PatientID)
	}

	return profile, nil
}
