package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("agent-live: ")

	if len(os.Args) < 3 || os.Args[1] != "run" {
		fmt.Fprintf(os.Stderr, "Usage: agent-live run -- <agent-command> [args...]\n")
		fmt.Fprintf(os.Stderr, "Example: agent-live run -- opencode \"explain this repo\"\n")
		os.Exit(1)
	}

	// agent command is everything after "run" (optional "--" separator)
	args := os.Args[2:]
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) == 0 {
		log.Fatal("No agent command specified.")
	}

	cmdName := args[0]
	cmdArgs := args[1:]

	// ── Hub ──────────────────────────────────────────────
	hub := NewHub()

	// ── HTTP server ──────────────────────────────────────
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", hub.HandleWS)

	// Static files — serve the built dashboard
	distDir := "./dashboard/dist"
	mux.Handle("/", http.FileServer(http.Dir(distDir)))

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		log.Printf("Dashboard at http://localhost:8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// ── Open browser ─────────────────────────────────────
	_ = exec.Command("open", "http://localhost:8080").Start()

	// ── Session state ────────────────────────────────────
	sessionID := fmt.Sprintf("%x", time.Now().UnixNano())
	startTime := time.Now()

	// ── Create PTY ───────────────────────────────────────
	cmd := exec.Command(cmdName, cmdArgs...)

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 40, Cols: 160})
	if err != nil {
		log.Fatalf("Failed to start PTY: %v", err)
	}
	defer ptmx.Close()

	log.Printf("Session started — PID %d", cmd.Process.Pid)

	// ── Parser ───────────────────────────────────────────
	parser := newOpenCodeParser()

	// doneCh is closed when the agent process finishes (PTY read loop ends)
	doneCh := make(chan struct{})
	var closeOnce sync.Once
	safeClose := func() { closeOnce.Do(func() { close(doneCh) }) }

	// Read PTY output in a goroutine, parse and broadcast
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("PTY read error: %v", err)
				}
				safeClose()
				return
			}
			if n == 0 {
				continue
			}

			chunk := string(buf[:n])
			lines := parser.Feed(chunk)

			for _, line := range lines {
				ts := time.Since(startTime).Seconds()
				evt := Event{
					Type:      EventType(line.EventType),
					Timestamp: ts,
					Payload:   line.Payload,
					SessionID: sessionID,
				}

				data, err := json.Marshal(evt)
				if err != nil {
					continue
				}
				hub.Broadcast(data)
			}
		}
	}()

	// Wait for agent to finish, then send done event
	go func() {
		_ = cmd.Wait()
		ts := time.Since(startTime).Seconds()
		evt := Event{
			Type:      EventDone,
			Timestamp: ts,
			Payload:   "Agent finished",
			SessionID: sessionID,
		}
		data, _ := json.Marshal(evt)
		hub.Broadcast(data)
		ptmx.Close()
		safeClose()
	}()

	// Wait for agent to finish, then keep server alive
	<-doneCh
	log.Printf("Agent finished — dashboard stays open at http://localhost:8080")
	log.Println("Press Ctrl+C to stop the server.")

	// Keep server alive until Ctrl+C
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	server.Close()
	log.Println("Done.")
}
