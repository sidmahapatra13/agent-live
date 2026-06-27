GO := /opt/homebrew/bin/go

.PHONY: all build check dev clean deps dashboard ci tscheck

# Default target
all: build

# Install Node dependencies (run once per clone / after package.json changes)
deps:
	cd dashboard && npm install

# Build the Vite frontend
dashboard:
	cd dashboard && npx vite build

# Build the Go binary (assumes dashboard/dist is up-to-date)
build:
	$(GO) build -o agent-live .

# Run Go lint/check
check:
	$(GO) vet ./...

# Run TypeScript checks
tscheck:
	cd dashboard && npx tsc --noEmit

# Run in development mode
dev:
	@echo "Starting dashboard dev server on :5173..."
	cd dashboard && npx vite &
	@echo "Run agent-live in another terminal:"
	@echo "  ./agent-live run -- opencode \"your prompt\""

# Clean build artifacts
clean:
	rm -rf dashboard/dist
	rm -f agent-live agent-live.exe agent-live-darwin agent-live-linux
	$(GO) clean

# Full CI check (runs all stages in order)
ci: deps tscheck dashboard check build
