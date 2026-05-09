package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"

	"github.com/ad201/deephermes/pkg/agent"
)

//go:embed templates/*.html
var embeddedFiles embed.FS

type Server struct {
	agent *agent.Agent
	port  int
	tmpl  *template.Template
}

func NewServer(ag *agent.Agent, port int) (*Server, error) {
	tmpl, err := template.ParseFS(embeddedFiles, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("template parse error: %w", err)
	}
	return &Server{
		agent: ag,
		port:  port,
		tmpl:  tmpl,
	}, nil
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Main page
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.tmpl.ExecuteTemplate(w, "index.html", nil)
	})

	// Chat API (SSE streaming)
	mux.HandleFunc("/api/chat", s.handleChat)

	// Health
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("Web UI starting on http://localhost%s\n", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Use RunStream to get streaming output
	_, err := s.agent.RunStream(r.Context(), req.Message, func(delta string) error {
		fmt.Fprint(w, delta)
		flusher.Flush()
		return nil
	})

	if err != nil {
		fmt.Fprintf(w, "\n[Error: %v]\n", err)
		flusher.Flush()
	}

	// Signal end of stream
	io.WriteString(w, "data: [DONE]\n\n")
	flusher.Flush()
}

