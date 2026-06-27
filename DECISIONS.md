# DECISIONS.md — Running Decision Log

## 2026-06-27 — Project inception

### Language: Go for CLI/server
- **Alternatives:** TypeScript/Node
- **Why Go:** Single static binary, excellent PTY support (`go-pty` / `creack/pty`), goroutines for concurrent parsing + WebSocket fan-out, no runtime dependency for end users.
- **Trade-off:** Slightly slower iteration than Node if the developer isn't fluent in Go, but the PTY wrapping is the hardest part and Go handles it natively.

### Graph engine: D3-force + SVG (v0.1)
- **Alternatives:** Custom Canvas renderer, Three.js/WebGL
- **Why D3-force:** Confident < 200 active nodes per session in practice. SVG + D3 gives the fastest path to a polished, animated graph without building a physics engine from scratch. Can upgrade to canvas or WebGL in v0.2 if needed.
- **Trade-off:** SVG DOM performance degrades past ~1,000 elements. We'll cap visible nodes at ~500 and fade older nodes to clusters.

### First agent adapter: OpenCode
- **Alternatives:** Claude Code, Codex
- **Why OpenCode:** Installed on this machine, immediate dogfooding. The adapter architecture is designed to be per-agent — OpenCode's output patterns will reveal the parser design assumptions, then Claude Code and Codex adapters are added after the architecture is proven.

### Transport: WebSocket
- **Alternatives:** Server-Sent Events, polling
- **Why WebSocket:** Bidirectional (future: dashboard → agent commands), efficient real-time push, well-supported in Go (gorilla/websocket) and browsers.

### Dashboard framework: Vite + React + TypeScript
- **Alternatives:** Svelte, plain HTML/JS
- **Why Vite+React:** The graph and timeline components benefit from React's component model and Vite's HMR for fast iteration. TypeScript catches layout bugs early.

### Embedded frontend assets
- **Decision:** Go's `embed` package will bundle the built Vite output into the binary for a single-file distribution.
- **Alternatives:** Serve from a separate directory, ship two artifacts.
- **Trade-off:** Slightly larger binary, but single-command install wins for adoption.

### Dependency discipline
- **Decision:** No global installs ever. No `npm install -g`, no `sudo`, no `brew install`. Python packages go in a `.venv` inside the project. Node packages resolve to `node_modules/` inside the project. Lockfiles required.
- **Why:** Reproducible builds, no system pollution, safe for multi-project environments.

## 2026-06-27 — Phase 1 implementation

### PTY wrapper approach
- **Decision:** Use `creack/pty` to spawn agent in a PTY, read output in a goroutine, parser emits events to WebSocket hub. Server shuts down when agent process exits.
- **Alternatives:** Log file tailing, MCP server hook, `os/exec` with pipes (no PTY)
- **Why PTY:** Agents expect a terminal — some write ANSI, check IsTerminal, or behave differently without one. PTY gives us the most faithful capture.
- **Trade-off:** Slightly more complex than `os/exec` pipes, but `creack/pty` is well-maintained.

### Parser design: chunk-based with regex
- **Decision:** Accept raw chunks from PTY, buffer partial lines, regex-match fully-formed lines to event types. Unmatched substantive lines → "thought" events.
- **Alternatives:** Native JSON events from `opencode run --format json`, finite-state machine per agent type
- **Why regex first:** Works across all agents without format-specific code. The JSON format is OpenCode-only and requires us to proxy both the HTTP stream and the agent's stdout.
- **Trade-off:** Regex is brittle to output format changes. We can add JSON events mode as an optimization later.

### Frontend serving: filesystem for dev, `embed` for later
- **Decision:** MVP serves `dashboard/dist/` from the filesystem. Will switch to Go's `embed` when ready for single-binary distribution.
- **Why:** Faster iteration during development. The `embed` change is a few lines once the dashboard is stable.
- **Trade-off:** Requires `dashboard/dist/` to exist at runtime, which means running `vite build` before `agent-live`.

### Go path: Homebrew install at `/opt/homebrew/bin/go`
- **Decision:** Use full path in Makefile (`GO := /opt/homebrew/bin/go`) rather than requiring PATH modification.
- **Why:** The Hermes terminal shell doesn't have Homebrew in PATH by default.
- **Trade-off:** Means Makefile is macOS-specific. Will generalize when targeting other platforms.
