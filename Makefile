# Build variables
BINARY_NAME=netronome
BUILD_DIR=bin
DOCKER_IMAGE=netronome

.PHONY: all build clean run docker-build docker-run watch dev

all: build

build: 
	@echo "Building frontend and backend..."
	@mkdir -p $(BUILD_DIR)
	@mkdir -p web/dist
	@cd web && pnpm install && pnpm build
	@touch web/dist/.gitkeep
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/netronome

clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)
	@rm -rf web/dist
	@mkdir -p web/dist
	@touch web/dist/.gitkeep
	@rm -rf web/node_modules

run: build
	@echo "Running application..."
	@./$(BUILD_DIR)/$(BINARY_NAME) serve --config config.toml

docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

docker-run: docker-build
	@echo "Running Docker container..."
	docker run -p 7575:7575 $(DOCKER_IMAGE)

# Development with live reload
dev:
	@echo "Starting development servers..."
	@GIN_MODE=debug tmux new-session -d -s dev 'cd web && pnpm dev'
	@touch web/dist/.gitkeep > /dev/null 2>&1
	@GIN_MODE=debug tmux split-window -h 'make watch'
	@tmux -2 attach-session -d

watch:
	@if command -v air > /dev/null; then \
		GIN_MODE=debug air -- serve --config config.toml; \
		echo "Watching...";\
	else \
		read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
		if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
			go install github.com/cosmtrek/air@latest; \
			GIN_MODE=debug air -- serve --config config.toml; \
			echo "Watching...";\
		else \
			echo "You chose not to install air. Exiting..."; \
			exit 1; \
		fi; \
	fi

