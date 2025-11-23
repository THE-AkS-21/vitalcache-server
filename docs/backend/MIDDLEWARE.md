# App Middleware
Version: 0.9.0  
Last Updated: 2025-11-07  
Maintainer: @THE-AkS-21  
Status: DRAFT

---

Order: recovery → requestid → logger → tracing → metrics → ratelimit → CORS → auth

<details><summary><b>Logger (excerpt)</b></summary>

```go
start := time.Now()
c.Next()
slog.Info("http", "method", c.Request.Method, "path", c.FullPath(), "status", c.Writer.Status())
```
</details>
