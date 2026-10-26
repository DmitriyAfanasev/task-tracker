package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	libredis "github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	limiterredis "github.com/ulule/limiter/v3/drivers/store/redis"

	"secunda/internal/domain"
)

type contextKey string

const (
	ctxKeyUserID    contextKey = "user_id"
	ctxKeyRequestID contextKey = "request_id"
)

// TokenParser — минимальный порт для проверки JWT в middleware.
type TokenParser interface {
	Parse(raw string) (int64, error)
}

func currentUserID(ctx context.Context) (int64, error) {
	userID, ok := ctx.Value(ctxKeyUserID).(int64)
	if !ok || userID <= 0 {
		return 0, domain.ErrUnauthorized
	}
	return userID, nil
}

func requestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxKeyRequestID).(string)
	return id
}

// RequestID проставляет X-Request-Id и кладёт его в контекст.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get("X-Request-Id"))
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), ctxKeyRequestID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func newRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(buf)
}

// Authenticate проверяет Bearer-токен, кроме публичных операций.
// Ключ publicPaths — "METHOD /path", набор строится из OpenAPI-спеки.
func Authenticate(tokens TokenParser, publicPaths map[string]struct{}, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, public := publicPaths[r.Method+" "+routePath(r)]; public {
				next.ServeHTTP(w, r)
				return
			}

			raw, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok {
				writeError(w, r, log, domain.ErrUnauthorized)
				return
			}

			userID, err := tokens.Parse(raw)
			if err != nil {
				writeError(w, r, log, domain.ErrInvalidToken.WithErr(err))
				return
			}

			ctx := context.WithValue(r.Context(), ctxKeyUserID, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func routePath(r *http.Request) string {
	if routeCtx := chi.RouteContext(r.Context()); routeCtx != nil && routeCtx.RoutePath != "" {
		return routeCtx.RoutePath
	}
	return r.URL.Path
}

// LimitBody ограничивает размер тела запроса, чтобы нельзя было
// отправить гигантский JSON и занять память.
func LimitBody(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequestLogger пишет структурированный лог по каждому запросу.
func RequestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(wrapped, r)

			log.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.Status(),
				"bytes", wrapped.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", requestIDFromContext(r.Context()),
				"remote_addr", r.RemoteAddr,
			)
		})
	}
}

// Recoverer перехватывает паники и возвращает ошибку в едином формате.
func Recoverer(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					if rec == http.ErrAbortHandler {
						panic(rec)
					}
					log.Error("panic recovered",
						"panic", rec,
						"request_id", requestIDFromContext(r.Context()),
						"stack", string(debug.Stack()),
					)
					writeError(w, r, log, domain.ErrInternal)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// isDomainError помогает тестам и обработчику ошибок.
func isDomainError(err error) (*domain.Error, bool) {
	var domainErr *domain.Error
	if errors.As(err, &domainErr) {
		return domainErr, true
	}
	return nil, false
}

// rateLimitMiddleware — chi-совместимое middleware ограничения частоты запросов.
func rateLimitMiddleware(redisClient *libredis.Client, rateFormatted string, log *slog.Logger) func(http.Handler) http.Handler {
	rate, err := limiter.NewRateFromFormatted(rateFormatted)
	if err != nil {
		log.Error("rate limit parse", "rate", rateFormatted, "error", err)
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			store, err := limiterredis.NewStore(redisClient)
			if err != nil {
				log.Warn("rate limit create store", "error", err)
				next.ServeHTTP(w, r)
				return
			}

			lim := limiter.New(store, rate)
			key := lim.GetIP(r).String()
			ctx, err := lim.Get(r.Context(), key)
			if err != nil {
				log.Warn("rate limit check", "error", err)
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("X-RateLimit-Limit", rateFormatted)
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", ctx.Remaining))
			w.Header().Set("X-RateLimit-Reset", time.Unix(ctx.Reset, 0).Format(time.RFC3339))

			if ctx.Reached {
				writeError(w, r, log, domain.ErrRateLimit)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
