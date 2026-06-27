# Agent-live

> A live dashboard that visualises your AI coding agent's brain in real-time.

---

## Goal

When you run an AI coding agent (OpenCode, Claude Code, Codex), `agent-live` wraps it in a PTY, parses the output stream into structured events, and opens a browser dashboard showing:

- **Live knowledge graph** — files the agent reads/writes as nodes with animated edges
- **Timeline feed** — scrollable event log with icons and timestamps
- **Status bar** — current action, elapsed time, files touched

The purpose: give developers a mesmerising, shareable window into what their agent is *actually doing*, turning the opaque black box of agent output into a visual narrative.

---

## Current Architecture (v0.1)

```
agent-live run -- <agent-command>
         │
         ▼
    ┌─────────────────────┐
    │ Go CLI              │
    │  • PTY wrapper      │
    │  • Output parser    │
    │  • WebSocket server │
    │  • Serves dashboard │
    └────────┬────────────┘
             │ ws:// + http://
             ▼
    ┌─────────────────────┐
    │ Vite + React        │
    │  • D3-force graph   │
    │  • Timeline feed    │
    │  • Status bar       │
    └─────────────────────┘
```

### Tech stack

| Layer | Choice | Why |
|-------|--------|-----|
| CLI / server | **Go** | Single static binary, excellent PTY support (`go-pty`), goroutines for concurrent parsing + WS broadcast |
| Frontend | **TypeScript + React + Vite** | Fast dev iteration, familiar ecosystem |
| Graph engine | **D3-force + SVG** | Confident < 200 active nodes in practice; can upgrade to canvas in v0.2 |
| Transport | **WebSocket** | Real-time event push from Go server to browser |
| Agent (first) | **OpenCode** | Installed on this machine, immediate dogfooding |

### Event types

| Event | Trigger | Visual |
|-------|---------|--------|
| `file_read` | Agent reads a file | Blue glow edge |
| `file_write` | Agent writes a file | Green pulse edge |
| `command` | Agent runs a shell command | Yellow action node |
| `thought` | Agent outputs reasoning (non-command lines) | Purple thought node |
| `plan_step` | Agent declares a plan step | Milestone marker |

---

## What's Done

- [x] Three architectural decisions locked (Go, D3-force, OpenCode-first)
- [x] Project scaffolded

## What's Not (MVP Roadmap)

Phase 1 — Go CLI skeleton
- [ ] `main.go` — PTY wrapper, event loop, WebSocket server
- [ ] `parser/opencode.go` — regex-based output parser for OpenCode
- [ ] `events.go` — event type definitions
- [ ] `server/hub.go` — WebSocket hub (connect/broadcast/disconnect)

Phase 2 — Dashboard MVP
- [ ] Vite + React + TypeScript scaffold
- [ ] WebSocket hook (`useWebSocket.ts`)
- [ ] StatusBar component
- [ ] Timeline component
- [ ] Graph component (D3-force, nodes, edges, agent particle)

Phase 3 — Integration
- [ ] `embed` frontend assets into Go binary
- [ ] Auto-open browser on `agent-live run`
- [ ] OpenCode adapter tuned from real output
- [ ] README with demo GIF

Phase 4 — Additional agents
- [ ] Claude Code adapter
- [ ] Codex adapter

## Open Questions

- Force simulation parameters for the graph (repulsion, link distance, charge strength) — will tune after seeing real agent data.
- Should the dashboard auto-refresh from a recorded session replay, or only live? (MVP: live only.)

## Agent-live project conventions

### Dependency installs
- No global installs ever. No `npm install -g`, `pip install` outside venv, `brew install`, `apt-get`, `sudo`.
- Python: `python -m venv .venv` inside project dir, all packages there.
- Node/JS: `npm install` resolves to `node_modules/` inside project dir. Use `npx` for CLI tools.
- Lockfiles required: `package-lock.json`, `requirements.txt` (exact versions).
- System-level dependencies: stop and ask before installing.
