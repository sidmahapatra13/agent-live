package main

import (
	"context"
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
	"strconv"
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
	host := flag.String("host", "127.0.0.1", "HTTP server host (0.0.0.0 for all interfaces)")
	origin := flag.String("origin", "", "Allowed WebSocket origin (default: http://<host>:<port>)")
	exitWhenDone := flag.Bool("exit-when-done", false, "Exit server when agent finishes (no keep-alive)")
	historySize := flag.Int("history-size", 500, "Max WebSocket history events to replay")
	maxNodes := flag.Int("max-nodes", 500, "Max graph nodes before pruning")
	maxEdges := flag.Int("max-edges", 1000, "Max graph edges before pruning")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Agent-live v%s — Watch your AI coding agent in real time.\n\n", version)
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  agent-live [flags] run -- <agent-command> [args...]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  agent-live run -- opencode \"explain this repo\"\n")
		fmt.Fprintf(os.Stderr, "  agent-live -host 0.0.0.0 -port 9090 run -- claude \"write tests\"\n")
		fmt.Fprintf(os.Stderr, "  agent-live -exit-when-done run -- opencode \"lint\"\n")
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
	originVal := *origin
	if originVal == "" {
		originVal = fmt.Sprintf("http://%s:%d", *host, *port)
	}
	hub := NewHub(originVal, *historySize)

	// ── HTTP server ──────────────────────────────────────
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.HandleWS)
	mux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{
			"maxNodes": *maxNodes,
			"maxEdges": *maxEdges,
		})
	})

	distFS, err := fs.Sub(dashboardFS, "dashboard/dist")
	if err != nil {
		log.Fatalf("Failed to load embedded dashboard: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(distFS)))

	addr := fmt.Sprintf("%s:%d", *host, *port)
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

	// Wait for agent to finish, capture exit code
	var exitCode int
	done := make(chan struct{})
	go func() {
		if err := cmd.Wait(); err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				exitCode = ee.ExitCode()
			} else {
				exitCode = 1
			}
		}
		time.Sleep(100 * time.Millisecond)
		ptmx.Close()

		ts := time.Since(startTime).Seconds()
		payload := "Agent finished"
		if exitCode != 0 {
			payload = fmt.Sprintf("Agent finished (exit code %d)", exitCode)
		}
		evt := Event{
			Type:      EventDone,
			Timestamp: ts,
			Payload:   payload,
			SessionID: sessionID,
		}
		data, _ := json.Marshal(evt)
		hub.Broadcast(data)
		safeClose()
		close(done)
	}()

	// Wait for agent to finish
	<-doneCh
	log.Printf("Agent finished — exit code %d", exitCode)

	if *exitWhenDone {
		log.Println("Shutting down (--exit-when-done)...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
		os.Exit(exitCode)
	}

	// Keep-alive mode: wait for Ctrl+C
	log.Printf("Dashboard stays open at http://localhost%s", addr)
	log.Println("Press Ctrl+C to stop the server.")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	os.Exit(exitCode)
}

// parseIntParam parses an int query param with a default.
func parseIntParam(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}
