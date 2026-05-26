# Services
Version: 0.9.0  
Last Updated: 2025-11-07  
Maintainer: @THE-AkS-21  
Status: DRAFT

---

**PatientsService**
- `Create(ctx, dto, user)` → insert patient with owner doctor_id
- `IsDoctorLinked(ctx, doctorID, patientID)` → (owner or link exists)

**PrescriptionsService**
- `Create(ctx, dto, doctorID)` → insert prescription (RLS verifies link)
- (future) enqueue notify task
