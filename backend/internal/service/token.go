package service

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func GenerateSecureToken(size int) (string, error) {
	if size <= 0 {
		size = 32
	}
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate secure token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
