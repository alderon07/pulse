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
