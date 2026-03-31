# AP2 Assignment 1: Doctor + Appointment Microservices (Go, Clean Architecture)

## Project Overview
This project implements a small medical scheduling platform split into two REST microservices:
- `doctor-service`: owns and manages doctor profiles.
- `appointment-service`: owns and manages appointments, and validates doctor existence through HTTP calls to `doctor-service`.

The design follows Clean Architecture inside each service:
- `model` (domain entities)
- `usecase` (business rules)
- `repository` (data access)
- `transport/http` (delivery layer)
- `app` (dependency wiring)

## Service Responsibilities
- `doctor-service`
  - Create doctor
  - Get doctor by ID
  - List doctors
  - Enforce required `full_name`, required `email`, unique `email`
- `appointment-service`
  - Create appointment
  - Get appointment by ID
  - List appointments
  - Update appointment status
  - Enforce required `title`, required `doctor_id`
  - Validate doctor existence via REST call to `doctor-service`
  - Enforce status values: `new`, `in_progress`, `done`
  - Block transition `done -> new`

## Architecture Diagram
```mermaid
flowchart LR
    Client["Client (Postman/curl)"] --> DS["Doctor Service :8081"]
    Client --> AS["Appointment Service :8082"]
    AS -->|GET /doctors/{id}| DS
    DS --> DDB["Doctor Data (owned by doctor-service)"]
    AS --> ADB["Appointment Data (owned by appointment-service)"]
```

## Why This Is Microservices (Not Distributed Monolith)
- Each service has separate codebase, module, and in-memory repository (separate data ownership).
- `appointment-service` does not directly read/write doctor storage.
- Cross-service interaction happens only through explicit REST API boundary.
- Business logic remains local to each bounded context.

## Inter-Service Communication Contract
- Endpoint used by `appointment-service` for validation:
  - `GET http://localhost:8081/doctors/{id}`
- Interpretation:
  - `200` => doctor exists
  - `404` => doctor does not exist
  - timeout/network/5xx => dependency unavailable

## Failure Scenario (Required)
If `doctor-service` is down during appointment create/update:
- `appointment-service` does **not** proceed with operation.
- Returns `503 Service Unavailable` with descriptive error.
- Logs failure internally.

Production improvements point:
- timeout tuning
- retries (with backoff)
- circuit breaker for repeated failures

## Folder Structure
```text
assignment1/
├── doctor-service/
│   ├── main.go
│   ├── go.mod
│   └── internal/
│       ├── app/
│       ├── model/
│       ├── repository/
│       ├── transport/http/
│       └── usecase/
├── appointment-service/
│   ├── main.go
│   ├── go.mod
│   └── internal/
│       ├── app/
│       ├── client/
│       ├── model/
│       ├── repository/
│       ├── transport/http/
│       └── usecase/
└── README.md
```

## Run Locally
Open two terminals.

1. Start Doctor Service:
```bash
cd doctor-service
go run .
```

2. Start Appointment Service:
```bash
cd appointment-service
go run .
```

Default ports:
- Doctor Service: `8081`
- Appointment Service: `8082`

### Optional Environment Variables
- Doctor Service:
  - `DOCTOR_SERVICE_PORT` (default: `8081`)
- Appointment Service:
  - `APPOINTMENT_SERVICE_PORT` (default: `8082`)
  - `DOCTOR_SERVICE_URL` (default: `http://localhost:8081`)
  - `DOCTOR_SERVICE_TIMEOUT_MS` (default: `2000`)

## API Examples
### Create Doctor
```bash
curl -X POST http://localhost:8081/doctors \
  -H "Content-Type: application/json" \
  -d "{\"full_name\":\"Dr. Aisha Seitkali\",\"specialization\":\"Cardiology\",\"email\":\"a.seitkali@clinic.kz\"}"
```

### Get Doctor
```bash
curl http://localhost:8081/doctors/doctor-1
```

### List Doctors
```bash
curl http://localhost:8081/doctors
```

### Create Appointment
```bash
curl -X POST http://localhost:8082/appointments \
  -H "Content-Type: application/json" \
  -d "{\"title\":\"Initial cardiac consultation\",\"description\":\"Patient referred\",\"doctor_id\":\"doctor-1\"}"
```

### Get Appointment
```bash
curl http://localhost:8082/appointments/appointment-1
```

### List Appointments
```bash
curl http://localhost:8082/appointments
```

### Update Appointment Status
```bash
curl -X PATCH http://localhost:8082/appointments/appointment-1/status \
  -H "Content-Type: application/json" \
  -d "{\"status\":\"in_progress\"}"
```

### Failure Demo (Doctor Service Down)
Stop `doctor-service`, then call:
```bash
curl -X POST http://localhost:8082/appointments \
  -H "Content-Type: application/json" \
  -d "{\"title\":\"Follow-up\",\"description\":\"Checkup\",\"doctor_id\":\"doctor-1\"}"
```
Expected: `503` and descriptive error message.
