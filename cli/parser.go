package main

import (
	"regexp"
	"strings"
)

// ParsedLine is one parsed event from a line of agent output.
type ParsedLine struct {
	EventType string
	Payload   string
}

// openCodeParser parses OpenCode CLI output into structured events.
type openCodeParser struct {
	fileReadRe  *regexp.Regexp
	fileWriteRe *regexp.Regexp
	cmdRunRe    *regexp.Regexp
	planStepRe  *regexp.Regexp
	doneRe      *regexp.Regexp
	buffer      string
}

func newOpenCodeParser() *openCodeParser {
	return &openCodeParser{
		// OpenCode patterns — will tune against real output
		fileReadRe:  regexp.MustCompile(`(?i)reading\s+(file|content):?\s*(.+)`),
		fileWriteRe: regexp.MustCompile(`(?i)(?:writing|creating|saving)\s+(file|to):?\s*(.+)`),
		cmdRunRe:    regexp.MustCompile(`(?i)(?:running|executing|run)\s+(?:command|cmd|shell)?:?\s*(.+)`),
		planStepRe:  regexp.MustCompile(`(?i)(?:step\s+\d+:|##\s+step)\s*(.+)`),
		doneRe:      regexp.MustCompile(`(?i)^(?:done|complete|finished|successfully)`),
	}
}

// Feed processes a chunk of agent output and returns any complete parsed lines.
// Internal buffering handles partial lines across chunks.
func (p *openCodeParser) Feed(chunk string) []ParsedLine {
	p.buffer += chunk

	lines := strings.Split(p.buffer, "\n")
	// Keep the last (possibly partial) line in the buffer
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
		} else {
			// Unmatched lines become "thought" events if they're substantive
			if len(trimmed) > 20 {
				result = append(result, ParsedLine{EventType: "thought", Payload: trimmed})
			}
		}
	}
	return result
}

func (p *openCodeParser) parseLine(line string) *ParsedLine {
	switch {
	case p.fileReadRe.MatchString(line):
		matches := p.fileReadRe.FindStringSubmatch(line)
		if len(matches) >= 3 {
			return &ParsedLine{EventType: "file_read", Payload: strings.TrimSpace(matches[2])}
		}
		return &ParsedLine{EventType: "file_read", Payload: line}

	case p.fileWriteRe.MatchString(line):
		matches := p.fileWriteRe.FindStringSubmatch(line)
		if len(matches) >= 3 {
			return &ParsedLine{EventType: "file_write", Payload: strings.TrimSpace(matches[2])}
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
