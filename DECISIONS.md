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
