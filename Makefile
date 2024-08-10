SHELL := /bin/bash
include .env
export
export APP_NAME := $(basename $(notdir $(shell pwd)))

.PHONY: help
help: ## display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: tool
tool: ## install the tools
	@go install github.com/sqldef/sqldef/cmd/psqldef@latest
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

.PHONY: local
local: ## run the server locally
	@docker compose --project-name ${APP_NAME} --file ./.docker/compose.yaml up --detach

.PHONY: psql
psql:
	@docker exec -it ${APP_NAME}-postgres psql -U postgres

.PHONY: down
down: ## stop the server
	@docker compose --project-name ${APP_NAME} down --volumes
	@docker rmi ${APP_NAME}-api

.PHONY: deploy
deploy: ## deploy to vercel
	@vercel --prod

.PHONY: migrate
migrate: ## run the migrations
	@psqldef --user=${POSTGRES_USER} --password=${POSTGRES_PASSWORD} --host=${POSTGRES_HOST} --port=5432 ${POSTGRES_DATABASE} < schema/schema.sql

.PHONY: destroy
destroy: ## destroy the vercel deployment
	@vercel project rm ${APP_NAME}

.PHONY: gen
gen: ## generate code.
	@sqlc generate
	@go mod tidy

.PHONY: mod
mod: ## go mod tidy & go mod vendor
	@go get -u -t ./...
	@go mod tidy

.PHONY: test
test: ## run the tests
	@go test -v ./...
