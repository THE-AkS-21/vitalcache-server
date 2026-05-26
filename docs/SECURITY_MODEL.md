# VitalCache Backend — SECURITY MODEL
Version: 0.9.0  
Last Updated: 2025-11-07  
Maintainer: @THE-AkS-21  
Status: DRAFT

---

## RBAC
- doctor: CRUD patients (owner), view linked, create prescriptions, view full history for linked
- developer: internal only (no PHI)
- patient (future): view own profile/prescriptions

## RLS (defense in depth)
- Headers forwarded: `X-User-Id`, `X-Role`, `X-Doctor-Id`
- Patients SELECT: owner OR linked
- Prescriptions SELECT/INSERT: any doctor linked to patient
- Immutable prescriptions (deny update/delete)

## Tokens
- Access 15m; Refresh 30d cookie
- HS256 keys rotated every 7d; retain last 2
