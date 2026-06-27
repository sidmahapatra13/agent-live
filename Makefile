GO := /opt/homebrew/bin/go

.PHONY: all build check dev clean deps

# Default target
all: build

# Install dependencies
deps:
	cd dashboard && npm install

# Build the Go binary
build: deps
	cd dashboard && npx vite build
	cd cli && $(GO) build -o ../agent-live .

# Run Go lint/check
check:
	cd cli && $(GO) vet ./...

# Run in development mode
dev: deps
	@echo "Starting dashboard dev server on :5173..."
	cd dashboard && npx vite &
	@echo "Run agent-live in another terminal:"
	@echo "  ./agent-live run -- opencode \"your prompt\""

# Clean build artifacts
clean:
	rm -rf dashboard/dist
	rm -f agent-live agent-live.exe
	cd cli && $(GO) clean
