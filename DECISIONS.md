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

### Parser: dual-mode JSON + regex
- **Decision:** Parser first tries JSON parse on each line (for OpenCode's `--format json` output), then falls back to regex patterns.
- **Alternatives:** JSON-only, regex-only, separate parser instances
- **Why:** JSON gives structured events with exact file paths; regex works with any agent. Both can coexist on the same output stream.
- **Trade-off:** Slightly more CPU per line (attempting JSON parse), negligible in practice.

### Dashboard design: live status, hover polish, grid background
- **Decision:** StatusBar uses its own live timer (500ms interval) triggered by `status === 'running'`. Graph edges have `hover: brighter + thicker` CSS transitions. Nodes have tooltips showing full label. Subtle dot grid background.
- **Alternatives:** Rely on server-sent elapsed timestamps; no hover effects; solid background
- **Why:** Live timers feel responsive. Hover effects add polish with zero complexity (pure CSS). Grid gives depth without clutter.
- **Trade-off:** Timer reset logic needs `startRef` management — works but slightly more code.

## 2026-06-27 — Parser and streaming fixes

### D3 v3 modular requires explicit imports
- **Decision:** Added `import 'd3-transition'` in GraphCanvas.tsx to register `.transition()` on D3 selections.
- **Why:** D3 v3 modular packages (`d3-transition`, `d3-force`, `d3-selection`) are separate npm packages. Installing them isn't enough — the import registers the plugin on the global selection prototype. Without it, `.transition()` throws `B.transition is not a function`.
- **Trade-off:** Adds ~10KB to bundle (191KB → 201KB). Acceptable for the animation capability gained.

### __agent__ pseudo-node in simulation
- **Decision:** Added a fixed-position `__agent__` node to the D3 simulation's node list.
- **Why:** Edges reference `__agent__` as source/target, but the link force requires all referenced nodes to exist in the node list. Without it, D3 throws "node not found" error.
- **Trade-off:** The agent node is static (fixed x,y). Could be made dynamic in v0.2.

### ANSI stripping before regex parsing
- **Decision:** Added `stripAnsi()` function using regex `\x1b\[[0-9;]*[a-zA-Z]` to remove ANSI escape codes from PTY output before regex matching.
- **Why:** OpenCode's PTY output contains ANSI escape codes (color reset `ESC[0m`, cursor movement, etc.) interspersed with visible text. These codes break regex patterns — e.g., `^→ Read` won't match `ESC[0m→ ESC[0mRead` even though the visible text is `→ Read`.
- **Trade-off:** Some exotic ANSI sequences (OSC, DCS) may not be stripped. Acceptable for agent output which mostly uses SGR codes.

### OpenCode output patterns (empirically discovered)
- **Decision:** Parser regex patterns match actual OpenCode output: `→ Read <path>`, `→ Write <path>`, `→ Edit <path>`, `✱ Glob <pattern>`, `✱ Grep <query>`.
- **Alternatives:** Original assumed generic patterns like `Read: <path>`, `Write: <path>`, `Command: <cmd>`
- **Why:** Discovered by running `opencode run "prompt" 2>&1 | cat -v | xxd` to inspect raw bytes. The `→` (U+2192) and `✱` (U+2731) are literal Unicode characters in OpenCode's output, not ASCII approximations.
- **Trade-off:** Patterns are OpenCode-specific. Other agents (Claude Code, Codex) will need their own patterns.

### Regex anchor removal
- **Decision:** Removed `^` anchor from `→ Read`/`→ Write`/etc. regex patterns.
- **Why:** After ANSI stripping and `strings.TrimSpace`, the line may still have leading whitespace from the PTY (e.g., indentation). The `^` anchor requires the arrow at position 0, causing misses.
- **Trade-off:** Slightly less precise matching (could match `→ Read` in the middle of a line). Acceptable since `→ Read` is unlikely to appear in agent output except as a tool invocation.

### PTY buffer flush on EOF
- **Decision:** Added `parser.Flush()` method called when the PTY read loop hits EOF. Processes any remaining buffered data as a final event.
- **Why:** The PTY read loop buffers incomplete lines (no trailing `\n`). When the agent finishes and the PTY closes, the last chunk often lacks a trailing newline, so it stays in the buffer forever. `Flush()` ensures it's processed.
- **Trade-off:** The flushed line may be a partial/truncated line if the agent was mid-output. Acceptable — better to show a partial event than lose it entirely.

### PTY drain delay before close
- **Decision:** Added 100ms `time.Sleep` before `ptmx.Close()` in the `cmd.Wait()` goroutine.
- **Why:** `cmd.Wait()` returns when the agent process exits, but the PTY read loop may not have drained all available data yet. Closing the PTY immediately discards unread buffer. The 100ms delay gives the read loop time to process remaining chunks.
- **Trade-off:** Adds 100ms to shutdown time. Could be replaced with a proper synchronization mechanism (e.g., read loop signals completion via channel) in v0.2.

## 2026-06-27 — Shipping readiness

### Embedded frontend via go:embed
- **Decision:** Moved Go module from `cli/` to project root and use `//go:embed dashboard/dist/*` to bundle the built frontend into the binary.
- **Alternatives:** Serve from filesystem (old approach), ship two artifacts (binary + dashboard)
- **Why go:embed:** Single-binary distribution was the stated goal from day 1. Embedding eliminates the relative-path problem — the binary works regardless of CWD. The 200KB increase in binary size (9.1MB from 8.9MB) is negligible.
- **Trade-off:** Requires `vite build` to run before `go build`. The Makefile handles this ordering. Minor build-time friction, zero runtime cost.

### CLI flags: --port, --help, --version
- **Decision:** Use Go's `flag` package for `--port` (default 8080), `--help`, and `--version` flags.
- **Alternatives:** `pflag`/`cobra` for richer CLI
- **Why flags:** The tool has exactly 3 flags. `flag` is stdlib, zero dependencies, sufficient for MVP. Can upgrade to `cobra` if flags grow.
- **Trade-off:** No subcommand-level help. `flag.Usage()` prints all flags regardless of subcommand. Acceptable for now.

### WebSocket reconnection: exponential backoff with jitter
- **Decision:** Dashboard WebSocket client attempts reconnection on close/error with exponential backoff: 1s base, 2x multiplier, 30s cap, ±30% jitter. Shows "Reconnecting…" indicator in status bar.
- **Alternatives:** Linear polling, no reconnection (previous behaviour)
- **Why exponential backoff:** Standard for network reliability. Jitter prevents thundering herd if multiple browser tabs reconnect simultaneously. The 30s cap ensures it doesn't wait forever.
- **Trade-off:** If the server goes down permanently, the browser keeps retrying forever. Acceptable — user can close the tab.

### Go module at project root
- **Decision:** Moved `go.mod` and all `.go` files from `cli/` to the project root.
- **Alternatives:** Keep `cli/` module, use `//go:embed ../dashboard/dist/*`
- **Why root:** Cleaner embed paths (`dashboard/dist/*` instead of `../dashboard/dist/*`). The `cli/` directory was a leftover from initial scaffolding when the module was nested. Single-binary tools conventionally have their module at the root.
- **Trade-off:** Changes `go build` target from `./cli` to `.`. Makefile and README updated accordingly.

### StatusBar uses server timestamps
- **Decision:** StatusBar receives `elapsed` prop from App.tsx (derived from the last event's server timestamp) instead of maintaining its own timer.
- **Why:** The original internal timer (500ms interval) never started properly because it depended on a state transition that didn't fire. Server timestamps are authoritative and always work.
- **Sub-second display:** Shows `<1s` when elapsed < 1s, otherwise `M:SS` format.
- **Trade-off:** Timer updates are now tied to event arrival rate rather than a steady clock. For display purposes this is fine — the timer only matters for the first few seconds of a session.
