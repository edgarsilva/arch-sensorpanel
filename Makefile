BIN_DIR := bin
BINARY := sensorpanel
INSTALL_DIR ?= $(HOME)/.local/bin
SHELL := /bin/bash

.DEFAULT_GOAL := help
.PHONY: help build install dev run air-check client-build client-run client-install client-test

##@ Meta
help: ## Show this help with available tasks
	@awk 'BEGIN {FS = ":.*## "}; \
	/^[a-zA-Z0-9_\/-]+:.*## / { printf "  \033[36m%-28s\033[0m %s\n", $$1, $$2 } \
	/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0,5) }' $(MAKEFILE_LIST)

##@ Build
build: ## Build binary into bin/
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY) ./cmd/app

install: build ## Build and copy binary to ~/.local/bin
	@mkdir -p $(INSTALL_DIR)
	cp $(BIN_DIR)/$(BINARY) $(INSTALL_DIR)/$(BINARY)
	@chmod +x $(INSTALL_DIR)/$(BINARY)
	@echo "Installed $(BINARY) to $(INSTALL_DIR)/$(BINARY)"

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

daemon: install ## Run app as a daemon from system installed
	nohup sensorpanel >/dev/null 2>&1 &

##@ Native client (experimental)
CLIENT_BIN := client/target/release/sensorpanel-client

client-build: ## Build the native Slint + libmpv panel client
	cd client && cargo build --release

client-run: client-build ## Run the native client windowed against the local server
	$(CLIENT_BIN) --windowed

client-test: ## Run the native client's unit tests
	cd client && cargo test

client-install: client-build ## Copy the native client to ~/.local/bin
	install -D -m 0755 $(CLIENT_BIN) $(INSTALL_DIR)/sensorpanel-client
	@echo "Installed sensorpanel-client to $(INSTALL_DIR)/sensorpanel-client"
