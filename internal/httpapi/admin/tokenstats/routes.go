package tokenstats

import "github.com/go-chi/chi/v5"

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Get("/token-stats", h.GetTokenStats)
}
