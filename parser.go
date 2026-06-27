package main

import (
	"encoding/json"
	"regexp"
	"strings"
)

// ansiEscapeRe matches ANSI escape sequences (SGR codes, etc.)
var ansiEscapeRe = regexp.MustCompile(`\x1b\[[?0-9;]*[a-zA-Z]`)

// stripAnsi removes ANSI escape codes from a string.
func stripAnsi(s string) string {
	return ansiEscapeRe.ReplaceAllString(s, "")
}

// ParsedLine is one parsed event from a line of agent output.
type ParsedLine struct {
	EventType string
	Payload   string
}

// skipLine is a sentinel returned when a line is deliberately ignored.
var skipLine = &ParsedLine{EventType: "__skip__"}

// JSON event structures from OpenCode's --format json output
type openCodeJSONEvent struct {
	Type      string          `json:"type"`
	Timestamp int64           `json:"timestamp"`
	Part      json.RawMessage `json:"part,omitempty"`
}

type openCodePart struct {
	Type  string `json:"type"`
	Tool  string `json:"tool,omitempty"`
	Text  string `json:"text,omitempty"`
	State *struct {
		Status string          `json:"status"`
		Input  json.RawMessage `json:"input,omitempty"`
		Output string          `json:"output,omitempty"`
	} `json:"state,omitempty"`
}

type toolInput struct {
	FilePath string `json:"filePath"`
	Path     string `json:"path"`
	Content  string `json:"content,omitempty"`
	Command  string `json:"command,omitempty"`
}

// openCodeParser parses OpenCode CLI output into structured events.
// Handles both JSON event lines (--format json) and plain text.
type openCodeParser struct {
	// Regex patterns for non-JSON output
	fileReadRe  *regexp.Regexp
	fileWriteRe *regexp.Regexp
	cmdRunRe    *regexp.Regexp
	globRe      *regexp.Regexp
	grepRe      *regexp.Regexp
	planStepRe  *regexp.Regexp
	doneRe      *regexp.Regexp
	buffer      string
}

func newOpenCodeParser() *openCodeParser {
	return &openCodeParser{
		fileReadRe:  regexp.MustCompile(`(?i)→\s+(?:Read|Cat)\s+(.+)`),
		fileWriteRe: regexp.MustCompile(`(?i)→\s+(?:Write|Edit|Create)\s+(.+)`),
		cmdRunRe:    regexp.MustCompile(`(?i)→\s+(?:Bash|Shell|Command|Run)\s+(.+)`),
		globRe:      regexp.MustCompile(`(?i)✱\s+Glob\s+(.+?)\s+\d+\s+match`),
		grepRe:      regexp.MustCompile(`(?i)✱\s+Grep\s+(.+)`),
		planStepRe:  regexp.MustCompile(`(?i)(?:step\s+\d+:|##\s+step)\s*(.+)`),
		doneRe:      nil, // no regex — process-level EventDone from cmd.Wait() is authoritative
	}
}

// Feed processes a chunk of agent output and returns any complete parsed lines.
func (p *openCodeParser) Feed(chunk string) []ParsedLine {
	p.buffer += chunk

	lines := strings.Split(p.buffer, "\n")
	if !strings.HasSuffix(chunk, "\n") && len(lines) > 0 {
		p.buffer = lines[len(lines)-1]
		lines = lines[:len(lines)-1]
	} else {
		p.buffer = ""
	}

	var result []ParsedLine
	for _, line := range lines {
		if pl := p.parseLine(line); pl != nil {
			result = append(result, *pl)
		}
	}
	return result
}

// Flush processes any remaining buffered data as a final line.
// Call this when the PTY closes (EOF) to avoid losing the last line.
func (p *openCodeParser) Flush() *ParsedLine {
	if p.buffer == "" {
		return nil
	}
	trimmed := strings.TrimSpace(stripAnsi(p.buffer))
	p.buffer = ""
	if trimmed == "" {
		return nil
	}
	return p.parseLine(trimmed)
}

func (p *openCodeParser) parseLine(line string) *ParsedLine {
	// Strip ANSI escape codes and trim whitespace
	line = strings.TrimSpace(stripAnsi(line))
	if line == "" {
		return nil
	}

	// First, try JSON — OpenCode outputs JSON events with --format json
	if pl := p.parseJSON(line); pl != nil {
		if pl.EventType == "__skip__" {
			return nil
		}
		return pl
	}

	// Fall back to regex for plain text output
	if pl := p.parseRegex(line); pl != nil {
		return pl
	}

	// Unmatched substantive lines become thought events
	if len(line) > 5 {
		return &ParsedLine{EventType: "thought", Payload: line}
	}
	return nil
}

func (p *openCodeParser) parseJSON(line string) *ParsedLine {
	if !strings.HasPrefix(line, "{") {
		return nil
	}

	var evt openCodeJSONEvent
	if err := json.Unmarshal([]byte(line), &evt); err != nil {
		return nil
	}

	var result *ParsedLine
	switch evt.Type {
	case "tool_use":
		result = p.parseJSONToolUse(evt.Part)
	case "text":
		result = p.parseJSONText(evt.Part)
	case "error":
		result = &ParsedLine{EventType: "error", Payload: line}
	case "step_start", "step_finish":
		result = skipLine
	default:
		result = nil
	}
	// If the line was valid JSON (even if suppressed), don't fall through to regex/thought
	if result == nil {
		result = skipLine
	}
	return result
}

func (p *openCodeParser) parseJSONToolUse(part json.RawMessage) *ParsedLine {
	var pctx openCodePart
	if err := json.Unmarshal(part, &pctx); err != nil {
		return nil
	}

	if pctx.State == nil || pctx.State.Status != "completed" {
		return nil
	}

	var input toolInput
	if pctx.State.Input != nil {
		json.Unmarshal(pctx.State.Input, &input)
	}

	switch pctx.Tool {
	case "read", "search", "grep":
		fp := input.FilePath
		if fp == "" {
			fp = input.Path
		}
		if fp != "" {
			return &ParsedLine{EventType: "file_read", Payload: fp}
		}
	case "edit", "write", "create":
		fp := input.FilePath
		if fp == "" {
			fp = input.Path
		}
		if fp != "" {
			return &ParsedLine{EventType: "file_write", Payload: fp}
		}
	case "command", "run", "execute", "bash":
		cmd := input.Command
		if cmd == "" {
			cmd = pctx.Tool
		}
		return &ParsedLine{EventType: "command", Payload: cmd}
	case "plan":
		return &ParsedLine{EventType: "plan_step", Payload: input.FilePath}
	}

	return nil
}

func (p *openCodeParser) parseJSONText(part json.RawMessage) *ParsedLine {
	var pctx openCodePart
	if err := json.Unmarshal(part, &pctx); err != nil {
		return nil
	}
	if pctx.Text != "" && len(pctx.Text) > 10 {
		return &ParsedLine{EventType: "thought", Payload: pctx.Text}
	}
	return nil
}

func (p *openCodeParser) parseRegex(line string) *ParsedLine {
	switch {
	case p.fileReadRe.MatchString(line):
		matches := p.fileReadRe.FindStringSubmatch(line)
		if len(matches) >= 2 {
			return &ParsedLine{EventType: "file_read", Payload: strings.TrimSpace(matches[1])}
		}
		return &ParsedLine{EventType: "file_read", Payload: line}

	case p.fileWriteRe.MatchString(line):
		matches := p.fileWriteRe.FindStringSubmatch(line)
		if len(matches) >= 2 {
			return &ParsedLine{EventType: "file_write", Payload: strings.TrimSpace(matches[1])}
		}
		return &ParsedLine{EventType: "file_write", Payload: line}

	case p.cmdRunRe.MatchString(line):
		matches := p.cmdRunRe.FindStringSubmatch(line)
		if len(matches) >= 2 {
			return &ParsedLine{EventType: "command", Payload: strings.TrimSpace(matches[1])}
		}
		return &ParsedLine{EventType: "command", Payload: line}

	case p.globRe.MatchString(line):
		matches := p.globRe.FindStringSubmatch(line)
		if len(matches) >= 2 {
			return &ParsedLine{EventType: "file_read", Payload: "glob: " + strings.TrimSpace(matches[1])}
		}

	case p.grepRe.MatchString(line):
		matches := p.grepRe.FindStringSubmatch(line)
		if len(matches) >= 2 {
			return &ParsedLine{EventType: "file_read", Payload: "grep: " + strings.TrimSpace(matches[1])}
		}

	case p.planStepRe.MatchString(line):
		matches := p.planStepRe.FindStringSubmatch(line)
		if len(matches) >= 2 {
			return &ParsedLine{EventType: "plan_step", Payload: strings.TrimSpace(matches[1])}
		}
		return &ParsedLine{EventType: "plan_step", Payload: line}

	case p.doneRe != nil && p.doneRe.MatchString(line):
		return &ParsedLine{EventType: "done", Payload: "Agent finished"}

	default:
		return nil
	}
	return nil
}
