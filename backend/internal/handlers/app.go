package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"nomorewaste/internal/payments"
)

type App struct {
	DB        *sql.DB
	JWTSecret string
	Stripe    *payments.Client
	PublicURL string
	DuesCents int64
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func decodeBody(r *http.Request, target interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}
