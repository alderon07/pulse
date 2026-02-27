package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"pulse/backend/internal/domain"
)

const maxPingBodyBytes = 16 * 1024
const maxDurationMS = 86_400_000
const maxOutputSize = 1_000_000_000

type pingJSON struct {
	DurationMS *int  `json:"duration_ms"`
	OutputSize *int  `json:"output_size"`
	Success    *bool `json:"success"`
}

func ParsePingInput(r *http.Request) (domain.PingInput, error) {
	input := domain.PingInput{}

	queryDuration, qDurOK, err := readInt(r.URL.Query().Get("duration_ms"))
	if err != nil {
		return input, fmt.Errorf("invalid query duration_ms: %w", err)
	}
	queryOutput, qOutOK, err := readInt(r.URL.Query().Get("output_size"))
	if err != nil {
		return input, fmt.Errorf("invalid query output_size: %w", err)
	}
	querySuccess, qSucOK, err := readBool(r.URL.Query().Get("success"))
	if err != nil {
		return input, fmt.Errorf("invalid query success: %w", err)
	}

	headDuration, hDurOK, err := readInt(r.Header.Get("X-Duration-MS"))
	if err != nil {
		return input, fmt.Errorf("invalid X-Duration-MS: %w", err)
	}
	headOutput, hOutOK, err := readInt(r.Header.Get("X-Output-Size"))
	if err != nil {
		return input, fmt.Errorf("invalid X-Output-Size: %w", err)
	}
	headSuccess, hSucOK, err := readBool(r.Header.Get("X-Success"))
	if err != nil {
		return input, fmt.Errorf("invalid X-Success: %w", err)
	}

	if qDurOK {
		input.DurationMS = &queryDuration
	}
	if qOutOK {
		input.OutputSize = &queryOutput
	}
	if qSucOK {
		input.Success = &querySuccess
	}

	if hDurOK {
		input.DurationMS = &headDuration
	}
	if hOutOK {
		input.OutputSize = &headOutput
	}
	if hSucOK {
		input.Success = &headSuccess
	}

	if r.Method == http.MethodPost {
		var payload pingJSON
		body, err := io.ReadAll(io.LimitReader(r.Body, maxPingBodyBytes+1))
		if err != nil {
			return input, fmt.Errorf("read request body: %w", err)
		}
		if len(body) > maxPingBodyBytes {
			return input, fmt.Errorf("request body exceeds %d bytes", maxPingBodyBytes)
		}
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil && err.Error() != "EOF" {
			return input, fmt.Errorf("invalid JSON body: %w", err)
		}
		if payload.DurationMS != nil {
			input.DurationMS = payload.DurationMS
		}
		if payload.OutputSize != nil {
			input.OutputSize = payload.OutputSize
		}
		if payload.Success != nil {
			input.Success = payload.Success
		}
	}

	if err := validateInputBounds(input); err != nil {
		return input, err
	}
	input.IdempotencyKey = truncate(r.Header.Get("Idempotency-Key"), 128)
	input.SourceIP = truncate(clientIP(r), 64)
	input.UserAgent = truncate(r.UserAgent(), 512)
	return input, nil
}

func validateInputBounds(input domain.PingInput) error {
	if input.DurationMS != nil {
		if *input.DurationMS < 0 {
			return fmt.Errorf("duration_ms must be >= 0")
		}
		if *input.DurationMS > maxDurationMS {
			return fmt.Errorf("duration_ms exceeds max allowed")
		}
	}
	if input.OutputSize != nil {
		if *input.OutputSize < 0 {
			return fmt.Errorf("output_size must be >= 0")
		}
		if *input.OutputSize > maxOutputSize {
			return fmt.Errorf("output_size exceeds max allowed")
		}
	}
	return nil
}

func readInt(raw string) (int, bool, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, false, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false, err
	}
	return v, true, nil
}

func readBool(raw string) (bool, bool, error) {
	if strings.TrimSpace(raw) == "" {
		return false, false, nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, false, err
	}
	return v, true, nil
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	hostPort := r.RemoteAddr
	if idx := strings.LastIndex(hostPort, ":"); idx > 0 {
		return hostPort[:idx]
	}
	return hostPort
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
