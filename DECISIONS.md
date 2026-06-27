# Decision Log

## 2026-06-27 — Dashboard UI redesign

**Context:** User said graph nodes, site UI, and text "need to be way better." The initial dashboard had tiny circles (r=5–8) with muted colours and labels only visible on hover.

**Decision:** Complete visual overhaul of all three components and global styles.

**Changes:**

1. **Node sizing** — Increased from r=5–8 to r=22 (files), r=24 (commands), r=20 (thoughts), r=28 (agent). Inner highlight dots added for visual depth.
2. **Colour palette** — Switched from muted to vibrant saturated colours: `#60a5fa` (reads), `#34d399` (writes), `#fbbf24` (commands), `#c084fc` (thoughts), `#22d3ee` (plan steps).
3. **Labels** — Changed from hover-only to always-visible at 60% opacity, full on hover. Label pills use `rgba(7,11,20,0.88)` background with subtle stroke.
4. **Edges** — Added arrow markers (`<marker>` SVG defs per edge kind) for direction. Thickened to 1.8px.
5. **Force simulation** — Tuned parameters: charge -500, collision radius 50, link distance 180, alpha decay 0.014. Creates wider, more readable layout.
6. **Animations** — Node entrance: scale from 0.3→1 with 400ms ease. Agent node: pulsing halo via CSS keyframe animation (`pulse-halo` 3s ease-in-out infinite). Edges: 0.15s stroke transitions.
7. **Status bar** — Compact layout with shorthand labels (rd/wr/cmd), tabular-nums timer, coloured connection dot with glow shadow.
8. **Timeline** — Narrowed to 300px, word-wrap enabled for long payloads, sticky header with event count, compact row spacing.
9. **Global CSS** — Added Inter font from Google Fonts, CSS custom properties (`--bg-primary`, `--color-*`), custom scrollbar, `-webkit-font-smoothing` antialiased.

**Rationale:** Larger, colour-coded nodes with always-visible labels reduce cognitive load when scanning the graph. Arrow markers clarify edge direction (which is non-obvious in a force-directed layout). Better force spread prevents node overlap with more realistic agent workloads.

**Trade-offs:** Labels truncate at 28 chars instead of 22. Longer file paths still visible in timeline. Inter font requires external request, but adds significant readability polish.

## 2026-06-27 — WebSocket reconnection with exponential backoff

**Context:** Previous implementation would hang forever if the server restarted or WS dropped mid-session.

**Decision:** Implement reconnection using exponential backoff (1s–30s) with ±30% jitter, surfaced to user via a "Reconnecting…" indicator in the StatusBar.

**Implementation:** `App.tsx` `useEffect` loop with `setTimeout`. Starts at 1s, doubles on each failure, caps at 30s. Jitter applied as `delay * (0.7 + Math.random() * 0.6)`. StatusBar shows yellow "Reconnecting…" text while `reconnecting` prop is true.

## 2026-06-27 — Embedded dashboard (go:embed) over os.DirFS

**Context:** The dashboard could be served from the filesystem (os.DirFS) or embedded in the binary (//go:embed). DirFS is simpler during development but prevents single-binary distribution.

**Decision:** Use `//go:embed dashboard/dist` to embed the built frontend assets into the Go binary. The dashboard is built via `vite build` before `go build`, and the binary is fully self-contained.

**Rationale:** Single-binary distribution — no separate frontend directory to ship. Verified by running the binary from `/tmp` (no dashboard/dist/) and confirming all assets (HTML, JS, WS) are served.

**Trade-off:** Development requires `make build` (vite + go) instead of just `go run`. Acceptable for the single-binary goal.

## 2026-06-27 — Go module at project root

**Context:** The module was previously in `cli/`, requiring a separate `go.mod` there. This complicated the import path for the embedded dashboard.

**Decision:** Move all Go source files to the project root, delete `cli/` directory. Single `go.mod` at root with module path `agent-live`.

**Rationale:** Simpler build, cleaner project structure, no module nesting confusion.

## 2026-06-27 — CLI flags over config file

**Context:** Needed a way to configure port, origin, and binary version at runtime.

**Decision:** Use Go's `flag` package. Flags: `--port` (default 8080), `--origin` (default "http://localhost:8080"), `--version`, `--help`, `--cmd`, `--timeout`.

**Rationale:** Zero dependencies, standard Go practice, self-documenting via `--help`.

## 2026-06-27 — Agent particle position animation

**Context:** The "agent particle" (small glowing dot) jumps instantly to the latest node when it appears, which looks jarring.

**Decision:** Use a `requestAnimationFrame` loop that lerps the particle position toward its target at 8% per frame. Smooth tracking without a physics engine.

**Constraint:** Initial position (100, 100) before first event — minor cosmetic issue noted.
