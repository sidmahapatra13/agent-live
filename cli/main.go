package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("agent-live: ")

	if len(os.Args) < 3 || os.Args[1] != "run" {
		log.Fatalf("Usage: agent-live run -- <agent-command> [args...]")
	}

	// agent command is everything after "run --"
	args := os.Args[2:]
	if args[0] == "--" {
		args = args[1:]
	}
	if len(args) == 0 {
		log.Fatal("No agent command specified. Use: agent-live run -- opencode \"prompt\"")
	}

	// TODO Phase 1: PTY wrapper, parser, WebSocket server, dashboard serving
	log.Printf("agent-live starting — wrapping command: %v", args)

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Println("Shutting down...")
}
