package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/VakZok/chirpy/internal/auth"
	"github.com/VakZok/chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
}

type chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
	jwtSecret      string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(writer, request)
	})
}

func (cfg *apiConfig) myMetricHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "text/html")
	writer.WriteHeader(http.StatusOK)
	writer.Write([]byte(fmt.Sprintf(
		`<html>
  			<body>
     			<h1>Welcome, Chirpy Admin</h1>
        		<p>Chirpy has been visited %d times!</p>
          	</body>
		</html>`, cfg.fileserverHits.Load(),
	)))
}

func (cfg *apiConfig) myResetHandler(writer http.ResponseWriter, request *http.Request) {
	if cfg.platform != "dev" {
		writer.WriteHeader(http.StatusForbidden)
		return
	}

	// reset page visit counter
	cfg.fileserverHits.Store(0)

	// reset aka delete allusers from database
	err := cfg.db.DeleteAllUsers(request.Context())
	if err != nil {
		respondWithError(writer, 500, fmt.Sprintf("Error deleteing user: %s", err))
		return
	}

	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type returnVals struct {
		Error string `json:"error"`
	}

	respBody := returnVals{
		Error: msg,
	}

	respondWithJSON(w, code, respBody)
}

func cleanBody(body string) string {
	words := strings.Split(body, " ")
	for idx, word := range words {
		if strings.ToLower(word) == "kerfuffle" || strings.ToLower(word) == "sharbert" || strings.ToLower(word) == "fornax" {
			words[idx] = "****"
		}
	}

	cleanedBody := strings.Join(words, " ")
	return cleanedBody
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}

func (cfg *apiConfig) createChirpHandler(w http.ResponseWriter, r *http.Request) {
	// Client Authentication
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, fmt.Sprintf("could not extract token: %s", err))
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
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

	respBody := chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}

	respondWithJSON(w, 201, respBody)
}

func (cfg *apiConfig) myUserHandler(w http.ResponseWriter, r *http.Request) {
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

func (cfg *apiConfig) loginHandler(w http.ResponseWriter, r *http.Request) {
	// JSON decoding
	type parameters struct { // what we expect (the request we get)
		Password         string `json:"password"`
		Email            string `json:"email"`
		ExpiresInSeconds int    `json:"expires_in_seconds"` // optionally sent by client
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
	if match == false {
		respondWithError(w, 401, fmt.Sprint("Incorrect email or password"))
		return
	}

	// configure authentication token expiry
	if params.ExpiresInSeconds == 0 || params.ExpiresInSeconds > 3600 {
		params.ExpiresInSeconds = 3600 // one hour in seconds
	}

	// generate new authentication token
	token, err := auth.MakeJWT(dbUser.ID, cfg.jwtSecret, time.Duration(params.ExpiresInSeconds)*time.Second)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("failed to create authentication token: %s", err))
		return
	}

	respBody := User{ // fill user struct with values from db user
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
		Token:     token,
	}

	respondWithJSON(w, 200, respBody)
}

func (cfg *apiConfig) getChirpsHandler(w http.ResponseWriter, r *http.Request) {
	// since request does not contain json/body, we can tackle the db directly
	dbChirps, err := cfg.db.GetAllChirps(r.Context())
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error getting chirps: %s", err))
		return
	}

	// transform chirps from DB to struct type
	chirps := []chirp{}
	for _, dbChirp := range dbChirps {
		chirp := chirp{
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

func (cfg *apiConfig) getChirpHandler(w http.ResponseWriter, r *http.Request) {
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
	chirp := chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}

	respondWithJSON(w, 200, chirp)
}

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Printf("Error connecting to DB: %s", err)
		return
	}

	dbQueries := database.New(db)

	myHandler := http.NewServeMux()

	// Load environmental variables
	platform := os.Getenv("PLATFORM")
	if platform == "" {
		log.Fatal("PLATFORM must be set")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET must be set")
	}

	cfg := &apiConfig{
		db:        dbQueries,
		platform:  platform,
		jwtSecret: jwtSecret,
	}

	// Endpoints
	// handling fileserver
	fileServer := http.FileServer(http.Dir("."))
	strippedPrefixFileServer := http.StripPrefix("/app", fileServer)
	wrappedFileServer := cfg.middlewareMetricsInc(strippedPrefixFileServer)
	myHandler.Handle("/app/", wrappedFileServer)

	// handling health data using anonymous function that returns handler function
	myHandler.HandleFunc("GET /api/healthz", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		writer.WriteHeader(http.StatusOK)
		writer.Write([]byte("OK"))
	})

	// handling website visit counter
	myHandler.HandleFunc("GET /admin/metrics", cfg.myMetricHandler)
	myHandler.HandleFunc("POST /admin/reset", cfg.myResetHandler)

	// handle posting a chirp in json format (cleaned some bad words)
	myHandler.HandleFunc("POST /api/chirps", cfg.createChirpHandler)

	// handle new user creation
	myHandler.HandleFunc("POST /api/users", cfg.myUserHandler)

	// handle retrieving chirps
	myHandler.HandleFunc("GET /api/chirps/{chirpID}", cfg.getChirpHandler)
	myHandler.HandleFunc("GET /api/chirps", cfg.getChirpsHandler)

	// handle login authentication
	myHandler.HandleFunc("POST /api/login", cfg.loginHandler)

	// configuring http server
	s := &http.Server{
		Addr:    ":8080",
		Handler: myHandler,
	}

	// start server and have it continously run listening for requests to serve
	log.Fatal(s.ListenAndServe())
}
