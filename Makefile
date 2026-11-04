SHELL := /bin/bash
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
DATABASE_URL ?= postgres://refresco:refresco@localhost:5432/refresco?sslmode=disable
export DATABASE_URL

.PHONY: help dev api web seed test test-backend test-frontend lint build build-backend build-frontend docker-build up down logs clean dev-db seed

help: ## Show targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-16s\033[0m %s\n",$$1,$$2}'

api: ## Run the API locally in development mode (needs Postgres)
	cd backend && APP_ENV=development go run ./cmd/api

web: ## Run the Vite dev server (proxies /api to :8080)
	cd frontend && npm run dev

seed: ## Load demo data into a non-production database
	cd backend && APP_ENV=development go run ./cmd/seed

dev-db: ## Start only Postgres in Docker for local development
	docker run -d --name refresco-pg -p 5432:5432 -e POSTGRES_USER=refresco -e POSTGRES_PASSWORD=refresco -e POSTGRES_DB=refresco postgres:16-alpine

test: test-backend test-frontend ## Run all tests

test-backend: ## Go vet and tests with race detector
	cd backend && go vet ./... && go test -race -count=1 ./...

test-frontend: ## Typecheck the SPA
	cd frontend && npm run typecheck

lint: test-backend test-frontend ## Alias for vet and typecheck

build: build-backend build-frontend ## Build binaries and SPA

build-backend:
	cd backend && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o ../bin/api ./cmd/api

build-frontend:
	cd frontend && npm ci --no-audit --no-fund && npm run build

docker-build: ## Build production images
	VERSION=$(VERSION) docker compose build --build-arg VERSION=$(VERSION)

up: ## Start the production stack (reads .env)
	VERSION=$(VERSION) docker compose up -d

down: ## Stop the stack
	docker compose down

logs: ## Follow API logs
	docker compose logs -f api

clean:
	rm -rf bin frontend/dist
