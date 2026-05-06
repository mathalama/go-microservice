SHELL := /bin/sh

include .env
export

GO ?= go
DOCKER_COMPOSE ?= docker compose
MIGRATE_IMAGE ?= migrate/migrate:v4.19.1

DOCTOR_DIR := doctor-service
APPOINTMENT_DIR := appointment-service
NOTIFICATION_DIR := notification-service

.PHONY: help infra-up infra-down doctor-run appointment-run notification-run doctor-migrate-up doctor-migrate-down appointment-migrate-up appointment-migrate-down migrate-up migrate-down

help:
	@printf "%s\n" "Available targets:" \
		"  infra-up               Start Postgres and NATS" \
		"  infra-down             Stop infrastructure" \
		"  doctor-run             Run Doctor Service" \
		"  appointment-run        Run Appointment Service" \
		"  notification-run       Run Notification Service" \
		"  doctor-migrate-up      Apply Doctor migrations" \
		"  doctor-migrate-down    Roll back one Doctor migration" \
		"  appointment-migrate-up Apply Appointment migrations" \
		"  appointment-migrate-down Roll back one Appointment migration" \
		"  migrate-up             Apply both services' migrations" \
		"  migrate-down           Roll back one migration in both services"

infra-up:
	$(DOCKER_COMPOSE) up -d

infra-down:
	$(DOCKER_COMPOSE) down

doctor-run:
	cd $(DOCTOR_DIR) && $(GO) run ./cmd/doctor-service

appointment-run:
	cd $(APPOINTMENT_DIR) && $(GO) run ./cmd/appointment-service

notification-run:
	cd $(NOTIFICATION_DIR) && $(GO) run ./cmd/notification-service | jq

doctor-migrate-up:
	docker run --rm --network host -v "$(PWD)/$(DOCTOR_DIR)/migrations:/migrations" $(MIGRATE_IMAGE) -path=/migrations -database "$(DOCTOR_DATABASE_URL)" up

doctor-migrate-down:
	docker run --rm --network host -v "$(PWD)/$(DOCTOR_DIR)/migrations:/migrations" $(MIGRATE_IMAGE) -path=/migrations -database "$(DOCTOR_DATABASE_URL)" down 1

appointment-migrate-up:
	docker run --rm --network host -v "$(PWD)/$(APPOINTMENT_DIR)/migrations:/migrations" $(MIGRATE_IMAGE) -path=/migrations -database "$(APPOINTMENT_DATABASE_URL)" up

appointment-migrate-down:
	docker run --rm --network host -v "$(PWD)/$(APPOINTMENT_DIR)/migrations:/migrations" $(MIGRATE_IMAGE) -path=/migrations -database "$(APPOINTMENT_DATABASE_URL)" down 1

migrate-up: doctor-migrate-up appointment-migrate-up

migrate-down: doctor-migrate-down appointment-migrate-down