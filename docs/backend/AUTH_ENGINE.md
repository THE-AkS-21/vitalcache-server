# Auth Engine
Version: 0.9.0  
Last Updated: 2025-11-07  
Maintainer: @THE-AkS-21  
Status: DRAFT

---

- Refresh cookie: HttpOnly; Secure; SameSite=Strict; Path=/api/auth
- Access token: Authorization: Bearer <JWT>
- Claims: `sub`, `role`, `doctor_id`
- Include `doctor_id` to avoid DB hop on each call.
