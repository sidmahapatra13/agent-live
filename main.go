package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
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

//go:embed dashboard/dist/*
var dashboardFS embed.FS

const version = "0.1.0"

func main() {
	// ── CLI flags ────────────────────────────────────────
	port := flag.Int("port", 8080, "HTTP server port")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Agent-live v%s — Watch your AI coding agent in real time.\n\n", version)
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  agent-live [flags] run -- <agent-command> [args...]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  agent-live run -- opencode \"explain this repo\"\n")
		fmt.Fprintf(os.Stderr, "  agent-live --port 9090 run -- claude \"write tests\"\n")
	}
	flag.Parse()

	if *showVersion {
		fmt.Printf("agent-live v%s\n", version)
		return
	}

	args := flag.Args()
	if len(args) < 2 || args[0] != "run" {
		flag.Usage()
		os.Exit(1)
	}

	// agent command is everything after "run" (optional "--" separator)
	cmdArgs := args[1:]
	if len(cmdArgs) > 0 && cmdArgs[0] == "--" {
		cmdArgs = cmdArgs[1:]
	}
	if len(cmdArgs) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	cmdName := cmdArgs[0]
	cmdArgs = cmdArgs[1:]

	// ── Hub ──────────────────────────────────────────────
	hub := NewHub()

	// ── HTTP server ──────────────────────────────────────
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", hub.HandleWS)

	// Static files — serve from embedded dashboard
	distFS, err := fs.Sub(dashboardFS, "dashboard/dist")
	if err != nil {
		log.Fatalf("Failed to load embedded dashboard: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(distFS)))

	addr := fmt.Sprintf(":%d", *port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		log.Printf("Dashboard at http://localhost%s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

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
				// Flush any remaining buffered data as a final event
				if pl := parser.Flush(); pl != nil {
					ts := time.Since(startTime).Seconds()
					evt := Event{
						Type:      EventType(pl.EventType),
						Timestamp: ts,
						Payload:   pl.Payload,
						SessionID: sessionID,
					}
					if data, err := json.Marshal(evt); err == nil {
						hub.Broadcast(data)
					}
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

	// Wait for agent to finish, then send done event and close PTY
	go func() {
		_ = cmd.Wait()
		// Give the PTY read loop time to drain remaining data
		// before closing the PTY (which discards unread buffer)
		time.Sleep(100 * time.Millisecond)
		ptmx.Close()
		ts := time.Since(startTime).Seconds()
		evt := Event{
			Type:      EventDone,
			Timestamp: ts,
			Payload:   "Agent finished",
			SessionID: sessionID,
		}
		data, _ := json.Marshal(evt)
		hub.Broadcast(data)
		safeClose()
	}()

	// Wait for agent to finish, then keep server alive
	<-doneCh
	log.Printf("Agent finished — dashboard stays open at http://localhost%s", addr)
	log.Println("Press Ctrl+C to stop the server.")

	// Keep server alive until Ctrl+C
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	server.Close()
	log.Println("Done.")
}
