package main

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestCleanBody(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "no profane words",
			body: "This is a clean chirp",
			want: "This is a clean chirp",
		},
		{
			name: "single profane word",
			body: "This is kerfuffle",
			want: "This is ****",
		},
		{
			name: "multiple profane words, mixed case",
			body: "Kerfuffle and SHARBERT and fornax",
			want: "**** and **** and ****",
		},
		{
			name: "profane word as substring is not replaced",
			body: "kerfufflewhat is happening",
			want: "kerfufflewhat is happening",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanBody(tt.body)
			if got != tt.want {
				t.Errorf("cleanBody(%q) = %q, want %q", tt.body, got, tt.want)
			}
		})
	}
}

func TestRespondWithJSON(t *testing.T) {
	recorder := httptest.NewRecorder()
	payload := map[string]string{"foo": "bar"}

	respondWithJSON(recorder, 201, payload)

	if recorder.Code != 201 {
		t.Errorf("expected status 201, got %d", recorder.Code)
	}

	contentType := recorder.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", contentType)
	}

	var got map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}
	if got["foo"] != "bar" {
		t.Errorf("expected body foo=bar, got %v", got)
	}
}

func TestRespondWithError(t *testing.T) {
	recorder := httptest.NewRecorder()

	respondWithError(recorder, 400, "something went wrong")

	if recorder.Code != 400 {
		t.Errorf("expected status 400, got %d", recorder.Code)
	}

	var got struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}
	if got.Error != "something went wrong" {
		t.Errorf("expected error message %q, got %q", "something went wrong", got.Error)
	}
}
