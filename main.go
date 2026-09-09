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

	"github.com/VakZok/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
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
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
	cfg.fileserverHits.Store(0)
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

func (cfg *apiConfig) myChirpValidationHandler(w http.ResponseWriter, r *http.Request) {
	// JSON decoding
	type parameters struct { // what we expect
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params) // we decode response.Body into the parameters struct using pointers
	if err != nil {                // decoding unsucessfull
		respondWithError(w, 500, fmt.Sprintf("Error decoding parameters: %s", err))
		return
	}

	if len(params.Body) > 140 { // body too long
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	// JSON properly decoded and has proper format
	// Replace any "profane" words

	cleanedBody := cleanBody(params.Body)

	type returnVals struct {
		Cleaned_body string `json:"cleaned_body"`
	}

	respBody := returnVals{
		Cleaned_body: cleanedBody,
	}

	respondWithJSON(w, 200, respBody)
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
	cfg := &apiConfig{
		db: dbQueries,
	}

	// handling fileserver
	fileServer := http.FileServer(http.Dir("."))
	strippedPrefixFileServer := http.StripPrefix("/app", fileServer)
	wrappedFileServer := cfg.middlewareMetricsInc(strippedPrefixFileServer)
	myHandler.Handle("/app/", wrappedFileServer)

	// handling healt data using anonymous function that returns handler function
	myHandler.HandleFunc("GET /api/healthz", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		writer.WriteHeader(http.StatusOK)
		writer.Write([]byte("OK"))
	})

	// handling website visit counter
	myHandler.HandleFunc("GET /admin/metrics", cfg.myMetricHandler)
	myHandler.HandleFunc("POST /admin/reset", cfg.myResetHandler)

	// new route for json
	myHandler.HandleFunc("POST /api/validate_chirp", cfg.myChirpValidationHandler)

	// configuring http server
	s := &http.Server{
		Addr:    ":8080",
		Handler: myHandler,
	}

	// start server and have it continously run listening for requests to serve
	log.Fatal(s.ListenAndServe())
}
