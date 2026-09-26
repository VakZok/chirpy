package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/VakZok/chirpy/internal/auth"
	"github.com/VakZok/chirpy/internal/database"
	"github.com/google/uuid"
)

var bannedWords = map[string]bool{
	"kerfuffle": true,
	"sharbert":  true,
	"fornax":    true,
}

func cleanBody(body string) string {
	words := strings.Split(body, " ")
	for idx, word := range words {
		if bannedWords[strings.ToLower(word)] {
			words[idx] = "****"
		}
	}

	cleanedBody := strings.Join(words, " ")
	return cleanedBody
}

func (cfg *Config) createChirpHandler(w http.ResponseWriter, r *http.Request) {
	// Client Authentication
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, fmt.Sprintf("could not extract token: %s", err))
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret) // get user from access token
	if err != nil {
		respondWithError(w, 401, fmt.Sprintf("user not authorized: %s", err))
		return
	}

	// JSON decoding
	type parameters struct { // what we expect (the client request we get)
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params) // we decode response.Body into the parameters struct using pointers
	if err != nil {               // decoding unsucessfull
		respondWithError(w, 400, fmt.Sprintf("Error decoding parameters: %s", err))
		return
	}

	if len(params.Body) > 140 { // body too long
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	// JSON properly decoded and has proper format (chirp aka tweet is valid)

	// Replace any "profane" words
	cleanedBody := cleanBody(params.Body)

	// Fill DB query struct (use auto-generated struct from saveChirp.sql.go)
	queryParameters := database.SaveChirpParams{
		Body:   cleanedBody,
		UserID: userID,
	}

	// Query Database
	dbChirp, err := cfg.db.SaveChirp(r.Context(), queryParameters)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error posting chirp: %s", err))
		return
	}

	respBody := Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}

	respondWithJSON(w, 201, respBody)
}

func (cfg *Config) getChirpsHandler(w http.ResponseWriter, r *http.Request) {
	// since request does not contain json/body, we can tackle the db directly
	dbChirps, err := cfg.db.GetAllChirps(r.Context())
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error getting chirps: %s", err))
		return
	}

	// transform chirps from DB to struct type
	chirps := []Chirp{}
	for _, dbChirp := range dbChirps {
		chirp := Chirp{
			ID:        dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt,
			UpdatedAt: dbChirp.UpdatedAt,
			Body:      dbChirp.Body,
			UserID:    dbChirp.UserID,
		}

		chirps = append(chirps, chirp)
	}

	respondWithJSON(w, 200, chirps)
}

func (cfg *Config) getChirpHandler(w http.ResponseWriter, r *http.Request) {
	// since request does not contain json/body, we can tackle the db directly
	// but first: ransform String to UUID type
	parsedID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, 400, fmt.Sprintf("Error parsing UserID: %s", err))
		return
	}

	dbChirp, err := cfg.db.GetChirp(r.Context(), parsedID)
	if err != nil {
		respondWithError(w, 404, fmt.Sprintf("Error getting chirp: %s", err))
		return
	}

	// transform chirps from DB to struct type
	chirp := Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}

	respondWithJSON(w, 200, chirp)
}


func (cfg *Config) deleteChirpHandler(w http.ResponseWriter, r *http.Request) {
	// Client Authentication
	accessToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, fmt.Sprintf("could not extract token: %s", err))
		return
	}

	userID, err := auth.ValidateJWT(accessToken, cfg.jwtSecret) // get user from access token
	if err != nil {
		respondWithError(w, 401, fmt.Sprintf("user not authorized: %s", err))
		return
	}

	// ransform String to UUID type
	chirpID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, 400, fmt.Sprintf("error parsing UserID: %s", err))
		return
	}

	// get chirp
	dbChirp, err := cfg.db.GetChirp(r.Context(), chirpID)
	if err != nil {
		respondWithError(w, 404, fmt.Sprintf("chirp not found: %s", err))
		return
	}

	// check that the client sending the deletion request is the author of the chirp
	if dbChirp.UserID != userID {
		respondWithError(w, 403, "user not authenticated to delete this	tweet")
		return
	}

	err = cfg.db.DeleteChirp(r.Context(), chirpID)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("error deleting chirp: %s", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}