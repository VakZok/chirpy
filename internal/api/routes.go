package api

import "net/http"

// NewRouter builds and returns the fully configured HTTP handler for the app.
func NewRouter(cfg *Config) http.Handler {
	mux := http.NewServeMux()

	// handling fileserver
	fileServer := http.FileServer(http.Dir("."))
	strippedPrefixFileServer := http.StripPrefix("/app", fileServer)
	wrappedFileServer := cfg.middlewareMetricsInc(strippedPrefixFileServer)
	mux.Handle("/app/", wrappedFileServer)

	// handling health data using anonymous function that returns handler function
	mux.HandleFunc("GET /api/healthz", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		writer.WriteHeader(http.StatusOK)
		writer.Write([]byte("OK"))
	})

	// handling website visit counter
	mux.HandleFunc("GET /admin/metrics", cfg.metricsHandler)
	mux.HandleFunc("POST /admin/reset", cfg.resetHandler)

	// handle posting a chirp in json format (cleaned some bad words)
	mux.HandleFunc("POST /api/chirps", cfg.createChirpHandler)

	// delete chirps
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", cfg.deleteChirpHandler)

	// handle new user creation
	mux.HandleFunc("POST /api/users", cfg.createUserHandler)

	// provide ability for users to change their email and password
	mux.HandleFunc("PUT /api/users", cfg.updateUserHandler)

	// handle retrieving chirps
	mux.HandleFunc("GET /api/chirps/{chirpID}", cfg.getChirpHandler)
	mux.HandleFunc("GET /api/chirps", cfg.getChirpsHandler)

	// handle login authentication
	mux.HandleFunc("POST /api/login", cfg.loginHandler)
	mux.HandleFunc("POST /api/refresh", cfg.refreshHandler)
	mux.HandleFunc("POST /api/revoke", cfg.revokeHandler)

	return mux
}
