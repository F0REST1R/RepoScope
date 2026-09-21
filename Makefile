.PHONY: help dev up down logs build test test-race test-cover fmt lint frontend-install frontend-build compose-check

help:
	@echo "RepoScope: make up | dev | test | lint | build | down"

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f api web db

dev:
	go run ./cmd/api

build:
	go build ./cmd/api
	cd web && npm run build

test:
	go test ./...

test-race:
	CGO_ENABLED=1 go test -race ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

fmt:
	gofmt -w cmd internal

lint:
	golangci-lint run ./...
	cd web && npm run build

frontend-install:
	cd web && npm ci

frontend-build:
	cd web && npm run build

compose-check:
	docker compose config --quiet
