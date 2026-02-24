BIN_DIR := bin
BINARY := sensorpanel
SHELL := /bin/bash

ifneq ($(wildcard ./.env),)
  include .env
  export
else
  env_check = $(shell echo "🟡 WARNING: .env file not found! continue only with exported shell env variables\n\n")
  $(info ${env_check})
endif

.DEFAULT_GOAL := help
.PHONY: help build dev run air-check

##@ Meta
help: ## Show this help with available tasks
	@awk 'BEGIN {FS = ":.*## "}; \
	/^[a-zA-Z0-9_\/-]+:.*## / { printf "  \033[36m%-28s\033[0m %s\n", $$1, $$2 } \
	/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0,5) }' $(MAKEFILE_LIST)

##@ Build
build: ## Build binary into bin/
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY) ./cmd/app

##@ Dev
air-check: ## Verify Air is installed
	@command -v air >/dev/null 2>&1 || { \
		echo "Air is not installed. Install it with:"; \
		echo "  go install github.com/air-verse/air@latest"; \
		exit 1; \
	}

dev: ## Run app with Air (hot reload)
	@$(MAKE) air-check
	air

run: ## Run app once with go run
	go run ./cmd/app
