# Testing Artifact - Assignment 4

This document provides exact commands and expected outputs to verify the functionality of Assignment 4.

## 1. Caching Verification

### 1.1 GetDoctor (Cache-Aside)
**Command:**
```bash
grpcurl -plaintext -d '{"id": "DOC_ID_HERE"}' localhost:50051 doctor.DoctorService/GetDoctor
```
**First Call (Miss):**
- `redis-cli MONITOR` shows: `GET doctor:DOC_ID_HERE` (nil), then `SET doctor:DOC_ID_HERE ...`.
**Second Call (Hit):**
- `redis-cli MONITOR` shows: `GET doctor:DOC_ID_HERE` (returns data). No DB query in service logs.

### 1.2 ListDoctors & Invalidation
**Command (List):**
```bash
grpcurl -plaintext localhost:50051 doctor.DoctorService/ListDoctors
```
**Command (Create - Invalidation):**
```bash
grpcurl -plaintext -d '{"full_name": "Dr. Test", "specialization": "General", "email": "test@clinic.kz"}' localhost:50051 doctor.DoctorService/CreateDoctor
```
- `redis-cli MONITOR` shows: `DEL doctors:list`.

---

## 2. Rate Limiting Verification

**Command (Run 101 times):**
```bash
for i in {1..101}; do grpcurl -plaintext localhost:50051 doctor.DoctorService/ListDoctors; done
```
**Expected Output on 101st request:**
```
Error: rpc error: code = ResourceExhausted desc = rate limit exceeded: 100 requests per minute allowed. Try again later.
```

---

## 3. Job Queue Verification

### 3.1 Successful Job
**Command:**
```bash
grpcurl -plaintext -d '{"id": "APP_ID_HERE", "status": "done"}' localhost:50052 appointment.AppointmentService/UpdateAppointmentStatus
```
**Expected Notification Service stdout:**
```json
{"time":"...","level":"info","job_id":"...","attempt":1,"status":"enqueued"}
{"time":"...","level":"info","job_id":"...","attempt":1,"status":"processing"}
{"time":"...","level":"info","job_id":"...","attempt":1,"status":"success"}
```
**Expected Mock Gateway stdout:**
```json
{"time":"...","request":{"idempotency_key":"...","channel":"email","recipient":"patient@clinic.kz","message":"..."}}
```

### 3.2 Retry Logic (Simulated by Gateway 503)
If the gateway returns 503 (20% chance), observe:
```json
{"time":"...","level":"warn","job_id":"...","attempt":1,"status":"retry","error":"gateway returned status 503"}
{"time":"...","level":"info","job_id":"...","attempt":2,"status":"processing"}
```

### 3.3 Idempotency
**Action**: Replay the NATS message or call `UpdateAppointmentStatus` to `done` again.
**Expected Notification Service stdout:**
```json
{"time":"...","level":"info","job_id":"...","status":"dropped","error":"already processed"}
```

### 3.4 Dead Letter Log
**Action**: Stop `mock-gateway` and call `UpdateAppointmentStatus` to `done`.
**Expected Notification Service stderr:**
```json
{"time":"...","level":"error","job_id":"...","status":"dead_letter","attempt":3,"error":"max retries reached: ..."}
```
