package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"pulse/backend/internal/config"
	"pulse/backend/internal/domain"
	"pulse/backend/internal/ratelimit"
	"pulse/backend/internal/repo"
	"pulse/backend/internal/service"
)

var pingPathPattern = regexp.MustCompile(`^/(v1/)?ping/[^/]+$`)

type Server struct {
	cfg            config.Config
	logger         *slog.Logger
	store          *repo.Store
	pingLimiter    *ratelimit.Limiter
	invalidLimiter *ratelimit.Limiter
}

func New(cfg config.Config, logger *slog.Logger, store *repo.Store) *Server {
	return &Server{
		cfg:            cfg,
		logger:         logger,
		store:          store,
		pingLimiter:    ratelimit.New(cfg.PingRatePerMinute, cfg.PingBurst),
		invalidLimiter: ratelimit.New(cfg.InvalidTokenRatePerMinute, cfg.InvalidTokenBurst),
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(s.requestLogger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	})

	r.Get("/v1/ping/{token}", s.handlePing)
	r.Post("/v1/ping/{token}", s.handlePing)

	r.Get("/ping/{token}", s.handleDeprecatedPing)
	r.Post("/ping/{token}", s.handleDeprecatedPing)

	// Authenticated management endpoints are stubbed in this scaffold and can be filled incrementally.
	r.Route("/api/v1", func(api chi.Router) {
		api.MethodFunc(http.MethodGet, "/checks", s.notImplemented)
		api.MethodFunc(http.MethodPost, "/checks", s.notImplemented)
		api.MethodFunc(http.MethodGet, "/checks/{id}", s.notImplemented)
		api.MethodFunc(http.MethodPatch, "/checks/{id}", s.notImplemented)
		api.MethodFunc(http.MethodPost, "/checks/{id}/pause", s.notImplemented)
		api.MethodFunc(http.MethodPost, "/checks/{id}/resume", s.notImplemented)
		api.MethodFunc(http.MethodPost, "/checks/{id}/rotate-token", s.notImplemented)
		api.MethodFunc(http.MethodGet, "/checks/{id}/events", s.notImplemented)
		api.MethodFunc(http.MethodGet, "/alert-channels", s.notImplemented)
		api.MethodFunc(http.MethodPost, "/alert-channels", s.notImplemented)
		api.MethodFunc(http.MethodPatch, "/alert-channels/{id}", s.notImplemented)
	})

	return r
}

func (s *Server) handleDeprecatedPing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Deprecation", "true")
	w.Header().Set("Sunset", "Fri, 26 Jun 2026 00:00:00 GMT")
	s.handlePing(w, r)
}

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	setNoStoreHeaders(w)
	token := chi.URLParam(r, "token")
	if token == "" {
		writeError(w, http.StatusBadRequest, "missing token")
		return
	}
	ip := clientIP(r)
	if !s.pingLimiter.Allow(token+"|"+ip, time.Now()) {
		writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}

	input, err := service.ParsePingInput(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := s.store.RecordPing(r.Context(), token, input, s.cfg.IdempotencyTTL)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			if !s.invalidLimiter.Allow("invalid|"+ip, time.Now()) {
				writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			writeError(w, http.StatusNotFound, "check not found")
			return
		}
		s.logger.Error("record ping failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"status":      result.Status,
		"next_due_at": result.NextDueAt.UTC().Format(time.RFC3339),
		"idempotent":  result.Idempotent,
	})
}

func (s *Server) notImplemented(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}

func setNoStoreHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, fmt.Sprintf(`{"code":"encode_error","message":"%v"}`, err), http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]any{
		"code":    http.StatusText(statusCode),
		"message": message,
	})
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	return r.RemoteAddr
}

func (s *Server) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Info("http request",
			"method", r.Method,
			"path", sanitizePath(r.URL.Path),
			"request_id", middleware.GetReqID(r.Context()),
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func sanitizePath(path string) string {
	if pingPathPattern.MatchString(path) {
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) >= 2 {
			parts[len(parts)-1] = "[redacted]"
			return "/" + strings.Join(parts, "/")
		}
	}
	return path
}
