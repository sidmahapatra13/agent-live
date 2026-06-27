package main

import (
	"encoding/json"
	"regexp"
	"strings"
)

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
	planStepRe  *regexp.Regexp
	doneRe      *regexp.Regexp
	buffer      string
}

func newOpenCodeParser() *openCodeParser {
	return &openCodeParser{
		fileReadRe:  regexp.MustCompile(`(?i)reading\s+(?:file|content):?\s*(.+)`),
		fileWriteRe: regexp.MustCompile(`(?i)(?:writing|creating|saving|editing)\s+(?:file|to)?:?\s*(.+)`),
		cmdRunRe:    regexp.MustCompile(`(?i)(?:running|executing|run)\s+(?:command|cmd|shell)?:?\s*(.+)`),
		planStepRe:  regexp.MustCompile(`(?i)(?:step\s+\d+:|##\s+step)\s*(.+)`),
		doneRe:      regexp.MustCompile(`(?i)^(?:done|complete|finished|successfully)`),
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
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if pl := p.parseLine(trimmed); pl != nil {
			result = append(result, *pl)
		}
	}
	return result
}

func (p *openCodeParser) parseLine(line string) *ParsedLine {
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
	if len(line) > 20 {
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

	switch evt.Type {
	case "tool_use":
		return p.parseJSONToolUse(evt.Part)
	case "text":
		return p.parseJSONText(evt.Part)
	case "error":
		return &ParsedLine{EventType: "error", Payload: line}
	case "step_start", "step_finish":
		// lifecycle events, skip
		return skipLine
	default:
		return nil
	}
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
	case "read":
		if input.FilePath != "" {
			return &ParsedLine{EventType: "file_read", Payload: input.FilePath}
		}
	case "edit", "write", "create":
		if input.FilePath != "" {
			return &ParsedLine{EventType: "file_write", Payload: input.FilePath}
		}
	case "command", "run", "execute":
		cmd := input.Command
		if cmd == "" {
			cmd = pctx.Tool
		}
		return &ParsedLine{EventType: "command", Payload: cmd}
	case "search", "grep":
		return &ParsedLine{EventType: "file_read", Payload: input.FilePath}
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

	case p.planStepRe.MatchString(line):
		matches := p.planStepRe.FindStringSubmatch(line)
		if len(matches) >= 2 {
			return &ParsedLine{EventType: "plan_step", Payload: strings.TrimSpace(matches[1])}
		}
		return &ParsedLine{EventType: "plan_step", Payload: line}

	case p.doneRe.MatchString(line):
		return &ParsedLine{EventType: "done", Payload: "Agent finished"}

	default:
		return nil
	}
}
