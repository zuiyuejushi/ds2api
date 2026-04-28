package poolserver

import (
	"encoding/json"
	"net/http"
	"os"

	"ds2api/internal/account"
	"ds2api/internal/config"
)

// Server exposes account.Pool operations over HTTP.
type Server struct {
	pool  *account.Pool
	store *config.Store
}

// New creates a new Server.
func New(pool *account.Pool, store *config.Store) *Server {
	return &Server{pool: pool, store: store}
}

// RegisterRoutes registers all pool server routes on the given mux.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/pool/acquire", s.handleAcquire)
	mux.HandleFunc("POST /api/pool/release", s.handleRelease)
	mux.HandleFunc("POST /api/pool/reset", s.handleReset)
	mux.HandleFunc("GET /api/pool/stats", s.handleStats)
	mux.HandleFunc("GET /api/pool/config", s.handleConfig)
}

// auth checks the Authorization header against DS2API_POOL_SERVER_KEY.
// If the env var is empty, auth is bypassed. Returns true if authorized.
func (s *Server) auth(r *http.Request) bool {
	expected := os.Getenv("DS2API_POOL_SERVER_KEY")
	if expected == "" {
		return true
	}
	return r.Header.Get("Authorization") == expected
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) handleAcquire(w http.ResponseWriter, r *http.Request) {
	if !s.auth(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Target  string          `json:"target"`
		Exclude []string        `json:"exclude"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	exclude := make(map[string]bool, len(req.Exclude))
	for _, id := range req.Exclude {
		exclude[id] = true
	}

	acc, err := s.pool.AcquireWaitPool(r.Context(), req.Target, exclude)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"accountID": acc.ID,
		"token":     acc.Token,
	})
}

func (s *Server) handleRelease(w http.ResponseWriter, r *http.Request) {
	if !s.auth(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		AccountID string `json:"accountID"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	s.pool.Release(req.AccountID)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	if !s.auth(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	s.pool.Reset()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if !s.auth(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	writeJSON(w, http.StatusOK, s.pool.Status())
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if !s.auth(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	writeJSON(w, http.StatusOK, map[string]config.Config{
		"config": s.store.Snapshot(),
	})
}
