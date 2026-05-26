# VitalCache Backend — DATABASE DESIGN
Version: 0.9.0  
Last Updated: 2025-11-07  
Maintainer: @THE-AkS-21  
Status: DRAFT

---

## ERD
```mermaid
erDiagram
  USERS ||--|| DOCTORS : "has profile"
  DOCTORS ||--o{ PATIENTS : "owns primary"
  PATIENTS ||--o{ PATIENT_DOCTORS : "links"
  DOCTORS ||--o{ PATIENT_DOCTORS : "links"
  PATIENTS ||--o{ PRESCRIPTIONS : "history"
  DOCTORS ||--o{ PRESCRIPTIONS : "writes"
  PRESCRIPTIONS }o--|| PRESCRIPTION_BUNDLES : "optional"
  PRESCRIPTION_BUNDLES }o--o{ BUNDLE_MEDICINES : "contains"
  MEDICINES ||--o{ BUNDLE_MEDICINES : "catalog"
```

## Tables (key points)
- `patients(mobile_number)` indexed; `(patient_id, created_at)` index on `prescriptions`
- RLS enabled on **all** business tables
- `patient_doctors(patient_id, doctor_id)` as link table for multi-doctor care
