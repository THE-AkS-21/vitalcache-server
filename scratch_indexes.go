package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dbURL := "postgresql://postgres.mbsqvihwuroltzxgysxq:rHirELm5uGLUonAD@aws-1-ap-south-1.pooler.supabase.com:5432/postgres?sslmode=require"
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	query := `
	-- 1. Users Indexes
	CREATE INDEX IF NOT EXISTS idx_users_email ON public.users(email);
	CREATE INDEX IF NOT EXISTS idx_users_first_name ON public.users(first_name);
	CREATE INDEX IF NOT EXISTS idx_users_last_name ON public.users(last_name);

	-- 2. Patients Indexes
	CREATE INDEX IF NOT EXISTS idx_patients_user_id ON public.patients(user_id);

	-- 3. Appointments Indexes
	CREATE INDEX IF NOT EXISTS idx_appointments_patient_id ON public.appointments(patient_id);
	CREATE INDEX IF NOT EXISTS idx_appointments_doctor_id ON public.appointments(doctor_id);
	CREATE INDEX IF NOT EXISTS idx_appointments_appointment_time ON public.appointments(appointment_time);

	-- 4. Billings Indexes
	CREATE INDEX IF NOT EXISTS idx_billings_patient_id ON public.billings(patient_id);
	CREATE INDEX IF NOT EXISTS idx_billings_doctor_id ON public.billings(doctor_id);
	CREATE INDEX IF NOT EXISTS idx_billings_created_at ON public.billings(created_at);
	`

	_, err = pool.Exec(ctx, query)
	if err != nil {
		log.Fatalf("Failed to execute migrations: %v", err)
	}

	fmt.Println("Successfully created database indexes!")
}
