package api

import (
	"encoding/json"
	"embed"
	"net/http"
	"strconv"

	"orbital-sentinel/internal/logger"
	"orbital-sentinel/internal/store"
)

//go:embed index.html
var dashboardHTML embed.FS

type StatusProvider interface {
	StatusSnapshot() []map[string]interface{}
}

type Server struct {
	store  *store.Store
	engine StatusProvider
	hub    *Hub
	port   int
	mux    *http.ServeMux
}

func NewServer(st *store.Store, engine StatusProvider, hub *Hub, port int) *Server {
	s := &Server{
		store:  st,
		engine: engine,
		hub:    hub,
		port:   port,
		mux:    http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/", s.handleDashboard)
	s.mux.HandleFunc("/api/status", s.handleStatus)
	s.mux.HandleFunc("/api/history", s.handleHistory)
	s.mux.HandleFunc("/api/stats", s.handleStats)
	s.mux.HandleFunc("/ws", s.hub.HandleWS)
}

func (s *Server) Start() error {
	logger.DashboardStarted(s.port)
	return http.ListenAndServe(":"+strconv.Itoa(s.port), s.mux)
}

func (s *Server) BroadcastStatus() {
	if s.engine != nil {
		s.hub.Broadcast(s.engine.StatusSnapshot())
	}
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	html, err := dashboardHTML.ReadFile("index.html")
	if err != nil {
		http.Error(w, "dashboard not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(html)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if s.engine == nil {
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.engine.StatusSnapshot())
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}
	target := r.URL.Query().Get("target")
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}
	records, err := s.store.GetHistory(target, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}
	stats, err := s.store.GetAllStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
