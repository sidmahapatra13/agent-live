GO := /opt/homebrew/bin/go

.PHONY: all build check dev clean deps

# Default target
all: build

# Install Node dependencies
deps:
	cd dashboard && npm install

# Build the Go binary with embedded dashboard
build: deps
	cd dashboard && npx vite build
	$(GO) build -o agent-live .

# Run Go lint/check
check:
	$(GO) vet ./...

# Run TypeScript checks
tscheck:
	cd dashboard && npx tsc --noEmit

# Run in development mode
dev: deps
	@echo "Starting dashboard dev server on :5173..."
	cd dashboard && npx vite &
	@echo "Run agent-live in another terminal:"
	@echo "  ./agent-live run -- opencode \"your prompt\""

# Clean build artifacts
clean:
	rm -rf dashboard/dist
	rm -f agent-live agent-live.exe agent-live-darwin agent-live-linux
	$(GO) clean

# Full CI check
ci: deps tscheck check build
