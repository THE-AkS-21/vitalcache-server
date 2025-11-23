# RLS Policies (Supabase SQL)
Version: 0.9.0  
Last Updated: 2025-11-07  
Maintainer: @THE-AkS-21  
Status: DRAFT

---

```sql
alter table patients enable row level security;

create or replace function request_header_int(key text)
returns int language sql stable as $$
  select nullif(trim((current_setting('request.headers', true)::jsonb ->> key)), '')::int
$$;

create policy "patients_read_by_link"
on patients for select using (
  current_setting('request.headers', true)::jsonb ->> 'X-Role' = 'doctor' and
  (
    doctor_id = request_header_int('X-Doctor-Id') or
    exists (select 1 from patient_doctors pd
            where pd.patient_id = patients.id
              and pd.doctor_id  = request_header_int('X-Doctor-Id'))
  )
);

create policy "patients_insert_by_owner"
on patients for insert with check (
  current_setting('request.headers', true)::jsonb ->> 'X-Role' = 'doctor' and
  doctor_id = request_header_int('X-Doctor-Id')
);
```
