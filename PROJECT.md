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
- [x] Project scaffolded: Go module, React+Vite dashboard
- [x] Go CLI: PTY wrapper, event types, dual-mode parser (JSON + regex), WebSocket hub
- [x] Parser handles both OpenCode JSON events (`--format json`) and plain text regex fallback
- [x] Dashboard: StatusBar (live timer), Timeline (event feed with icons), GraphCanvas (D3-force simulation)
- [x] Graph: edge hover highlighting, node tooltips, dot grid background, smooth enter/update transitions
- [x] Dashboard build verified: `tsc --noEmit` clean, `vite build` produces dist/
- [x] Go binary: builds, vets clean, starts HTTP server on :8080 serving dashboard
- [x] End-to-end: `agent-live run -- opencode run --format json "prompt"` captures real events
- [x] README with architecture diagram, quick start, usage guides

## What's Not (MVP Roadmap)

- [ ] Dashboard polish: edge highlighting, smoother transitions, responsive layout
- [ ] README with demo GIF and setup instructions
- [ ] Publish to GitHub

## Known Issues

- Agent particle starts at (100, 100) before first event positions it — minor cosmetic
- Edge list capped at 1000 entries — graph shows last 800 edges
- Node labels truncated at 22 chars — full path visible in timeline

## Open Questions

- Force simulation parameters for the graph (repulsion, link distance, charge strength) — will tune after seeing real agent data.
- Should the dashboard auto-refresh from a recorded session replay, or only live? (MVP: live only.)
- OpenCode JSON events format (`opencode run --format json`) outputs structured events — should the parser prefer this over regex? Its structured data would be more reliable.

## Agent-live project conventions

### Dependency installs
- No global installs ever. No `npm install -g`, `pip install` outside venv, `brew install`, `apt-get`, `sudo`.
- Python: `python -m venv .venv` inside project dir, all packages there.
- Node/JS: `npm install` resolves to `node_modules/` inside project dir. Use `npx` for CLI tools.
- Lockfiles required: `package-lock.json`, `requirements.txt` (exact versions).
- System-level dependencies: stop and ask before installing.
