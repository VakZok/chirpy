package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGetBearerToken_Success(t *testing.T) {
	headers := http.Header{}
	headers.Set("Authorization", "Bearer my.jwt.token")

	token, err := GetBearerToken(headers)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if token != "my.jwt.token" {
		t.Errorf("expected token %q, got %q", "my.jwt.token", token)
	}
}

func TestGetBearerToken_MissingHeader(t *testing.T) {
	headers := http.Header{}

	_, err := GetBearerToken(headers)
	if err == nil {
		t.Fatal("expected an error for missing Authorization header, got nil")
	}
}

func TestGetBearerToken_EmptyHeader(t *testing.T) {
	headers := http.Header{}
	headers.Set("Authorization", "")

	_, err := GetBearerToken(headers)
	if err == nil {
		t.Fatal("expected an error for empty Authorization header, got nil")
	}
}

func TestGetBearerToken_NoBearerPrefix(t *testing.T) {
	headers := http.Header{}
	headers.Set("Authorization", "my.jwt.token")

	// TrimPrefix is a no-op when the prefix isn't present, so the raw
	// value is returned unchanged rather than an error.
	token, err := GetBearerToken(headers)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if token != "my.jwt.token" {
		t.Errorf("expected token %q, got %q", "my.jwt.token", token)
	}
}

func TestMakeAndValidateJWT_Success(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"

	token, err := MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatalf("expected no error creating JWT, got: %v", err)
	}

	gotID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("expected no error validating JWT, got: %v", err)
	}
	if gotID != userID {
		t.Errorf("expected userID %v, got %v", userID, gotID)
	}
}

func TestValidateJWT_WrongSecret(t *testing.T) {
	userID := uuid.New()
	token, err := MakeJWT(userID, "correct-secret", time.Hour)
	if err != nil {
		t.Fatalf("expected no error creating JWT, got: %v", err)
	}

	_, err = ValidateJWT(token, "wrong-secret")
	if err == nil {
		t.Fatal("expected an error validating JWT with wrong secret, got nil")
	}
}

func TestValidateJWT_Expired(t *testing.T) {
	userID := uuid.New()
	token, err := MakeJWT(userID, "test-secret", -time.Hour)
	if err != nil {
		t.Fatalf("expected no error creating JWT, got: %v", err)
	}

	_, err = ValidateJWT(token, "test-secret")
	if err == nil {
		t.Fatal("expected an error validating an expired JWT, got nil")
	}
}

func TestHashAndCheckPassword(t *testing.T) {
	password := "correct-horse-battery-staple"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("expected no error hashing password, got: %v", err)
	}

	match, err := CheckPasswordHash(password, hash)
	if err != nil {
		t.Fatalf("expected no error checking password hash, got: %v", err)
	}
	if !match {
		t.Error("expected password to match hash")
	}

	match, err = CheckPasswordHash("wrong-password", hash)
	if err != nil {
		t.Fatalf("expected no error checking password hash, got: %v", err)
	}
	if match {
		t.Error("expected wrong password not to match hash")
	}
}
