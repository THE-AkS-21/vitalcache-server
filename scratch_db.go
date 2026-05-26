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
	-- 1. Create Billings Table
	CREATE TABLE IF NOT EXISTS public.billings (
		id uuid NOT NULL DEFAULT uuid_generate_v4(),
		patient_id uuid NOT NULL,
		doctor_id uuid NOT NULL,
		medical_report_id character varying,
		amount numeric(10,2) NOT NULL DEFAULT 0.00,
		status character varying DEFAULT 'PENDING',
		created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
		updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT billings_pkey PRIMARY KEY (id),
		CONSTRAINT billings_patient_id_fkey FOREIGN KEY (patient_id) REFERENCES public.patients(id) ON DELETE CASCADE,
		CONSTRAINT billings_doctor_id_fkey FOREIGN KEY (doctor_id) REFERENCES public.doctors(id) ON DELETE CASCADE
	);

	-- 2. Add consultation_fee to doctors if it doesn't exist
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='doctors' AND column_name='consultation_fee') THEN
			ALTER TABLE public.doctors ADD COLUMN consultation_fee numeric(10,2) NOT NULL DEFAULT 0.00;
		END IF;
	END $$;

	-- 3. Insert Roles & Designations
	DO $$
	DECLARE
		v_dev_role_id uuid;
		v_test_role_id uuid;
		v_doc_role_id uuid;
		v_staff_role_id uuid;
		v_pat_role_id uuid;

		v_gf_des_id uuid;
		v_jd_test_des_id uuid;
		v_gd_test_des_id uuid;
		v_doc_des_id uuid;
		v_staff_des_id uuid;
		v_pat_des_id uuid;

		v_user_id uuid;
	BEGIN
		-- Ensure Roles exist
		INSERT INTO public.roles (name) VALUES ('Developer') ON CONFLICT (name) DO NOTHING;
		INSERT INTO public.roles (name) VALUES ('Tester') ON CONFLICT (name) DO NOTHING;
		INSERT INTO public.roles (name) VALUES ('Doctor') ON CONFLICT (name) DO NOTHING;
		INSERT INTO public.roles (name) VALUES ('Staff') ON CONFLICT (name) DO NOTHING;
		INSERT INTO public.roles (name) VALUES ('Patient') ON CONFLICT (name) DO NOTHING;

		SELECT id INTO v_dev_role_id FROM public.roles WHERE name = 'Developer';
		SELECT id INTO v_test_role_id FROM public.roles WHERE name = 'Tester';
		SELECT id INTO v_doc_role_id FROM public.roles WHERE name = 'Doctor';
		SELECT id INTO v_staff_role_id FROM public.roles WHERE name = 'Staff';
		SELECT id INTO v_pat_role_id FROM public.roles WHERE name = 'Patient';

		-- Ensure GodFather designation exists under Developer
		SELECT id INTO v_gf_des_id FROM public.designations WHERE name = 'GodFather' AND role_id = v_dev_role_id;
		IF v_gf_des_id IS NULL THEN
			INSERT INTO public.designations (name, role_id) VALUES ('GodFather', v_dev_role_id) RETURNING id INTO v_gf_des_id;
		END IF;

		-- Ensure Tester designations exist
		SELECT id INTO v_jd_test_des_id FROM public.designations WHERE name = 'Junior Developer' AND role_id = v_test_role_id;
		IF v_jd_test_des_id IS NULL THEN
			INSERT INTO public.designations (name, role_id) VALUES ('Junior Developer', v_test_role_id) RETURNING id INTO v_jd_test_des_id;
		END IF;

		SELECT id INTO v_gd_test_des_id FROM public.designations WHERE name = 'GodFather Doctor' AND role_id = v_test_role_id;
		IF v_gd_test_des_id IS NULL THEN
			INSERT INTO public.designations (name, role_id) VALUES ('GodFather Doctor', v_test_role_id) RETURNING id INTO v_gd_test_des_id;
		END IF;

		-- Other Designations
		SELECT id INTO v_doc_des_id FROM public.designations WHERE role_id = v_doc_role_id LIMIT 1;
		IF v_doc_des_id IS NULL THEN
			INSERT INTO public.designations (name, role_id) VALUES ('General Physician', v_doc_role_id) RETURNING id INTO v_doc_des_id;
		END IF;

		SELECT id INTO v_staff_des_id FROM public.designations WHERE role_id = v_staff_role_id LIMIT 1;
		IF v_staff_des_id IS NULL THEN
			INSERT INTO public.designations (name, role_id) VALUES ('Receptionist', v_staff_role_id) RETURNING id INTO v_staff_des_id;
		END IF;

		SELECT id INTO v_pat_des_id FROM public.designations WHERE role_id = v_pat_role_id LIMIT 1;
		IF v_pat_des_id IS NULL THEN
			INSERT INTO public.designations (name, role_id) VALUES ('Patient', v_pat_role_id) RETURNING id INTO v_pat_des_id;
		END IF;

		-- Delete old GodFather role if it exists and has no users (cleanup)
		DELETE FROM public.roles WHERE name = 'GodFather' AND NOT EXISTS (SELECT 1 FROM public.user_roles WHERE role_id = public.roles.id);

		-- ==========================================
		-- Seed Super User (Ankit)
		-- ==========================================
		INSERT INTO public.users (email, password_hash, first_name, last_name)
		VALUES ('ankit.s2117@gmail.com', '$2a$10$w3S24W.O0z8oFByxI8L9kOUWqQj6250I7y1.54i.9mJ45W0h2L9g6', 'Ankit', '') 
		ON CONFLICT (email) DO NOTHING;
		
		SELECT id INTO v_user_id FROM public.users WHERE email = 'ankit.s2117@gmail.com';
		INSERT INTO public.user_roles (user_id, role_id, designation_id) VALUES (v_user_id, v_dev_role_id, v_gf_des_id) ON CONFLICT DO NOTHING;

		-- ==========================================
		-- Seed Tester Users
		-- ==========================================
		INSERT INTO public.users (email, password_hash, first_name, last_name)
		VALUES ('junior_developer@tester.com', '$2a$10$w3S24W.O0z8oFByxI8L9kOUWqQj6250I7y1.54i.9mJ45W0h2L9g6', 'Junior', 'Tester')
		ON CONFLICT (email) DO NOTHING;
		SELECT id INTO v_user_id FROM public.users WHERE email = 'junior_developer@tester.com';
		INSERT INTO public.user_roles (user_id, role_id, designation_id) VALUES (v_user_id, v_test_role_id, v_jd_test_des_id) ON CONFLICT DO NOTHING;

		INSERT INTO public.users (email, password_hash, first_name, last_name)
		VALUES ('godfather_doctor@tester.com', '$2a$10$w3S24W.O0z8oFByxI8L9kOUWqQj6250I7y1.54i.9mJ45W0h2L9g6', 'GodFather', 'Tester')
		ON CONFLICT (email) DO NOTHING;
		SELECT id INTO v_user_id FROM public.users WHERE email = 'godfather_doctor@tester.com';
		INSERT INTO public.user_roles (user_id, role_id, designation_id) VALUES (v_user_id, v_test_role_id, v_gd_test_des_id) ON CONFLICT DO NOTHING;

		-- ==========================================
		-- Seed Regular Users
		-- ==========================================
		-- Doctor User
		INSERT INTO public.users (email, password_hash, first_name, last_name)
		VALUES ('test_user@doctor.com', '$2a$10$w3S24W.O0z8oFByxI8L9kOUWqQj6250I7y1.54i.9mJ45W0h2L9g6', 'Test', 'Doctor')
		ON CONFLICT (email) DO NOTHING;
		
		SELECT id INTO v_user_id FROM public.users WHERE email = 'test_user@doctor.com';
		INSERT INTO public.user_roles (user_id, role_id, designation_id) VALUES (v_user_id, v_doc_role_id, v_doc_des_id) ON CONFLICT DO NOTHING;
		INSERT INTO public.doctors (user_id, specialization) VALUES (v_user_id, 'General Physician') ON CONFLICT DO NOTHING;

		-- Staff User
		INSERT INTO public.users (email, password_hash, first_name, last_name)
		VALUES ('test_user@staff.com', '$2a$10$w3S24W.O0z8oFByxI8L9kOUWqQj6250I7y1.54i.9mJ45W0h2L9g6', 'Test', 'Staff')
		ON CONFLICT (email) DO NOTHING;
		
		SELECT id INTO v_user_id FROM public.users WHERE email = 'test_user@staff.com';
		INSERT INTO public.user_roles (user_id, role_id, designation_id) VALUES (v_user_id, v_staff_role_id, v_staff_des_id) ON CONFLICT DO NOTHING;

		-- Patient User
		INSERT INTO public.users (email, password_hash, first_name, last_name)
		VALUES ('test_user@patient.com', '$2a$10$w3S24W.O0z8oFByxI8L9kOUWqQj6250I7y1.54i.9mJ45W0h2L9g6', 'Test', 'Patient')
		ON CONFLICT (email) DO NOTHING;
		
		SELECT id INTO v_user_id FROM public.users WHERE email = 'test_user@patient.com';
		INSERT INTO public.user_roles (user_id, role_id, designation_id) VALUES (v_user_id, v_pat_role_id, v_pat_des_id) ON CONFLICT DO NOTHING;
		INSERT INTO public.patients (user_id) VALUES (v_user_id) ON CONFLICT DO NOTHING;

	END $$;
	`

	_, err = pool.Exec(ctx, query)
	if err != nil {
		log.Fatalf("Failed to execute migrations: %v", err)
	}

	fmt.Println("Successfully ran migrations and seeded GodFather user!")
}
