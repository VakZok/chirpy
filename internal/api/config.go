package api

import (
	"sync/atomic"

	"github.com/VakZok/chirpy/internal/database"
)

type Config struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
	jwtSecret      string
}

func NewConfig(db *database.Queries, platform string, jwtSecret string) *Config {
	return &Config{
		db:        db,
		platform:  platform,
		jwtSecret: jwtSecret,
	}
}
