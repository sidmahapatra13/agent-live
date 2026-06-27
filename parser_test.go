package main

import (
	"strings"
	"testing"
)

func newTestParser() *openCodeParser {
	return newOpenCodeParser()
}

func TestParserJSONFileRead(t *testing.T) {
	p := newTestParser()
	line := `{"type":"tool_use","part":{"type":"tool_use","tool":"read","state":{"status":"completed","input":{"filePath":"/tmp/main.go"}}}}`
	results := p.Feed(line + "\n")
	if len(results) != 1 {
		t.Fatalf("expected 1 event, got %d", len(results))
	}
	if results[0].EventType != "file_read" {
		t.Errorf("expected file_read, got %s", results[0].EventType)
	}
	if results[0].Payload != "/tmp/main.go" {
		t.Errorf("expected /tmp/main.go, got %s", results[0].Payload)
	}
}

func TestParserJSONFileWrite(t *testing.T) {
	p := newTestParser()
	line := `{"type":"tool_use","part":{"type":"tool_use","tool":"edit","state":{"status":"completed","input":{"filePath":"README.md"}}}}`
	results := p.Feed(line + "\n")
	if len(results) != 1 {
		t.Fatalf("expected 1 event, got %d", len(results))
	}
	if results[0].EventType != "file_write" {
		t.Errorf("expected file_write, got %s", results[0].EventType)
	}
	if results[0].Payload != "README.md" {
		t.Errorf("expected README.md, got %s", results[0].Payload)
	}
}

func TestParserJSONCommand(t *testing.T) {
	p := newTestParser()
	line := `{"type":"tool_use","part":{"type":"tool_use","tool":"bash","state":{"status":"completed","input":{"command":"go test ./..."}}}}`
	results := p.Feed(line + "\n")
	if len(results) != 1 {
		t.Fatalf("expected 1 event, got %d", len(results))
	}
	if results[0].EventType != "command" {
		t.Errorf("expected command, got %s", results[0].EventType)
	}
	if results[0].Payload != "go test ./..." {
		t.Errorf("expected 'go test ./...', got %s", results[0].Payload)
	}
}

func TestParserJSONPathFallback(t *testing.T) {
	p := newTestParser()
	// Some tools send "path" instead of "filePath"
	line := `{"type":"tool_use","part":{"type":"tool_use","tool":"read","state":{"status":"completed","input":{"path":"fallback.go"}}}}`
	results := p.Feed(line + "\n")
	if len(results) != 1 {
		t.Fatalf("expected 1 event, got %d", len(results))
	}
	if results[0].EventType != "file_read" {
		t.Errorf("expected file_read, got %s", results[0].EventType)
	}
	if results[0].Payload != "fallback.go" {
		t.Errorf("expected fallback.go, got %s", results[0].Payload)
	}
}

func TestParserJSONTextThought(t *testing.T) {
	p := newTestParser()
	line := `{"type":"text","part":{"type":"text","text":"We should refactor the auth module to use JWT tokens for better security and scalability."}}`
	results := p.Feed(line + "\n")
	if len(results) != 1 {
		t.Fatalf("expected 1 event, got %d", len(results))
	}
	if results[0].EventType != "thought" {
		t.Errorf("expected thought, got %s", results[0].EventType)
	}
	if !strings.Contains(results[0].Payload, "refactor") {
		t.Errorf("expected thought to contain 'refactor', got %s", results[0].Payload)
	}
}

func TestParserJSONLifecycleSkip(t *testing.T) {
	p := newTestParser()
	line := `{"type":"step_start","part":{}}`
	results := p.Feed(line + "\n")
	if len(results) != 0 {
		t.Errorf("expected 0 events (lifecycle skip), got %d", len(results))
	}
}

func TestParserJSONIncompleteToolIgnored(t *testing.T) {
	p := newTestParser()
	// Tool not yet completed should not emit an event
	line := `{"type":"tool_use","part":{"type":"tool_use","tool":"read","state":{"status":"pending","input":{"filePath":"main.go"}}}}`
	results := p.Feed(line + "\n")
	if len(results) != 0 {
		t.Errorf("expected 0 events (pending status), got %d", len(results))
	}
}

func TestParserRegexFileRead(t *testing.T) {
	p := newTestParser()
	results := p.Feed("→ Read main.go\n")
	if len(results) != 1 {
		t.Fatalf("expected 1 event, got %d", len(results))
	}
	if results[0].EventType != "file_read" {
		t.Errorf("expected file_read, got %s", results[0].EventType)
	}
}

func TestParserRegexFileWrite(t *testing.T) {
	p := newTestParser()
	results := p.Feed("→ Write README.md\n")
	if len(results) != 1 {
		t.Fatalf("expected 1 event, got %d", len(results))
	}
	if results[0].EventType != "file_write" {
		t.Errorf("expected file_write, got %s", results[0].EventType)
	}
}

func TestParserRegexCommand(t *testing.T) {
	p := newTestParser()
	results := p.Feed("→ Bash go test ./...\n")
	if len(results) != 1 {
		t.Fatalf("expected 1 event, got %d", len(results))
	}
	if results[0].EventType != "command" {
		t.Errorf("expected command, got %s", results[0].EventType)
	}
}

func TestParserDoneRegexIsNil(t *testing.T) {
	p := newTestParser()
	// The doneRe is nil (not compiled), so "done" should fall through to thought
	results := p.Feed("done\n")
	// "done" is only 4 chars, shorter than the thought threshold of 5
	if len(results) != 0 {
		// If length > 5 it would be a thought. Either way, never EventType "done"
		for _, r := range results {
			if r.EventType == "done" {
				t.Fatal("doneRe is nil — parser should never emit done from regex")
			}
		}
	}
}

func TestParserThoughtFallback(t *testing.T) {
	p := newTestParser()
	results := p.Feed("This is a substantive thought about the codebase structure.\n")
	if len(results) != 1 {
		t.Fatalf("expected 1 event, got %d", len(results))
	}
	if results[0].EventType != "thought" {
		t.Errorf("expected thought, got %s", results[0].EventType)
	}
}

func TestParserShortLineIgnored(t *testing.T) {
	p := newTestParser()
	results := p.Feed("ok\n")
	if len(results) != 0 {
		t.Errorf("expected 0 events for short line, got %d", len(results))
	}
}

func TestParserEmptyLineIgnored(t *testing.T) {
	p := newTestParser()
	results := p.Feed("\n")
	if len(results) != 0 {
		t.Errorf("expected 0 events for empty line, got %d", len(results))
	}
}

func TestParserANSIRemoved(t *testing.T) {
	p := newTestParser()
	// ANSI codes should be stripped before parsing
	results := p.Feed("\x1b[32m→ Read main.go\x1b[0m\n")
	if len(results) != 1 {
		t.Fatalf("expected 1 event, got %d", len(results))
	}
	if results[0].EventType != "file_read" {
		t.Errorf("expected file_read, got %s", results[0].EventType)
	}
}

func TestParserChunkedInput(t *testing.T) {
	p := newTestParser()
	// Feed partial line (no newline) — should buffer
	results := p.Feed("→ Read main")
	if len(results) != 0 {
		t.Fatalf("expected 0 events from partial line, got %d", len(results))
	}
	// Complete the line
	results = p.Feed(".go\n")
	if len(results) != 1 {
		t.Fatalf("expected 1 event after completing line, got %d", len(results))
	}
	if results[0].EventType != "file_read" {
		t.Errorf("expected file_read, got %s", results[0].EventType)
	}
}

func TestParserFlush(t *testing.T) {
	p := newTestParser()
	// Feed data without trailing newline
	_ = p.Feed("→ Write README.md")
	pl := p.Flush()
	if pl == nil {
		t.Fatal("expected flush to return event, got nil")
	}
	if pl.EventType != "file_write" {
		t.Errorf("expected file_write from flush, got %s", pl.EventType)
	}
}

func TestParserMultipleEventsPerChunk(t *testing.T) {
	p := newTestParser()
	results := p.Feed("→ Read main.go\n→ Bash go test\n→ Write README.md\n")
	if len(results) != 3 {
		t.Fatalf("expected 3 events, got %d", len(results))
	}
	if results[0].EventType != "file_read" {
		t.Errorf("first event: expected file_read, got %s", results[0].EventType)
	}
	if results[1].EventType != "command" {
		t.Errorf("second event: expected command, got %s", results[1].EventType)
	}
	if results[2].EventType != "file_write" {
		t.Errorf("third event: expected file_write, got %s", results[2].EventType)
	}
}

func TestStripAnsi(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"\x1b[32mhello\x1b[0m", "hello"},
		{"\x1b[1;34mcolor\x1b[0m", "color"},
		{"no escape", "no escape"},
		{"\x1b[?25lhide cursor", "hide cursor"},
	}
	for _, tt := range tests {
		got := stripAnsi(tt.input)
		if got != tt.want {
			t.Errorf("stripAnsi(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
