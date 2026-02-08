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
.PHONY: help build dev

##@ Meta
help: ## Show this help with available tasks
	@awk 'BEGIN {FS = ":.*## "}; \
	/^[a-zA-Z0-9_\/-]+:.*## / { printf "  \033[36m%-28s\033[0m %s\n", $$1, $$2 } \
	/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0,5) }' $(MAKEFILE_LIST)

##@ Build
build: ## Build binary into bin/
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY) .

##@ Dev
dev: ## Run app with go run
	go run .
