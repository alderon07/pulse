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
const maxMessageChars = 1024
const maxEnvironmentChars = 64
const maxMetricChars = 256

type pingJSON struct {
	DurationMS *int    `json:"duration_ms"`
	OutputSize *int    `json:"output_size"`
	Success    *bool   `json:"success"`
	State      *string `json:"state"`
	Msg        *string `json:"msg"`
	Env        *string `json:"env"`
	Metric     *string `json:"metric"`
}

func ParsePingInput(r *http.Request) (domain.PingInput, error) {
	input := domain.PingInput{EventType: "ping"}

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
	queryState, qStateOK, err := readState(r.URL.Query().Get("state"))
	if err != nil {
		return input, fmt.Errorf("invalid query state: %w", err)
	}
	queryMsg, qMsgOK := readString(r.URL.Query().Get("msg"))
	queryEnv, qEnvOK := readString(r.URL.Query().Get("env"))
	queryMetric, qMetricOK := readString(r.URL.Query().Get("metric"))

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
	headState, hStateOK, err := readState(r.Header.Get("X-State"))
	if err != nil {
		return input, fmt.Errorf("invalid X-State: %w", err)
	}
	headMsg, hMsgOK := readString(r.Header.Get("X-Msg"))
	headEnv, hEnvOK := readString(r.Header.Get("X-Env"))
	headMetric, hMetricOK := readString(r.Header.Get("X-Metric"))

	if qDurOK {
		input.DurationMS = &queryDuration
	}
	if qOutOK {
		input.OutputSize = &queryOutput
	}
	if qSucOK {
		input.Success = &querySuccess
	}
	if qStateOK {
		input.State = queryState
	}
	if qMsgOK {
		input.Message = queryMsg
	}
	if qEnvOK {
		input.Environment = queryEnv
	}
	if qMetricOK {
		input.Metric = queryMetric
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
	if hStateOK {
		input.State = headState
	}
	if hMsgOK {
		input.Message = headMsg
	}
	if hEnvOK {
		input.Environment = headEnv
	}
	if hMetricOK {
		input.Metric = headMetric
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
		if payload.State != nil {
			state, ok, err := readState(*payload.State)
			if err != nil {
				return input, fmt.Errorf("invalid JSON state: %w", err)
			}
			if ok {
				input.State = state
			}
		}
		if payload.Msg != nil {
			if msg, ok := readString(*payload.Msg); ok {
				input.Message = msg
			}
		}
		if payload.Env != nil {
			if env, ok := readString(*payload.Env); ok {
				input.Environment = env
			}
		}
		if payload.Metric != nil {
			if metric, ok := readString(*payload.Metric); ok {
				input.Metric = metric
			}
		}
	}

	applyStateDefaults(&input)
	if err := validateInputBounds(input); err != nil {
		return input, err
	}
	input.Message = truncate(input.Message, maxMessageChars)
	input.Environment = truncate(input.Environment, maxEnvironmentChars)
	input.Metric = truncate(input.Metric, maxMetricChars)
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

func readString(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}

func readState(raw string) (string, bool, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return "", false, nil
	}

	switch trimmed {
	case "ok", "complete", "success", "pass":
		return "ok", true, nil
	case "run", "start":
		return "run", true, nil
	case "fail", "failed", "error":
		return "fail", true, nil
	default:
		return "", false, fmt.Errorf("unsupported state %q", raw)
	}
}

func applyStateDefaults(input *domain.PingInput) {
	switch input.State {
	case "run":
		input.EventType = "start"
	case "fail":
		input.EventType = "fail"
		v := false
		input.Success = &v
	case "ok":
		input.EventType = "ping"
		if input.Success == nil {
			v := true
			input.Success = &v
		}
	default:
		input.EventType = "ping"
	}
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
