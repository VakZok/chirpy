package api

import (
	"fmt"
	"net/http"
)

func (cfg *Config) metricsHandler(writer http.ResponseWriter, request *http.Request) {
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

func (cfg *Config) resetHandler(writer http.ResponseWriter, request *http.Request) {
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
