package parser

import (
	"bufio"
	"io"
	"regexp"
)

// EventCallback is called for every parsed event.
type EventCallback func(eventType string, payload string)

// OpenCodeParser parses OpenCode CLI output into structured events.
type OpenCodeParser struct {
	fileReadRe  *regexp.Regexp
	fileWriteRe *regexp.Regexp
	cmdRunRe    *regexp.Regexp
	thoughtRe   *regexp.Regexp
	planStepRe  *regexp.Regexp
	doneRe      *regexp.Regexp
}

// NewOpenCode creates a new parser for OpenCode output.
func NewOpenCode() *OpenCodeParser {
	// TODO Phase 1: tune regex patterns against real OpenCode output
	return &OpenCodeParser{
		fileReadRe:  regexp.MustCompile(`Reading file: (.+)`),
		fileWriteRe: regexp.MustCompile(`Writing file: (.+)`),
		cmdRunRe:    regexp.MustCompile(`Running command: (.+)`),
		thoughtRe:   regexp.MustCompile(`(?:Thinking|Reasoning): (.+)`),
		planStepRe:  regexp.MustCompile(`Step \d+: (.+)`),
		doneRe:      regexp.MustCompile(`(?:Done|Complete|Finished)`),
	}
}

// Parse reads lines from r and emits events via callback.
func (p *OpenCodeParser) Parse(r io.Reader, cb EventCallback) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		p.parseLine(line, cb)
	}
}

func (p *OpenCodeParser) parseLine(line string, cb EventCallback) {
	// TODO Phase 1: implement pattern matching against real OpenCode output
	switch {
	case p.fileReadRe.MatchString(line):
		matches := p.fileReadRe.FindStringSubmatch(line)
		cb("file_read", matches[1])
	case p.fileWriteRe.MatchString(line):
		matches := p.fileWriteRe.FindStringSubmatch(line)
		cb("file_write", matches[1])
	case p.cmdRunRe.MatchString(line):
		matches := p.cmdRunRe.FindStringSubmatch(line)
		cb("command", matches[1])
	case p.thoughtRe.MatchString(line):
		matches := p.thoughtRe.FindStringSubmatch(line)
		cb("thought", matches[1])
	case p.planStepRe.MatchString(line):
		matches := p.planStepRe.FindStringSubmatch(line)
		cb("plan_step", matches[1])
	case p.doneRe.MatchString(line):
		cb("done", "")
	}
}
