package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	dbURL := "postgresql://postgres.mbsqvihwuroltzxgysxq:rHirELm5uGLUonAD@aws-1-ap-south-1.pooler.supabase.com:5432/postgres?sslmode=require"
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// 1. Roles and Designations
	query := `
	-- Insert Developer role
	INSERT INTO public.roles (name) VALUES ('Developer') ON CONFLICT (name) DO NOTHING;
	-- Insert Tester role
	INSERT INTO public.roles (name) VALUES ('Tester') ON CONFLICT (name) DO NOTHING;
	`
	if _, err := pool.Exec(ctx, query); err != nil {
		log.Fatalf("Failed to insert roles: %v", err)
	}

	var devRoleID, testerRoleID string
	err = pool.QueryRow(ctx, "SELECT id FROM public.roles WHERE name = 'Developer'").Scan(&devRoleID)
	if err != nil {
		log.Fatalf("Failed to get Developer role ID: %v", err)
	}
	err = pool.QueryRow(ctx, "SELECT id FROM public.roles WHERE name = 'Tester'").Scan(&testerRoleID)
	if err != nil {
		log.Fatalf("Failed to get Tester role ID: %v", err)
	}

	// Remove Godfather role (if exists) and its designations
	var godRoleID string
	err = pool.QueryRow(ctx, "SELECT id FROM public.roles WHERE name = 'Godfather'").Scan(&godRoleID)
	if err == nil {
		// Clean up user mappings for Godfather role
		pool.Exec(ctx, "DELETE FROM public.user_roles WHERE role_id = $1", godRoleID)
		pool.Exec(ctx, "DELETE FROM public.designations WHERE role_id = $1", godRoleID)
		pool.Exec(ctx, "DELETE FROM public.roles WHERE id = $1", godRoleID)
	}

	// Insert Designations for Developer and Tester
	designationQuery1 := `INSERT INTO public.designations (role_id, name) VALUES ($1, 'Godfather') ON CONFLICT (role_id, name) DO NOTHING;`
	if _, err := pool.Exec(ctx, designationQuery1, devRoleID); err != nil {
		log.Fatalf("Failed to insert Godfather designation: %v", err)
	}

	designationQuery2 := `INSERT INTO public.designations (role_id, name) VALUES ($1, 'Tester') ON CONFLICT (role_id, name) DO NOTHING;`
	if _, err := pool.Exec(ctx, designationQuery2, testerRoleID); err != nil {
		log.Fatalf("Failed to insert Tester designation: %v", err)
	}

	// 2. Create Superuser (Ankit)
	email := strings.ToLower("ankit.s2117@gmail.com")
	hash, _ := bcrypt.GenerateFromPassword([]byte("12345670"), bcrypt.DefaultCost)

	var userID string
	err = pool.QueryRow(ctx, "SELECT id FROM public.users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		// Insert User
		err = pool.QueryRow(ctx, `
			INSERT INTO public.users (email, password_hash, first_name, last_name, full_name, is_active)
			VALUES ($1, $2, 'Ankit', 'Singh', 'Ankit Singh', true)
			RETURNING id
		`, email, string(hash)).Scan(&userID)
		if err != nil {
			log.Fatalf("Failed to insert superuser: %v", err)
		}
	} else {
		// Update password if exists
		pool.Exec(ctx, "UPDATE public.users SET password_hash = $1 WHERE id = $2", string(hash), userID)
	}

	// Assign Godfather Designation to Ankit
	var godDesignationID string
	err = pool.QueryRow(ctx, "SELECT id FROM public.designations WHERE role_id = $1 AND name = 'Godfather'", devRoleID).Scan(&godDesignationID)
	if err != nil {
		log.Fatalf("Failed to get Godfather designation ID: %v", err)
	}

	// Delete old roles for this user and insert the new one
	pool.Exec(ctx, "DELETE FROM public.user_roles WHERE user_id = $1", userID)
	_, err = pool.Exec(ctx, "INSERT INTO public.user_roles (user_id, role_id, designation_id) VALUES ($1, $2, $3)", userID, devRoleID, godDesignationID)
	if err != nil {
		log.Fatalf("Failed to assign role to superuser: %v", err)
	}

	fmt.Println("Successfully ran RBAC migrations and created superuser!")
}
