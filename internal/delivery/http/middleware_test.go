package httpapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLimitBodyRejectsOversizedRequest(t *testing.T) {
	var readErr error
	handler := LimitBody(4)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
		if readErr != nil {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
		}
	}))

	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("12345"))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if readErr == nil {
		t.Fatal("expected body read error for oversized request")
	}
	if !isMaxBytesError(readErr) {
		t.Fatalf("expected MaxBytesError, got %v", readErr)
	}
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestLimitBodyAllowsSmallRequest(t *testing.T) {
	handler := LimitBody(4)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if string(body) != "1234" {
			t.Fatalf("body = %q, want %q", body, "1234")
		}
	}))

	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("1234"))
	handler.ServeHTTP(httptest.NewRecorder(), request)
}
