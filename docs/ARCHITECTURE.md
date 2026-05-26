# VitalCache Backend — ARCHITECTURE
Version: 0.9.0  
Last Updated: 2025-11-07  
Maintainer: @THE-AkS-21  
Status: DRAFT

---

**Stack:** Go + Gin, Supabase (PostgREST + RLS), AWS Secrets Manager (JWT keys), Prometheus, OpenTelemetry.  
**Goals:** p99 ≤ 250ms, 500 RPS, 99.9% uptime, strict RBAC+RLS.

```mermaid
flowchart TD
    FE[Next.js Client] --> API[Go/Gin API]
    subgraph API
      MW[Middlewares]
      H[Handlers]
      S[Service]
      R[Repo]
      ST[Store(PostgREST)]
    end
    ST -->|RLS headers| SUPA[(Supabase / PostgREST / PG)]
    API --> METRICS[(Prometheus /metrics)]
    API --> OTEL[(OTel -> Tempo/Jaeger)]
    API --> Q[(Queue: in-mem now, Redis later)]
```

**Folder map**
- `cmd/api`: main entry
- `internal/app`: server + middlewares
- `internal/http`: handlers + dto + errors
- `internal/domain`: core models
- `internal/service`: business logic
- `internal/repo`: interfaces
- `internal/store/supabase`: PostgREST client + queries
- `pkg/jwt`: keyring + rotation
- `pkg/config`: AWS secrets
