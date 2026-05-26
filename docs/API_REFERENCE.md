# VitalCache Backend — API REFERENCE
Version: 0.9.0  
Last Updated: 2025-11-07  
Maintainer: @THE-AkS-21  
Status: DRAFT

---

**Error envelope**
```json
{ "error": { "code": "forbidden", "message":"...", "details": {} } }
```

### Auth
POST `/api/auth/register`  
POST `/api/auth/login` → `{accessToken}` + refresh cookie  
POST `/api/auth/refresh` → `{accessToken}` (rotates cookie)  
POST `/api/auth/logout` → 204

### Public
GET `/api/medicines`

### Doctor (JWT)
GET `/api/v1/profiles/me`  
POST `/api/v1/patients`  
GET `/api/v1/patients/search?mobile=`  
GET `/api/v1/patients/:id`  
PATCH `/api/v1/patients/:id`  
GET `/api/v1/patients/:id/prescriptions?start=&end=&limit=&offset=`  
POST `/api/v1/prescriptions`
