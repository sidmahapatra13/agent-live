# Agent-live v0.2 Design

## Goal

Make `agent-live` stand out as a real agent observability tool, not just a live dashboard, by adding local session recording, deterministic replay, clean agent adapter boundaries, derived run insights, and a polished README that reflects the new product shape.

## Scope

This release focuses on source-code improvements that are visible when someone runs, reads, or extends the project:

- Persist normalized agent events to a local replay file.
- Replay saved sessions through the existing dashboard without requiring an agent process.
- Split parsing into adapter-oriented code for OpenCode JSON, Claude/Codex-style text, and generic fallback behavior.
- Derive useful session insights from the event stream.
- Refresh the README after implementation so it sells the actual behavior accurately.

This release does not add hosted sharing, accounts, remote storage, authentication, or cloud sync. Exported replay files are local artifacts that users can share manually if they choose.

## User Experience

### Run And Record

Users can record a live agent run:

```bash
agent-live run --record session.agentlive -- opencode run --format json "refactor auth"
```

During the run, the dashboard behaves as it does today. Each normalized event is also appended to `session.agentlive` as JSON Lines. If the agent exits with a non-zero status, the final `done` event records that exit code.

### Replay

Users can replay a previous run:

```bash
agent-live replay session.agentlive
```

The server opens the same embedded dashboard, streams the recorded events using their original timing by default, and keeps the dashboard available after replay finishes. The dashboard includes replay controls for play/pause, restart, and speed selection.

### Inspect

The dashboard adds a compact insights panel alongside the graph and timeline:

- top files by reads and writes
- command count
- write-without-command/test warning
- failed run indicator when the final event contains a non-zero exit code
- inferred phase counts for planning, exploration, editing, verification, and finished

## Architecture

### Event Model

Create a single normalized event type used by live runs, recording, replay, WebSocket history, frontend state, and tests.

Fields:

- `type`: one of `file_read`, `file_write`, `command`, `thought`, `plan_step`, `error`, `done`
- `timestamp`: seconds since session start
- `payload`: human-readable event payload
- `session_id`: stable session identifier
- `source`: adapter name, such as `opencode-json`, `claude-text`, `codex-text`, or `generic-text`
- `metadata`: optional string map for structured details such as `exit_code`, `tool`, `path`, or `phase`

Keep backward compatibility in the frontend by treating missing `source` and `metadata` as empty values.

### Agent Adapters

Replace the single OpenCode-oriented parser with an adapter registry:

```go
type Adapter interface {
    Name() string
    ParseLine(line string) []ParsedLine
    Flush() []ParsedLine
}
```

Adapters:

- `OpenCodeJSONAdapter`: parses OpenCode `--format json` events.
- `TextAdapter`: parses CLI text output with reusable regex patterns.
- `AutoAdapter`: tries OpenCode JSON first, then text fallback.

Codex and Claude support should start as named text adapter presets using the same implementation, with fixture tests proving expected behavior. This keeps the release useful without overfitting to unstable CLI output formats.

### Session Recording

Create a recorder that accepts normalized `Event` values and appends compact JSON Lines to disk. Recording failure must be explicit: if the file cannot be created, `agent-live` exits before starting the wrapped agent; if a write fails during the run, `agent-live` logs the failure, emits an `error` event, and exits non-zero after the wrapped agent exits.

Recording owns file I/O only. It should not know about PTYs, parsers, WebSockets, or the dashboard.

### Session Replay

Create a replay loader that reads JSON Lines into normalized events and validates required fields. Replayed events flow through the same WebSocket hub as live events. This keeps the frontend simple and verifies that the event stream is the product boundary.

Replay timing:

- Default mode preserves relative event timing.
- `--speed 2` replays twice as fast.
- `--speed 0` broadcasts all events immediately for tests and fast review.

### Insights

Implement insights as a pure frontend derivation from events first. That avoids expanding the Go API and lets replay and live runs share the same behavior.

The frontend should expose an `InsightsPanel` component that consumes `events` and returns deterministic summaries. Put the summary logic in a small pure TypeScript helper so it can be unit-tested later if the project adds a frontend test runner.

### CLI

Extend the existing CLI conservatively:

```text
agent-live [flags] run [run flags] -- <agent-command> [args...]
agent-live [flags] replay [replay flags] <session.agentlive>
```

Run flags:

- `--record <path>`: write replay file during a live run
- `--adapter auto|opencode-json|text|claude|codex`: parsing mode, default `auto`

Replay flags:

- `--speed <float>`: replay speed multiplier, default `1`
- `--exit-when-done`: preserve existing server shutdown behavior

Global flags remain `-host`, `-port`, `-origin`, `-history-size`, `-max-nodes`, `-max-edges`, `-version`.

### README Polish

After implementation, rewrite the README around the stronger product promise:

1. one-sentence pitch: "Live and replayable observability for AI coding agents"
2. quick-start for live mode
3. record/replay example
4. screenshot section
5. supported agents and adapter behavior
6. install/build instructions that use the actual repository path
7. development and CI commands

Fix trust issues while editing docs:

- replace `yourusername` with `sidmahapatra13`
- document privacy: recordings are local JSONL files
- avoid claiming richer per-agent support than fixtures verify

## Error Handling

- If `--record` cannot create the target file, exit before starting the agent.
- If replay input cannot be opened, parsed, or validated, print a clear error and exit non-zero.
- If an adapter sees valid but unsupported JSON, skip it rather than converting it into noisy thought events.
- If the dashboard receives older events without `source` or `metadata`, render them normally.

## Testing

Go tests:

- adapter tests for OpenCode JSON and text fallback
- JSONL recorder round-trip test
- replay loader validation test
- CLI parsing tests where practical without spawning a real agent
- existing parser, hub, and ANSI tests preserved or moved without losing coverage

Manual verification:

- `make ci`
- `agent-live replay --speed 0 testdata/sample.agentlive`
- live smoke test with a simple command such as `agent-live run --record /tmp/sample.agentlive -- sh -c 'echo "→ Read main.go"; echo "→ Bash go test ./..."'`

Frontend verification:

- `make tscheck`
- verify the dashboard renders live events, replay controls, timeline, graph, and insights without layout overlap at desktop width.

## Migration Notes

Existing users can keep using:

```bash
agent-live run -- <agent> "prompt"
```

The default adapter remains automatic, and the dashboard remains embedded in the Go binary. The new features add flags and the `replay` subcommand without removing the current path.
