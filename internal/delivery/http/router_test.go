package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"secunda/internal/delivery/http/openapi"
)

func TestSwaggerUIHasCanonicalURL(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router, err := NewRouter(nil, nil, nil, logger, RouterOptions{EnableSwagger: true})
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	t.Run("serves UI at trailing slash", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/swagger/", nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "swagger-ui") || !strings.Contains(body, "doc.json") {
			t.Fatalf("response does not look like Swagger UI: status=%d location=%q body=%q", rec.Code, rec.Header().Get("Location"), body)
		}
	})

	t.Run("serves static assets", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/swagger/swagger-ui.css", nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "text/css") {
			t.Fatalf("Content-Type = %q, want text/css", contentType)
		}
	})

	for _, path := range []string{"/swagger", "/swagger/index.html"} {
		t.Run("redirects "+path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

			if rec.Code != http.StatusMovedPermanently {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusMovedPermanently)
			}
			if location := rec.Header().Get("Location"); location != "/swagger/" {
				t.Fatalf("Location = %q, want /swagger/", location)
			}
		})
	}
}

func TestSwaggerIsDisabledWhenNotEnabled(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router, err := NewRouter(nil, nil, nil, logger, RouterOptions{EnableSwagger: false})
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	for _, path := range []string{"/swagger", "/swagger/", "/swagger/doc.json", "/swagger/swagger-ui.css"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
			}
		})
	}
}

func TestPublicOperationsAreDerivedFromSpec(t *testing.T) {
	spec, err := openapi.GetSpec()
	if err != nil {
		t.Fatalf("GetSpec() error = %v", err)
	}

	public := publicOperations(spec)

	wantPublic := []string{"POST /register", "POST /login"}
	for _, operation := range wantPublic {
		if _, ok := public[operation]; !ok {
			t.Errorf("operation %q must be public", operation)
		}
	}

	wantProtected := []string{"GET /teams", "POST /teams", "POST /teams/{id}/invite", "GET /teams/{team_id}/stats"}
	for _, operation := range wantProtected {
		if _, ok := public[operation]; ok {
			t.Errorf("operation %q must be protected", operation)
		}
	}
}
