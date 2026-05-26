# JWT Key Rotation
Version: 0.9.0  
Last Updated: 2025-11-07  
Maintainer: @THE-AkS-21  
Status: DRAFT

---

```mermaid
flowchart LR
    Start([Start/Ticker]) --> Load[Load Secret JSON from AWS SM]
    Load --> Check{active >= 7d?}
    Check -- Yes --> Gen[Generate new key + kid]
    Gen --> Retain[Keep last 2 keys]
    Retain --> Put[PutSecretValue]
    Check -- No --> Done
    Put --> Done
```

<details><summary><b>Go: rotation ticker (excerpt)</b></summary>

```go
go func() {
  t := time.NewTicker(24 * time.Hour)
  for range t.C {
    _ = jwt.RotateIfNeeded(context.Background(), keySrc)
  }
}()
```
</details>
