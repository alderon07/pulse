package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"pulse/backend/internal/config"
)

type authContextKey string

const (
	ctxUserIDKey    authContextKey = "user_id"
	ctxUserEmailKey authContextKey = "user_email"
)

type JWTClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		rawToken := strings.TrimSpace(authHeader[len("Bearer "):])
		claims, err := parseJWT(s.cfg, rawToken)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		userID := strings.TrimSpace(claims.Subject)
		if _, err := uuid.Parse(userID); err != nil {
			writeError(w, http.StatusUnauthorized, "token subject must be UUID")
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserIDKey, userID)
		ctx = context.WithValue(ctx, ctxUserEmailKey, strings.TrimSpace(claims.Email))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func parseJWT(cfg config.Config, token string) (*JWTClaims, error) {
	if cfg.JWTHS256Secret == "" {
		return nil, errors.New("jwt secret not configured")
	}
	claims := &JWTClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(cfg.JWTHS256Secret), nil
	})
	if err != nil || !parsed.Valid {
		return nil, errors.New("invalid jwt")
	}

	if cfg.JWTIssuer != "" && claims.Issuer != cfg.JWTIssuer {
		return nil, errors.New("unexpected issuer")
	}
	if cfg.JWTAudience != "" {
		ok := false
		for _, aud := range claims.Audience {
			if aud == cfg.JWTAudience {
				ok = true
				break
			}
		}
		if !ok {
			return nil, errors.New("unexpected audience")
		}
	}
	if claims.Subject == "" {
		return nil, errors.New("missing subject")
	}
	return claims, nil
}

func userFromContext(ctx context.Context) (string, string, error) {
	userID, ok := ctx.Value(ctxUserIDKey).(string)
	if !ok || strings.TrimSpace(userID) == "" {
		return "", "", errors.New("user missing in context")
	}
	email, _ := ctx.Value(ctxUserEmailKey).(string)
	return userID, email, nil
}
