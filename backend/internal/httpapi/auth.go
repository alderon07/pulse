package httpapi

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"pulse/backend/internal/config"
)

type authContextKey string

const (
	ctxUserIDKey    authContextKey = "user_id"
	ctxUserEmailKey authContextKey = "user_email"
	jwksCacheTTL                   = 10 * time.Minute
)

var (
	jwksCacheMu sync.RWMutex
	jwksCache   = map[string]cachedJWKS{}
)

type JWTClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

type cachedJWKS struct {
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
}

type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		rawToken := strings.TrimSpace(authHeader[len("Bearer "):])
		claims, err := parseJWT(r.Context(), s.cfg, rawToken)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		userID, err := resolveUserID(claims.Subject, claims.Issuer)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "token subject missing")
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserIDKey, userID)
		ctx = context.WithValue(ctx, ctxUserEmailKey, strings.TrimSpace(claims.Email))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func parseJWT(ctx context.Context, cfg config.Config, token string) (*JWTClaims, error) {
	if cfg.JWTHS256Secret == "" && cfg.JWTJWKSURL == "" {
		return nil, errors.New("jwt verification not configured")
	}

	claims := &JWTClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		switch t.Method.Alg() {
		case jwt.SigningMethodHS256.Alg():
			if cfg.JWTHS256Secret == "" {
				return nil, errors.New("hs256 secret not configured")
			}
			return []byte(cfg.JWTHS256Secret), nil
		case jwt.SigningMethodRS256.Alg():
			if cfg.JWTJWKSURL == "" {
				return nil, errors.New("jwks url not configured")
			}
			rawKid, _ := t.Header["kid"].(string)
			kid := strings.TrimSpace(rawKid)
			if kid == "" {
				return nil, errors.New("missing kid")
			}
			return getJWKSPublicKey(ctx, cfg.JWTJWKSURL, kid)
		default:
			return nil, errors.New("unexpected signing method")
		}
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
	if strings.TrimSpace(claims.Subject) == "" {
		return nil, errors.New("missing subject")
	}
	return claims, nil
}

func resolveUserID(subject, issuer string) (string, error) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return "", errors.New("missing subject")
	}

	if _, err := uuid.Parse(subject); err == nil {
		return subject, nil
	}

	seed := strings.TrimSpace(issuer) + "|" + subject
	derived := uuid.NewSHA1(uuid.NameSpaceURL, []byte(seed))
	return derived.String(), nil
}

func getJWKSPublicKey(ctx context.Context, jwksURL, kid string) (*rsa.PublicKey, error) {
	if cachedKey, ok := getCachedJWK(jwksURL, kid); ok {
		return cachedKey, nil
	}

	keys, err := fetchJWKS(ctx, jwksURL)
	if err != nil {
		return nil, err
	}

	jwksCacheMu.Lock()
	jwksCache[jwksURL] = cachedJWKS{
		keys:      keys,
		expiresAt: time.Now().UTC().Add(jwksCacheTTL),
	}
	jwksCacheMu.Unlock()

	key, ok := keys[kid]
	if !ok {
		return nil, fmt.Errorf("kid %q not found in jwks", kid)
	}
	return key, nil
}

func getCachedJWK(jwksURL, kid string) (*rsa.PublicKey, bool) {
	now := time.Now().UTC()

	jwksCacheMu.RLock()
	cached, ok := jwksCache[jwksURL]
	jwksCacheMu.RUnlock()
	if !ok || !cached.expiresAt.After(now) {
		return nil, false
	}

	key, keyOk := cached.keys[kid]
	if !keyOk {
		return nil, false
	}
	return key, true
}

func fetchJWKS(ctx context.Context, jwksURL string) (map[string]*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create jwks request: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("jwks endpoint returned status %d", resp.StatusCode)
	}

	var payload jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode jwks response: %w", err)
	}

	parsed := make(map[string]*rsa.PublicKey, len(payload.Keys))
	for _, key := range payload.Keys {
		if strings.TrimSpace(key.Kid) == "" || key.Kty != "RSA" {
			continue
		}

		nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
		if err != nil {
			continue
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
		if err != nil {
			continue
		}

		e := 0
		for _, b := range eBytes {
			e = (e << 8) + int(b)
		}
		if e <= 0 {
			continue
		}

		parsed[key.Kid] = &rsa.PublicKey{
			N: new(big.Int).SetBytes(nBytes),
			E: e,
		}
	}

	if len(parsed) == 0 {
		return nil, errors.New("no usable RSA keys in jwks response")
	}
	return parsed, nil
}

func userFromContext(ctx context.Context) (string, string, error) {
	userID, ok := ctx.Value(ctxUserIDKey).(string)
	if !ok || strings.TrimSpace(userID) == "" {
		return "", "", errors.New("user missing in context")
	}
	email, _ := ctx.Value(ctxUserEmailKey).(string)
	return userID, email, nil
}
