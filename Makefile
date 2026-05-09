SHELL := /bin/sh

include .env
export

GO ?= go
DOCKER_COMPOSE ?= docker compose

.PHONY: help up down doctor-run appointment-run notification-run mock-run migrate-up

help:
	@printf "%s\n" "Available targets:" \
		"  up                     Start infrastructure and all services" \
		"  down                   Stop infrastructure" \
		"  migrate-up             Run migrations for both services"

up:
	$(DOCKER_COMPOSE) up -d
	@echo "Waiting for DBs..."
	@sleep 10
	$(MAKE) -j 4 mock-run doctor-run appointment-run notification-run

down:
	$(DOCKER_COMPOSE) down

doctor-run:
	cd doctor-service && $(GO) run ./cmd/doctor-service

appointment-run:
	cd appointment-service && $(GO) run ./cmd/appointment-service

notification-run:
	cd notification-service && $(GO) run ./cmd/notification-service

mock-run:
	cd mock-gateway && $(GO) run .

migrate-up:
	docker run --rm --network host -v "$(PWD)/doctor-service/migrations:/migrations" migrate/migrate:v4.19.1 -path=/migrations -database "$(DOCTOR_DATABASE_URL)" up
	docker run --rm --network host -v "$(PWD)/appointment-service/migrations:/migrations" migrate/migrate:v4.19.1 -path=/migrations -database "$(APPOINTMENT_DATABASE_URL)" up