# VitalCache Server 🚀

The official Go and Gin backend for VitalCache, a modern clinic management system. This server provides a secure REST API for managing patient data, prescriptions, and doctor information, using Supabase (PostgreSQL) as the database.

## ✨ Key Features

- **Secure Authentication:** JWT-based authentication for doctors.
- **Patient Management:** Full CRUD (Create, Read, Update, Delete) functionality for patient records.
- **Prescription Handling:** API endpoints to create, view, and send digital prescriptions.
- **Scalable Architecture:** Built with Go and Gin for high performance and low resource consumption.

## 🛠️ Tech Stack

- **Language:** Go (Golang)
- **Framework:** Gin
- **Database:** Supabase (PostgreSQL)
- **Authentication:** JWT

## ⚙️ Setup and Installation

1.  **Clone the repository:**
    ```bash
    git clone git@github.com:THE-AkS-21/vitalcache-server.git
    cd vitalcache-server
    ```

2.  **Install dependencies:**
    ```bash
    go mod tidy
    ```

3.  **Set up environment variables:**
    Create a `.env` file in the root directory and add your Supabase credentials:
    ```env
    # Development Environment Settings
    PORT=8080
    GIN_MODE=debug

    # Supabase Credentials (Use your Transaction Pooler URI)
    SUPABASE_URL="YOUR_SUPABASE_PROJECT_URL"
    SUPABASE_KEY="YOUR_SUPABASE_ANON_PUBLIC_KEY"

    # JWT Secret Key
    JWT_SECRET="YOUR_SUPER_SECRET_KEY_FOR_JWT"
    ```

4.  **Run the server:**
    ```bash
    go run ./cmd/api  
    ```
    The server will start on `http://localhost:8080`.

##  API Endpoints

A brief overview of the main API endpoints:

- `POST /api/auth/register` - Register a new doctor.
- `POST /api/auth/login` - Login for an existing doctor.
- `GET /api/patients/search?mobile=<number>` - Search for patients by mobile number.
- `POST /api/patients` - Create a new patient.

---
