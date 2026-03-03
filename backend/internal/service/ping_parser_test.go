package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParsePingInputPrecedence(t *testing.T) {
	body := `{"duration_ms": 300, "output_size": 500, "success": true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/ping/token?duration_ms=100&output_size=200&success=false", strings.NewReader(body))
	req.Header.Set("X-Duration-MS", "150")
	req.Header.Set("X-Output-Size", "250")
	req.Header.Set("X-Success", "false")
	req.Header.Set("Idempotency-Key", "abc")
	req.RemoteAddr = "10.0.0.1:1234"

	in, err := ParsePingInput(req)
	if err != nil {
		t.Fatalf("ParsePingInput error: %v", err)
	}
	if in.DurationMS == nil || *in.DurationMS != 300 {
		t.Fatalf("expected duration from JSON body, got %#v", in.DurationMS)
	}
	if in.OutputSize == nil || *in.OutputSize != 500 {
		t.Fatalf("expected output_size from JSON body, got %#v", in.OutputSize)
	}
	if in.Success == nil || !*in.Success {
		t.Fatalf("expected success=true from JSON body, got %#v", in.Success)
	}
	if in.IdempotencyKey != "abc" {
		t.Fatalf("expected idempotency key, got %q", in.IdempotencyKey)
	}
}

func TestParsePingInputNegativeValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/ping/token?duration_ms=-1", nil)
	_, err := ParsePingInput(req)
	if err == nil {
		t.Fatal("expected validation error for negative duration")
	}
}

func TestParsePingInputStateAndTelemetryPrecedence(t *testing.T) {
	body := `{"state":"ok","msg":"body-msg","env":"body-env","metric":"count:300"}`
	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/ping/token?state=run&msg=query-msg&env=query-env&metric=count:100",
		strings.NewReader(body),
	)
	req.Header.Set("X-State", "fail")
	req.Header.Set("X-Msg", "header-msg")
	req.Header.Set("X-Env", "header-env")
	req.Header.Set("X-Metric", "count:200")

	in, err := ParsePingInput(req)
	if err != nil {
		t.Fatalf("ParsePingInput error: %v", err)
	}

	if in.State != "ok" {
		t.Fatalf("expected state from JSON body, got %q", in.State)
	}
	if in.EventType != "ping" {
		t.Fatalf("expected ping event type for ok state, got %q", in.EventType)
	}
	if in.Success == nil || !*in.Success {
		t.Fatalf("expected success=true default for ok state, got %#v", in.Success)
	}
	if in.Message != "body-msg" {
		t.Fatalf("expected msg from JSON body, got %q", in.Message)
	}
	if in.Environment != "body-env" {
		t.Fatalf("expected env from JSON body, got %q", in.Environment)
	}
	if in.Metric != "count:300" {
		t.Fatalf("expected metric from JSON body, got %q", in.Metric)
	}
}

func TestParsePingInputStateAliases(t *testing.T) {
	cases := []struct {
		name      string
		rawState  string
		wantState string
		wantType  string
		wantSucc  *bool
	}{
		{
			name:      "complete alias maps to ok",
			rawState:  "complete",
			wantState: "ok",
			wantType:  "ping",
			wantSucc:  boolPtr(true),
		},
		{
			name:      "run alias maps to start event",
			rawState:  "start",
			wantState: "run",
			wantType:  "start",
			wantSucc:  nil,
		},
		{
			name:      "error alias maps to fail and forces false",
			rawState:  "error",
			wantState: "fail",
			wantType:  "fail",
			wantSucc:  boolPtr(false),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/ping/token?state="+tc.rawState, nil)
			in, err := ParsePingInput(req)
			if err != nil {
				t.Fatalf("ParsePingInput error: %v", err)
			}
			if in.State != tc.wantState {
				t.Fatalf("expected state %q, got %q", tc.wantState, in.State)
			}
			if in.EventType != tc.wantType {
				t.Fatalf("expected event type %q, got %q", tc.wantType, in.EventType)
			}
			if !equalBoolPtr(in.Success, tc.wantSucc) {
				t.Fatalf("expected success %#v, got %#v", tc.wantSucc, in.Success)
			}
		})
	}
}

func TestParsePingInputFailStateOverridesSuccess(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/ping/token?state=fail&success=true", nil)
	in, err := ParsePingInput(req)
	if err != nil {
		t.Fatalf("ParsePingInput error: %v", err)
	}
	if in.EventType != "fail" {
		t.Fatalf("expected fail event type, got %q", in.EventType)
	}
	if in.Success == nil || *in.Success {
		t.Fatalf("expected forced success=false, got %#v", in.Success)
	}
}

func TestParsePingInputInvalidState(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/ping/token?state=weird", nil)
	_, err := ParsePingInput(req)
	if err == nil {
		t.Fatal("expected invalid state error")
	}
}

func TestParsePingInputTruncatesTelemetryFields(t *testing.T) {
	msg := strings.Repeat("m", maxMessageChars+100)
	env := strings.Repeat("e", maxEnvironmentChars+100)
	metric := strings.Repeat("c", maxMetricChars+100)
	req := httptest.NewRequest(http.MethodGet, "/v1/ping/token?msg="+msg+"&env="+env+"&metric="+metric, nil)

	in, err := ParsePingInput(req)
	if err != nil {
		t.Fatalf("ParsePingInput error: %v", err)
	}
	if len(in.Message) != maxMessageChars {
		t.Fatalf("expected truncated msg length=%d, got %d", maxMessageChars, len(in.Message))
	}
	if len(in.Environment) != maxEnvironmentChars {
		t.Fatalf("expected truncated env length=%d, got %d", maxEnvironmentChars, len(in.Environment))
	}
	if len(in.Metric) != maxMetricChars {
		t.Fatalf("expected truncated metric length=%d, got %d", maxMetricChars, len(in.Metric))
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func equalBoolPtr(a, b *bool) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	default:
		return *a == *b
	}
}
