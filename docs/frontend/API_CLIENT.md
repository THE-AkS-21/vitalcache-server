# Frontend — API Client
Version: 0.9.0  
Last Updated: 2025-11-07  
Maintainer: @THE-AkS-21  
Status: DRAFT

---

<details><summary><b>axios interceptor (example)</b></summary>

```ts
api.interceptors.response.use(undefined, async (err) => {
  if (err.response?.status === 401) {
    await api.post('/api/auth/refresh', {}, { withCredentials: true });
    err.config.headers.Authorization = 'Bearer ' + getAccess();
    return api.request(err.config);
  }
  throw err;
});
```
</details>
