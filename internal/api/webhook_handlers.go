package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

func (cfg *Config) webhookHandler(w http.ResponseWriter, r *http.Request) {
	// JSON decoding
	type parameters struct { // what we expect (the request we get)
		Event string `json:"event"`
		Data  struct {
			UserID uuid.UUID `json:"user_id"`
		} `json:"data"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params) //decode into params struct using pointer
	if err != nil {
		respondWithError(w, 400, fmt.Sprintf("Error decoding parameters: %s", err))
		return
	}

	// any event other than upgrading should immediately return 204
	if params.Event != "user.upgraded" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	rows, err := cfg.db.UpgradeUserMembership(r.Context(), params.Data.UserID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not upgrade user")
		return
	}
	if rows == 0 {
		respondWithError(w, http.StatusNotFound, "user not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
