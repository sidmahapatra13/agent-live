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
    │    (embedded)       │
    └────────┬────────────┘
             │ ws:// + http://
             ▼
    ┌─────────────────────┐
    │ Vite + React        │
    │  • D3-force graph   │
    │  • Timeline feed    │
    │  • Status bar       │
    │  • WS reconnect     │
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
- [x] End-to-end: `agent-live run -- opencode "prompt"` captures real events
- [x] README with architecture diagram, quick start, usage guides
- [x] **Embedded frontend** in Go binary via `go:embed` — single-file distribution
- [x] **CLI flags** — `--port`, `--host`, `--origin`, `--help`, `--version`
- [x] **WebSocket reconnection** — exponential backoff (1s–30s) with jitter
- [x] **Go module at project root** — no more `cli/` subdirectory module
- [x] **Dashboard UI redesign** (2026-06-27):
  - Larger graph nodes (r=22 files, r=24 commands, r=20 thoughts, r=28 agent)
  - Vibrant saturated colour palette per event type
  - Always-visible node labels on semi-transparent pill backgrounds
  - Arrow markers on edges showing direction
  - Node entrance scale-in animation on creation
  - Agent pulsing halo animation (CSS keyframes)
  - Inner highlight dot on all nodes for depth
  - Improved force simulation parameters (charge -500, collision 50, distance 180)
  - Compact status bar with read/write/cmd counters
  - Redesigned timeline panel: 300px, word-wrap for long payloads, sticky header, compact spacing
  - Global CSS: Inter font, CSS custom properties, custom scrollbar, deeper dark theme
- [x] **Full CI pipeline**: `make check` (go vet), `make build` (vite + go), `make ci` (tsc + vet + vite + go)

### Fixes applied

- [x] **d3-transition import**: Added `import 'd3-transition'` in GraphCanvas.tsx
- [x] **`__agent__` pseudo-node**: Agent node in simulation's node list
- [x] **Debug red circle removed**: Test marker at line 169 of GraphCanvas.tsx deleted
- [x] **Timer uses server elapsed**: StatusBar now receives `elapsed` prop
- [x] **ANSI stripping**: Added `stripAnsi()` using regex
- [x] **OpenCode regex patterns corrected**: Real OpenCode patterns
- [x] **Regex anchor removed**: No `^` anchor on → patterns
- [x] **Thought threshold lowered**: From `len(line) > 20` to `> 5`
- [x] **PTY buffer flush on EOF**: Final line not lost
- [x] **PTY drain delay**: 100ms sleep before ptmx.Close()
- [x] **Debug logging cleaned up**: All debug statements removed
- [x] **Edge key function fixed**: Handles string refs properly
- [x] **Auto-open removed**: User opens dashboard manually
- [x] **Embedded dashboard**: Single binary distribution
- [x] **WS origin allowlist** — `--origin` flag + automatic localhost/127.0.0.1/[::1] equivalence
- [x] **Parser tests** — 22 tests covering JSON, regex, chunking, ANSI, and WS origin
- [x] **Node/edge cap consistency** — orphaned edges cleaned up when node map is trimmed

## What's Not (MVP Roadmap)

- [ ] Publish to GitHub
- [ ] Claude Code and Codex adapters
- [ ] Session recording and replay
- [ ] Demo GIF / hero screenshot in README
- [ ] Collapsible timeline panel for smaller screens

## Known Issues

- Edge list capped at 1000 entries — graph shows last 800 edges
- Node labels truncated at 28 chars — full path visible in timeline
- Timeline can show duplicate Agent-finished entries under rare race conditions

## Open Questions

- Force simulation parameters for the graph (repulsion, link distance, charge strength) — will tune after seeing real agent data.
- Should the dashboard auto-refresh from a recorded session replay, or only live? (MVP: live only.)
- **Live event streaming reliability:** PTY data arrives in chunks that may split ANSI codes across reads. Current `stripAnsi` + `TrimSpace` handles this, but edge cases with very fast output may still lose events. Consider a more robust line assembly strategy in v0.2.
- **Agent detection:** Parser currently hardcodes OpenCode patterns. Need a way to auto-detect or configure which agent adapter to use (OpenCode vs Claude Code vs Codex).

## Agent-live project conventions

### Dependency installs
- No global installs ever. No `npm install -g`, `pip install` outside venv, `brew install`, `apt-get`, `sudo`.
- Python: `python -m venv .venv` inside project dir, all packages there.
- Node/JS: `npm install` resolves to `node_modules/` inside project dir. Use `npx` for CLI tools.
- Lockfiles required: `package-lock.json`, `requirements.txt` (exact versions).
- System-level dependencies: stop and ask before installing.
