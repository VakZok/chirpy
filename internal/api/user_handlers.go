package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/VakZok/chirpy/internal/auth"
	"github.com/VakZok/chirpy/internal/database"
)

func (cfg *Config) createUserHandler(w http.ResponseWriter, r *http.Request) {
	// JSON decoding
	type parameters struct { // what we expect (the request we get)
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params) //decode into params struct using pointer
	if err != nil {
		respondWithError(w, 400, fmt.Sprintf("Error decoding parameters: %s", err))
		return
	}

	// hash password
	hash, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, 400, fmt.Sprintf("Error hashing password: %s", err))
		return
	}

	queryParameters := database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hash,
	}

	dbUser, err := cfg.db.CreateUser(r.Context(), queryParameters) // set and get user from db
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error creating user: %s", err))
		return
	}

	respBody := User{ // fill user struct with values from db user
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
	}

	respondWithJSON(w, 201, respBody)
}

func (cfg *Config) loginHandler(w http.ResponseWriter, r *http.Request) {
	// JSON decoding
	type parameters struct { // what we expect (the request we get)
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	type response struct {
		User
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params) //decode into params struct using pointer
	if err != nil {
		respondWithError(w, 400, fmt.Sprintf("Error decoding parameters: %s", err))
		return
	}

	dbUser, err := cfg.db.GetUserByEmail(r.Context(), params.Email) // get user from db
	if err != nil {
		respondWithError(w, 401, fmt.Sprint("Incorrect email or password"))
		return
	}

	// check for matching password
	match, err := auth.CheckPasswordHash(params.Password, dbUser.HashedPassword)
	if err != nil {
		respondWithError(w, 401, fmt.Sprint("Incorrect email or password"))
		return
	}
	if !match {
		respondWithError(w, 401, fmt.Sprint("Incorrect email or password"))
		return
	}

	// generate new authentication token
	authenticationToken, err := auth.MakeJWT(
		dbUser.ID,
		cfg.jwtSecret,
		time.Hour,
	)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("failed to create authentication token: %s", err))
		return
	}

	// generate new refresh token and add to user account
	refreshToken := auth.MakeRefreshToken()

	refreshTokenParams := database.CreateRefreshTokenParams{
		UserID:    dbUser.ID,
		Token:     refreshToken,
		ExpiresAt: time.Now().UTC().Add(time.Hour * 24 * 60),
	}

	_, err = cfg.db.CreateRefreshToken(r.Context(), refreshTokenParams)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("failed to create refresh token: %s", err))
		return
	}

	respBody := response{ // fill struct with values from db user
		User: User{
			ID:        dbUser.ID,
			CreatedAt: dbUser.CreatedAt,
			UpdatedAt: dbUser.UpdatedAt,
			Email:     dbUser.Email,
		},
		Token:        authenticationToken,
		RefreshToken: refreshToken,
	}

	respondWithJSON(w, 200, respBody)
}

func (cfg *Config) refreshHandler(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("could not find refresh token: %s", err))
		return
	}

	dbUser, err := cfg.db.GetUserFromRefreshToken(r.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, fmt.Sprintf("Couldn't get user for refresh token: %s", err))
		return
	}

	accessToken, err := auth.MakeJWT(
		dbUser.ID,
		cfg.jwtSecret,
		time.Hour,
	)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, fmt.Sprintf("Couldn't validate token: %s", err))
		return
	}

	type response struct {
		Token string `json:"token"`
	}

	respBody := response{
		Token: accessToken,
	}

	respondWithJSON(w, http.StatusOK, respBody)
}

func (cfg *Config) revokeHandler(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("could not find refresh token: %s", err))
		return
	}

	_, err = cfg.db.RevokeRefreshToken(r.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Couldn't revoke session: %s", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
	
func (cfg *Config) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	// JSON decoding
	type parameters struct { // new credentials we get from with the request
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params) //decode into params struct using pointer
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("error decoding parameters: %s", err))
		return
	}

	// get user by token
	accessToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, fmt.Sprintf("could not find access token: %s", err))
		return
	}

	userID, err := auth.ValidateJWT(accessToken, cfg.jwtSecret) // get user from access token
	if err != nil {
		respondWithError(w, 401, fmt.Sprintf("user not authorized: %s", err))
		return
	}

	// hash password
	hash, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("error hashing password: %s", err))
		return
	}

	// update email and password
	dbParams := database.UpdateUserParams{
		Email:          params.Email,
		HashedPassword: hash,
		ID:             userID,
	}

	dbUser, err := cfg.db.UpdateUser(r.Context(), dbParams)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("error updating user: %s", err))
		return
	}

	responseBody := User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
	}

	respondWithJSON(w, http.StatusOK, responseBody)
}
