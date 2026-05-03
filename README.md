# Medical Scheduling Platform - Assignment 3

## Overview
This version of the Medical Scheduling Platform introduces asynchronous event-driven communication using NATS and persistent storage using PostgreSQL. The system consists of three services:
1. **Doctor Service**: Manages doctor information and publishes `doctors.created` events.
2. **Appointment Service**: Manages appointments, validates doctors via gRPC, and publishes `appointments.created` and `appointments.status_updated` events.
3. **Notification Service**: Subscribes to all domain events and logs them in a structured JSON format.

## Architecture
![Architecture Diagram](architecture.png)

```mermaid
graph TD
    User((User))
    User -- gRPC --> AS[Appointment Service]
    User -- gRPC --> DS[Doctor Service]
    AS -- gRPC --> DS
    AS -- Events --> NATS((NATS Broker))
    DS -- Events --> NATS
    NATS -- Events --> NS[Notification Service]
    AS -- DB --> DBA[(Postgres: appointment_db)]
    DS -- DB --> DBD[(Postgres: doctor_db)]
```

## Broker Choice: NATS
I chose **NATS (Core)** as the message broker for the following reasons:
- **Simplicity**: NATS has a very lightweight setup and a simple Go client.
- **Performance**: It provides high-throughput, low-latency messaging.
- **Statelessness**: For the requirement of "stateless notifications" and "fire-and-forget" delivery, NATS Core is ideal.
- **Ease of Use**: It doesn't require complex exchange/queue declarations like RabbitMQ for simple Pub/Sub.

## Environment Variables
Each service uses the following environment variables:

| Variable | Description | Default (Example) |
| --- | --- | --- |
| `DATABASE_URL` | PostgreSQL connection string | `postgres://user:password@localhost:5433/doctor_db?sslmode=disable` |
| `NATS_URL` | NATS connection URL | `nats://localhost:4222` |
| `DOCTOR_SERVICE_PORT` | Port for Doctor gRPC server | `50051` |
| `APPOINTMENT_SERVICE_PORT` | Port for Appointment gRPC server | `50052` |
| `DOCTOR_SERVICE_ADDR` | Address of Doctor service (for Appointment client) | `localhost:50051` |

## Infrastructure Setup
To start the required infrastructure (PostgreSQL and NATS), run:
```bash
docker-compose up -d
```

## Migrations
Migrations are handled automatically by each service on startup using `golang-migrate`.
- **Doctor Service** migrations are in `doctor-service/migrations/`.
- **Appointment Service** migrations are in `appointment-service/migrations/`.

To roll back a migration manually (using `golang-migrate` CLI):
```bash
migrate -path ./migrations -database "$DATABASE_URL" down
```

## Service Startup Order
1. **Infrastructure**: Start NATS and PostgreSQL first.
2. **Doctor Service**: `cd doctor-service && go run .`
3. **Appointment Service**: `cd appointment-service && go run .`
4. **Notification Service**: `cd notification-service && go run ./cmd/notification-service`

## Event Contract
| Subject | Trigger | Payload Fields |
| --- | --- | --- |
| `doctors.created` | New doctor created | `event_type`, `occurred_at`, `id`, `full_name`, `specialization`, `email` |
| `appointments.created` | New appointment created | `event_type`, `occurred_at`, `id`, `title`, `doctor_id`, `status` |
| `appointments.status_updated` | Status changed | `event_type`, `occurred_at`, `id`, `old_status`, `new_status` |

## Testing with grpcurl

### 1. Create a Doctor
```bash
grpcurl -plaintext -d '{"full_name": "Dr. Aisha Seitkali", "specialization": "Cardiology", "email": "a.seitkali@clinic.kz"}' localhost:50051 doctor.DoctorService/CreateDoctor
```
**Expected Notification Log:**
```json
{"time":"2026-05-01T10:23:44Z","subject":"doctors.created","event":{"event_type":"doctors.created","occurred_at":"2026-05-01T10:23:44Z","id":"...","full_name":"Dr. Aisha Seitkali","specialization":"Cardiology","email":"a.seitkali@clinic.kz"}}
```

### 2. Create an Appointment
```bash
grpcurl -plaintext -d '{"title": "Initial cardiac consultation", "doctor_id": "DOCTOR_ID_HERE"}' localhost:50052 appointment.AppointmentService/CreateAppointment
```
**Expected Notification Log:**
```json
{"time":"2026-05-01T10:24:01Z","subject":"appointments.created","event":{"event_type":"appointments.created","occurred_at":"2026-05-01T10:24:01Z","id":"...","title":"Initial cardiac consultation","doctor_id":"...","status":"new"}}
```

### 3. Update Appointment Status
```bash
grpcurl -plaintext -d '{"id": "APPOINTMENT_ID_HERE", "status": "in_progress"}' localhost:50052 appointment.AppointmentService/UpdateAppointmentStatus
```
**Expected Notification Log:**
```json
{"time":"2026-05-01T10:25:10Z","subject":"appointments.status_updated","event":{"event_type":"appointments.status_updated","occurred_at":"2026-05-01T10:25:10Z","id":"...","old_status":"new","new_status":"in_progress"}}
```

## Consistency Trade-offs
- **Best-Effort Delivery**: Publishing to NATS is fire-and-forget. If the broker is down during an RPC, the event is lost, but the gRPC response still succeeds.
- **Reliability**: To improve reliability, the **Outbox Pattern** could be used, where events are stored in the same database transaction as the business data and then published by a background worker. Alternatively, NATS JetStream could provide durable streams and guaranteed delivery.

## Broker Comparison
- **NATS (Core)**: Lightweight, fire-and-forget Pub/Sub, no built-in persistence, extremely fast. Ideal for real-time notifications where occasional loss is acceptable.
- **RabbitMQ**: Supports durable queues, acknowledgments, and complex routing logic. Better for systems requiring "At-least-once" delivery and task processing.
