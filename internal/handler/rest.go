package handler

import (
	"encoding/json"
	"net/http"

	"git.blinkpay.vn/software/authix/config"
	"git.blinkpay.vn/software/authix/internal/persistence"
)

type RESTHandler struct {
	persister persistence.Persister
	config    *config.Config
}

func NewREST(persister persistence.Persister, cfg *config.Config) *RESTHandler {
	return &RESTHandler{
		persister: persister,
		config:    cfg,
	}
}

func (h *RESTHandler) respondWithError(w http.ResponseWriter, code int, message string) {
	h.respondWithJSON(w, code, map[string]string{"error": message})
}

func (h *RESTHandler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}
