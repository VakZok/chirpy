package auth

import (
	"github.com/alexedwards/argon2id"
)

// https://pkg.go.dev/github.com/alexedwards/argon2id#section-readme
func HashPassword(password string) (string, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	return hash, err
}

func CheckPasswordHash(password, hash string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	return match, err
}	