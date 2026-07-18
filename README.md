# Agent-live

**Live observability for AI coding agents.**

[![CI](https://github.com/sidmahapatra13/agent-live/actions/workflows/ci.yml/badge.svg)](https://github.com/sidmahapatra13/agent-live/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.5+-3178C6?logo=typescript)](https://www.typescriptlang.org/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

Agent-live wraps a coding-agent CLI in a pseudo-terminal, normalizes its output into structured events, and streams those events to a browser dashboard in real time. It is meant for the moment when an agent is working and you want to see what it is touching, what it is running, and how the session is unfolding.

```bash
agent-live run -- opencode "refactor the auth module"
```

Open [http://localhost:8080](http://localhost:8080) to watch the graph, timeline, counters, and live connection state update as the agent works.

![Agent-live dashboard](dashboard-screenshot.png)

## Why It Stands Out

- **Live knowledge graph**: files, commands, and agent output become connected D3 nodes as the session runs.
- **Timeline you can scan**: every parsed action is shown with timestamps, icons, and expandable details.
- **Agent-agnostic wrapper**: OpenCode JSON output gets the richest events; Claude Code, Codex, and other CLIs still work through text parsing.
- **Single binary**: the React dashboard is embedded into the Go executable.
- **Local by default**: the dashboard binds to `127.0.0.1` unless you opt into another host.
- **Small, readable codebase**: Go handles the PTY, parser, WebSocket hub, and static serving; React owns the visual dashboard.

## Quick Start

### Prerequisites

- Go 1.26+
- Node.js 20+
- A coding agent CLI such as OpenCode, Claude Code, or Codex

### Build

```bash
git clone https://github.com/sidmahapatra13/agent-live.git
cd agent-live
make build
```

### Run

```bash
./agent-live run -- opencode "explain this codebase"
./agent-live run -- claude "write parser tests"
./agent-live run -- codex "review this repo"
```

Then open [http://localhost:8080](http://localhost:8080).

## OpenCode JSON Mode

Agent-live first tries to parse each line as structured OpenCode JSON, then falls back to regex parsing for plain text output.

```bash
./agent-live run -- opencode run --format json "summarize this project"
```

For richer OpenCode sessions, attach to an OpenCode server:

```bash
opencode serve --port 4096

./agent-live run -- opencode run \
  --attach http://localhost:4096 \
  --format json \
  "add tests for the parser"
```

## CLI

```bash
agent-live [flags] run -- <agent-command> [args...]
```

Useful flags:

```bash
-host 127.0.0.1       # HTTP host, defaults to local-only
-port 9090            # HTTP port, defaults to 8080
-origin URL           # restrict WebSocket origin
-history-size 500     # events replayed to newly connected dashboard clients
-max-nodes 500        # graph node cap
-max-edges 1000       # graph edge cap
-exit-when-done       # stop server when wrapped agent exits
-version              # print version
```

## How It Works

```text
agent-live run -- <agent> "prompt"
        |
        v
PTY wrapper captures the agent process
        |
        v
Parser normalizes JSON or text output into events
        |
        v
WebSocket hub broadcasts events and replays recent history
        |
        v
React dashboard renders graph, timeline, and counters
```

Event types:

| Event | Meaning |
| --- | --- |
| `file_read` | Agent read or searched a file |
| `file_write` | Agent wrote, edited, or created a file |
| `command` | Agent ran a shell command |
| `thought` | Agent emitted useful text output |
| `plan_step` | Agent declared a planning step |
| `error` | Parser or runtime error |
| `done` | Wrapped process finished |

## Project Structure

```text
agent-live/
├── main.go              # CLI, PTY runner, HTTP server
├── parser.go            # OpenCode JSON parser + generic text fallback
├── events.go            # Event schema
├── hub.go               # WebSocket broadcast hub with history replay
├── dashboard/           # React + Vite dashboard
│   └── src/
│       ├── App.tsx
│       ├── Graph/
│       ├── Timeline/
│       └── StatusBar/
├── Makefile
└── .github/workflows/ci.yml
```

## Development

```bash
make deps       # install dashboard dependencies
make tscheck    # TypeScript check
make dashboard  # build embedded dashboard assets
make check      # go vet
make build      # full local build
make ci         # local CI sequence
```

For dashboard iteration:

```bash
make deps
cd dashboard && npx vite
```

In another terminal:

```bash
./agent-live run -- opencode "your prompt"
```

## License

MIT
