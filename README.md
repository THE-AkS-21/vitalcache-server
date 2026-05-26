
# VitalCache – Clinic Management Backend (Go + Gin + Supabase)

VitalCache is a production‑ready, scalable backend for a clinic/hospital workflow. It manages **users, doctors, patients, medicines, prescriptions**, and enforces **RBAC** (role-based access control) with **JWT** that **auto‑rotates keys every 7 days**. Data is stored in **PostgreSQL via Supabase**, with **RLS** (Row‑Level Security) for defense‑in‑depth. Observability is built‑in: **Prometheus metrics** and **OpenTelemetry tracing**.

---

## Table of Contents

- [Highlights](#highlights)
- [Architecture](#architecture)
- [Project Layout](#project-layout)
- [Domain Model](#domain-model)
- [Security & RBAC](#security--rbac)
- [JWT Key Rotation](#jwt-key-rotation)
- [AWS Secrets Manager](#aws-secrets-manager)
- [API Surface](#api-surface)
- [Running Locally](#running-locally)
- [Docker](#docker)
- [Configuration](#configuration)
- [Observability](#observability)
- [Queue](#queue)
- [Database Notes (Supabase)](#database-notes-supabase)
- [Postman Collection](#postman-collection)
- [Troubleshooting](#troubleshooting)
- [Roadmap](#roadmap)
- [License](#license)

---

## Highlights

- **Go 1.25 + Gin** with clean layering and DI via simple constructors.
- **RBAC**: Doctor‑only routes for patient management and prescriptions.
- **JWT** (HS256) with **key rotation every 7 days**, old key retained for previously issued tokens.
- **Supabase (Postgres)** for storage; **RLS** policies recommended for least privilege.
- **Prescription history** persisted to DB and filterable by date range.
- **Metrics** (`/metrics`, Prometheus) and **Tracing** (otelgin + optional OTLP exporter).
- **Rate limiting**, **request ID**, **structured logs**, **panic recovery** middlewares.
- **In-memory queue** (simple, no external deps for now).

Target SLOs (non-functional):
- p99 latency ≤ **250 ms**
- throughput **500 RPS**
- availability **99.9%**

---

## Architecture

```
┌────────────┐       ┌───────────────┐        ┌──────────────┐
│   Client   │ ───▶  │   Gin HTTP    │  ───▶  │   Services   │
└────────────┘       │  (middleware) │        └──────────────┘
                     │  auth/rbac    │               │
                     │  logging      │               ▼
                     │  metrics/trx  │        ┌──────────────┐
                     └──────┬────────┘        │    Stores    │
                            │                 └──────┬───────┘
                            ▼                        │
                     ┌──────────────┐               ▼
                     │   Queue      │      ┌────────────────────┐
                     │  (in‑memory) │      │ Supabase (Postgres)│
                     └──────────────┘      └────────────────────┘
```

- **Handlers** are thin (validation & HTTP concerns).
- **Services** hold business logic and orchestrate **Stores** and **Queue**.
- **Stores** talk to Supabase/PostgREST.
- **Middlewares** add auth, RBAC, rate‑limit, logging, metrics, tracing.

---

## Project Layout

```
vitalcache-server/
├── cmd/
│   └── api/
│       └── main.go                  # bootstrap, secrets, tracing/metrics, server
├── internal/
│   ├── app/
│   │   ├── server.go                # gin engine + middlewares + /metrics
│   │   ├── routes/
│   │   │   └── routes.go            # API routes wiring
│   │   └── middlewares/             # auth, rbac, logger, recovery, reqid, ratelimit
│   ├── domain/                      # core models & policy helpers
│   │   ├── developers.go
│   │   ├── doctors.go
│   │   ├── medicines.go
│   │   ├── patients.go
│   │   ├── prescriptions.go
│   │   └── policies/
│   │       ├── rbac.go
│   │       └── ownership.go
│   ├── http/
│   │   ├── dto/                     # request/query DTOs
│   │   │   ├── auth.go
│   │   │   ├── patients.go
│   │   │   └── prescriptions.go
│   │   └── handlers/                # HTTP handlers
│   │       ├── auth.go
│   │       ├── medicines.go
│   │       ├── patients.go
│   │       ├── prescriptions.go
│   │       ├── profiles.go
│   │       └── hdeps/deps.go
│   ├── observability/               # logging, metrics, tracing
│   │   ├── logging.go
│   │   ├── metrics.go
│   │   └── tracing.go
│   ├── queue/
│   │   └── inmem.go                 # simple in‑memory worker
│   ├── repo/
│   │   └── users_repo.go            # repository interfaces (extensible)
│   ├── service/
│   │   ├── auth/service.go
│   │   ├── patients/service.go
│   │   └── prescriptions/service.go
│   └── store/
│       └── supabase/
│           ├── client.go
│           ├── developers_store.go
│           ├── doctors_store.go
│           ├── medicines_store.go
│           ├── patients_store.go
│           └── prescriptions_store.go
└── pkg/
    ├── config/aws_secrets.go        # Secrets client + struct
    ├── jwt/
    │   ├── jwt_rotator.go           # 7‑day rotation with KID; persists in secrets
    │   └── jwt_tokens.go            # mint/validate helpers
    └── utils/password.go            # bcrypt helpers
```

---

## Domain Model

**Primary roles:** `doctor`, `developer`, `patient` (on `users` table).  
**Developer roles (secondary):** `god`, `godfather`, `senior`, `junior`, `intern` (on `developers`).  
**Hospital staff (future):** `owner`, `admin`, `staff`.

**Entities** (simplified):
- `users(id, email, password_hash, role)`
- `doctors(id, name, designation, user_id)`
- `developers(id, name, role, user_id)`
- `patients(id, name, age, mobile_number, email, doctor_id, ...)`
- `medicines(id, name, dose, duration_days, frequency, ...)`
- `prescriptions(id, patient_id, doctor_id, file_url, sent_at)`
- `patient_doctors(patient_id, doctor_id)` (optional link for multi‑doctor care)

Indexes to consider:
- `idx_patients_mobile_number on patients(mobile_number)`
- `idx_prescriptions_patient_sent_at on prescriptions(patient_id, sent_at desc)`

---

## Security & RBAC

- **JWT** (HS256, with `kid` header) for stateless auth. Claims contain `sub` (user_id) and `role`.
- **Middleware**:
  - `Auth(ks)` → validates token and injects `user_id` + `user_role` into context.
  - `RequireDoctor()` → guards doctor‑only routes (patients & prescriptions).
- **Ownership** via `policies.DoctorCanAccessPatient`:
  - If `patient_doctors` exists → allow linked doctors.
  - Else fallback to `patients.doctor_id = doctorID`.

Supabase **RLS** should mirror RBAC for DB-level enforcement (see [Database Notes](#database-notes-supabase)).

---

## JWT Key Rotation

- Keys live in **AWS Secrets Manager** (see format below).
- Rotation policy: **every 7 days** a new key is generated, becomes **active**, the **previous** key is retained for validation of old tokens.
- **kid** in JWT header identifies which key to use on validation.
- A background ticker checks daily and rotates if needed.

**Token lifetime:** access token: **7 days** (matches key rotation).

---

## AWS Secrets Manager

### Secret name
```
AWS_SECRET_ID=vitalcache/prod   # configurable
AWS_REGION=ap-south-1           # example
```

### Secret JSON format (example)
```json
{
  "SupabaseURL": "https://YOUR_PROJECT.supabase.co",
  "SupabaseKey": "SUPABASE_SERVICE_ROLE_OR_ANON",
  "JWT": {
    "ActiveKID": "kid-2025-11-01",
    "Keys": {
      "kid-2025-11-01": "base64url-encoded-secret-1",
      "kid-2025-10-25": "base64url-encoded-secret-0"
    },
    "RotatesEveryDays": 7
  }
}
```

- `ActiveKID` → used to **sign** new tokens.
- `Keys` → map of **kid → HS256 secret**. Keep **at least 2** (current + previous).
- The app reads this secret on boot and persists updates on rotation.

---

## API Surface

**Base URL:** `http://localhost:8080`

### Public
- `POST /api/auth/register`
- `POST /api/auth/login`
- `GET  /api/medicines`

### Protected (JWT required; **doctor-only**)
- `GET  /api/v1/profiles/me`
- `POST /api/v1/patients`
- `GET  /api/v1/patients/search?mobile=...`
- `GET  /api/v1/patients/:id`
- `PATCH /api/v1/patients/:id`
- `POST /api/v1/prescriptions/send` (multipart; queues + persists)
- `GET  /api/v1/patients/:id/prescriptions?start=&end=&limit=&offset=`

**OpenAPI**: see the generated OpenAPI 3.1 spec (provided earlier in this repo/discussion).

---

## Running Locally

### Prereqs
- Go **1.25+**
- Supabase project (URL + Key)
- AWS IAM permission to read/write the secret (if testing rotation)
- (Optional) Docker for containerized run

### Env
Create local `.env` only if you want to override **non-secret** settings (recommended to keep secrets in AWS). Example:

```dotenv
PORT=8080
GIN_MODE=debug
OTEL_EXPORTER_OTLP_ENDPOINT= # e.g., localhost:4318
AWS_REGION=ap-south-1
AWS_SECRET_ID=vitalcache/prod
```

### Build & run
```bash
go mod tidy
go build ./...
go run ./cmd/api
```

The server starts on `:8080`.
- Metrics at `GET /metrics`
- Traces if `OTEL_EXPORTER_OTLP_ENDPOINT` is set.

---

## Docker

Build:
```bash
docker build -t vitalcache:local .
```

Run:
```bash
docker run --rm -p 8080:8080 \
  -e AWS_REGION=ap-south-1 \
  -e AWS_SECRET_ID=vitalcache/prod \
  -e OTEL_EXPORTER_OTLP_ENDPOINT= \
  vitalcache:local
```

> Provide container IAM creds (env or task role) to access Secrets Manager.

---

## Configuration

Key env vars:
- `PORT` – default `8080`
- `GIN_MODE` – `debug` or `release`
- `AWS_REGION` – AWS region for Secrets Manager
- `AWS_SECRET_ID` – secret name/ARN
- `OTEL_EXPORTER_OTLP_ENDPOINT` – optional, e.g. `localhost:4318`

Non‑secret config (e.g., feature flags) may also live in the same secret payload if desired.

---

## Observability

- **Metrics**: Prometheus at `/metrics`. Includes request counters and latency histograms labeled by method/path/status.
- **Tracing**: `otelgin` middleware; set `OTEL_EXPORTER_OTLP_ENDPOINT` to export to Jaeger/Tempo/OTel Collector.
- **Logging**: Go `slog` with JSON in prod; `lumberjack` rotation recommended if file output is enabled.

---

## Queue

- Current implementation: **in‑memory** worker (`internal/queue/inmem.go`).
- Uploading a prescription:
  1. Saves the file temporarily.
  2. Best‑effort upload to Supabase Storage (path recorded as `file_url`).
  3. Persists a `prescriptions` row (patient_id, doctor_id, file_url, sent_at).
  4. Enqueues a job to the in‑memory worker (which logs and deletes the temp file).

> Later you can switch to Redis/Asynq with minimal changes (implement the same `queue.Client` interface).

---

## Database Notes (Supabase)

Recommended tables (simplified):
- `users`, `doctors`, `developers`, `patients`, `medicines`, `prescriptions`
- Optional: `patient_doctors` (many‑to‑many: patient ↔ doctor)

Recommended RLS examples:
- **Prescriptions SELECT** allowed iff:
  - caller is the owning doctor for the patient (`patients.doctor_id`), or
  - there exists a link row in `patient_doctors` (caller is a treating doctor).
- **Prescriptions INSERT** allowed iff caller is an owning or linked doctor.
- **Patients SELECT/UPDATE** similarly constrained.

> Ensure `auth.uid()` maps to your `users.id` (or pass user id via PostgREST with service key only in backend).

Indexes:
```sql
create index if not exists idx_patients_mobile_number on patients(mobile_number);
create index if not exists idx_prescriptions_patient_sent_at on prescriptions (patient_id, sent_at desc);
```

---

## Postman Collection

A ready-to-import collection is included:
- **[Download VitalCache.postman_collection.json](sandbox:/mnt/data/VitalCache.postman_collection.json)**

Collection variables:
- `baseUrl` → default `http://localhost:8080`
- `authToken` → set by the **Login** request’s test script

---

## Troubleshooting

- **Mixed package names in a folder** → ensure every file in a folder uses the same `package` line.
- **Missing go.sum entries** → run `go mod tidy` after fixing all imports.
- **Unknown module version** → pin to a valid tag (e.g., remove Asynq until Redis is needed).
- **Supabase API mismatches** → note PostgREST API signatures (e.g., `.Order("col", &postgrest.OrderOpts{Ascending:true})`).

---

## Roadmap

- Redis/Asynq queue + durable retries
- Email delivery (SES/SMTP) with templates & delivery logs
- Prescription bundles CRUD + reuse
- Hospital staff hierarchy (owner/admin/staff) & org-scoped RBAC
- Developer hierarchy policies
- OpenAPI file committed to repo and CI contract tests
- CI/CD via GitHub Actions
- Caching layer for hot reads (e.g., patients, medicines)
- Pagination + consistent error types across handlers

---

## License

MIT (c) 2025 VitalCache Authors
