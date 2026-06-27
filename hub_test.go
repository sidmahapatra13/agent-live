package main

import (
	"net/http"
	"testing"
)

func makeOriginRequest(origin string) *http.Request {
	r, _ := http.NewRequest("GET", "/ws", nil)
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	return r
}

func TestCheckOriginLocalhostVariants(t *testing.T) {
	// Allowed origin = http://127.0.0.1:8080
	hub := NewHub("http://127.0.0.1:8080", 500)
	check := hub.upgrader.CheckOrigin

	tests := []struct {
		name   string
		origin string
		allow  bool
	}{
		{"same origin", "http://127.0.0.1:8080", true},
		{"localhost", "http://localhost:8080", true},
		{"localhost no port", "http://localhost", true},
		{"ipv6 localhost", "http://[::1]:8080", true},
		{"ipv6 no port", "http://[::1]", true},
		{"no origin (non-browser)", "", true},
		{"external domain", "https://evil.com", false},
		{"external ip", "http://10.0.0.1:8080", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := makeOriginRequest(tt.origin)
			got := check(r)
			if got != tt.allow {
				t.Errorf("CheckOrigin(origin=%q) = %v, want %v", tt.origin, got, tt.allow)
			}
		})
	}
}

func TestCheckOriginCustomOrigin(t *testing.T) {
	hub := NewHub("https://myapp.example.com", 500)
	check := hub.upgrader.CheckOrigin

	tests := []struct {
		name   string
		origin string
		allow  bool
	}{
		{"exact match", "https://myapp.example.com", true},
		{"localhost still allowed", "http://localhost", true},
		{"different domain", "https://other.com", false},
		{"no origin", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := makeOriginRequest(tt.origin)
			got := check(r)
			if got != tt.allow {
				t.Errorf("CheckOrigin(origin=%q) = %v, want %v", tt.origin, got, tt.allow)
			}
		})
	}
}

func TestCheckOriginDevMode(t *testing.T) {
	// Empty allowedOrigin = dev mode, all origins accepted
	hub := NewHub("", 500)
	check := hub.upgrader.CheckOrigin

	if !check(makeOriginRequest("https://anything.com")) {
		t.Error("dev mode should allow any origin")
	}
	if !check(makeOriginRequest("")) {
		t.Error("dev mode should allow missing origin")
	}
}

func TestStripScheme(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"http://localhost:8080", "localhost"},
		{"https://127.0.0.1:9999", "127.0.0.1"},
		{"http://[::1]:8080", "[::1]"},
		{"ws://example.com:80", "example.com"},
		{"no-scheme:8080", "no-scheme"},
	}
	for _, tt := range tests {
		got := stripScheme(tt.input)
		if got != tt.want {
			t.Errorf("stripScheme(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
