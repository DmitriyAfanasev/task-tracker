package httpapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	"secunda/internal/delivery/http/openapi"
	"secunda/internal/domain"
)

const maxRequestBodyBytes = 1 << 20 // 1 MiB

// NewRouterOptions задаёт необязательные возможности HTTP-слоя.
type NewRouterOptions struct {
	// EnableSwagger включает Swagger UI и /swagger/doc.json.
	// В production выключается, чтобы не раскрывать контракт и UI.
	EnableSwagger bool

	// RateLimitGlobalRate — общий лимит для authenticated-запросов.
	RateLimitGlobalRate string

	// RateLimitAuthRate — строгий лимит для аутентификации.
	RateLimitAuthRate string
}

// NewRouter собирает HTTP-слой: инфраструктурные middleware, health,
// Swagger UI и сгенерированный по OpenAPI роутер с JWT-аутентификацией.
func NewRouter(handler *Handler, tokens TokenParser, db *sqlx.DB, redisClient *redis.Client, log *slog.Logger, opts NewRouterOptions) (http.Handler, error) {
	spec, err := openapi.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("load openapi spec: %w", err)
	}
	specJSON, err := openapi.GetSpecJSON()
	if err != nil {
		return nil, fmt.Errorf("encode openapi spec: %w", err)
	}

	router := chi.NewRouter()
	router.Use(RequestID)
	router.Use(RequestLogger(log))
	router.Use(Recoverer(log))
	router.Use(chimiddleware.Timeout(30 * time.Second))

	router.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	router.Get("/readyz", readiness(db))

	if opts.EnableSwagger {
		router.Get("/swagger/doc.json", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_, _ = w.Write(specJSON)
		})
		registerSwaggerUI(router)
	}

	strictHandler := openapi.NewStrictHandlerWithOptions(handler, nil, openapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			if isMaxBytesError(err) {
				writeError(w, r, log, domain.ErrPayloadTooLarge.WithErr(err))
				return
			}
			writeError(w, r, log, domain.ErrBadRequest.WithErr(err))
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			writeError(w, r, log, err)
		},
	})

	apiRouter := chi.NewRouter()
	apiRouter.Use(
		LimitBody(maxRequestBodyBytes),
		rateLimitMiddleware(redisClient, opts.RateLimitGlobalRate, log),
		Authenticate(tokens, publicOperations(spec), log),
	)
	openapi.HandlerWithOptions(strictHandler, openapi.ChiServerOptions{BaseRouter: apiRouter})
	router.Mount("/api/v1", apiRouter)

	return router, nil
}

// registerSwaggerUI отдаёт Swagger UI по каноническому /swagger/.
// /swagger и /swagger/index.html перенаправляют на /swagger/, а
// относительные ассеты UI продолжают работать через wildcard-маршрут.
func registerSwaggerUI(router chi.Router) {
	handler := httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json"))

	serveIndex := func(w http.ResponseWriter, r *http.Request) {
		indexRequest := r.Clone(r.Context())
		indexRequest.URL.Path = "/swagger/index.html"
		indexRequest.RequestURI = "/swagger/index.html"
		handler(w, indexRequest)
	}
	redirectToIndex := func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	}

	router.Get("/swagger", redirectToIndex)
	router.Get("/swagger/", serveIndex)
	router.Get("/swagger/index.html", redirectToIndex)
	router.Get("/swagger/*", handler)
}

// publicOperations вычисляет операции без security из OpenAPI-спеки.
// Так список публичных эндпоинтов не может разойтись с контрактом.
func publicOperations(spec *openapi3.T) map[string]struct{} {
	public := make(map[string]struct{})
	if spec == nil || spec.Paths == nil {
		return public
	}

	for path, item := range spec.Paths.Map() {
		if item == nil {
			continue
		}
		for method, operation := range item.Operations() {
			if operation.Security != nil && len(*operation.Security) == 0 {
				public[method+" "+path] = struct{}{}
			}
		}
	}
	return public
}

func readiness(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}
