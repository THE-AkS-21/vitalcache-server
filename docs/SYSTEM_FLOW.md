# VitalCache Backend — SYSTEM FLOW
Version: 0.9.0  
Last Updated: 2025-11-07  
Maintainer: @THE-AkS-21  
Status: DRAFT

---

## Request Lifecycle
1. recovery → requestid → logger → tracing → metrics → ratelimit → CORS → auth
2. handler validates DTO → service rules → repo → store → PostgREST (RLS)

```mermaid
sequenceDiagram
  participant C as Client
  participant G as Gin/MW
  participant H as Handler
  participant S as Service
  participant R as Repo
  participant P as PostgREST(RLS)

  C->>G: HTTP Request
  G->>H: pass (after auth/ratelimit/etc)
  H->>S: validate & call
  S->>R: business call
  R->>P: SQL over HTTP + RLS headers
  P-->>R: rows
  R-->>S: models
  S-->>H: DTO
  H-->>C: JSON Response
```

## Auth / Token Refresh
- Access 15m, Refresh 30d (HttpOnly, Secure, SameSite=Strict, Path=/api/auth)
- Server validates access against **active or retained** HS256 keys (rotated every 7d)
