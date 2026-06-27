.PHONY: all dev build clean

# Default target
all: build

# Install dependencies
.PHONY: deps
deps:
	cd dashboard && npm install

# Run in development mode (dashboard dev server + Go CLI)
.PHONY: dev
dev: deps
	@echo "Starting dashboard dev server on :5173..."
	cd dashboard && npx vite &
	@echo "Run agent-live in another terminal:"
	@echo "  cd cli && go run . run -- opencode \"your prompt\""

# Build the Go binary (embeds built frontend)
.PHONY: build
build: deps
	cd dashboard && npx vite build
	cd cli && go build -o ../agent-live .

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf dashboard/dist
	rm -f agent-live agent-live.exe
	cd cli && go clean

# Run Go lint/check
.PHONY: check
check:
	cd cli && go vet ./...
