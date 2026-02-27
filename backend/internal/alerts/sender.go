package alerts

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"

	"pulse/backend/internal/domain"
)

type Sender interface {
	Send(ctx context.Context, delivery domain.AlertDelivery) error
}

type MultiSender struct {
	email   *EmailSender
	webhook *WebhookSender
}

func NewMultiSender(email *EmailSender, webhook *WebhookSender) *MultiSender {
	return &MultiSender{email: email, webhook: webhook}
}

func (m *MultiSender) Send(ctx context.Context, delivery domain.AlertDelivery) error {
	switch delivery.ChannelType {
	case domain.AlertChannelEmail:
		return m.email.Send(ctx, delivery)
	case domain.AlertChannelWebhook:
		return m.webhook.Send(ctx, delivery)
	default:
		return fmt.Errorf("unsupported alert channel type: %s", delivery.ChannelType)
	}
}

type EmailSender struct {
	logger *slog.Logger
}

func NewEmailSender(logger *slog.Logger) *EmailSender {
	return &EmailSender{logger: logger}
}

func (s *EmailSender) Send(_ context.Context, delivery domain.AlertDelivery) error {
	// Placeholder for provider integration. Keep payload out of logs to avoid leaking PII.
	s.logger.Info("email delivery accepted", "delivery_id", delivery.ID, "kind", delivery.Kind)
	return nil
}

type WebhookSender struct {
	client  *http.Client
	logger  *slog.Logger
	hmacKey string
	breaker *circuitBreaker
}

func NewWebhookSender(timeout time.Duration, hmacKey string, logger *slog.Logger) *WebhookSender {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &WebhookSender{
		client:  &http.Client{Timeout: timeout},
		logger:  logger,
		hmacKey: hmacKey,
		breaker: newCircuitBreaker(5, 2*time.Minute),
	}
}

func (s *WebhookSender) Send(ctx context.Context, delivery domain.AlertDelivery) error {
	if !s.breaker.Allow(delivery.ChannelID) {
		return errors.New("circuit breaker open")
	}
	if err := validateWebhookTarget(delivery.Target); err != nil {
		s.breaker.ReportFailure(delivery.ChannelID)
		return err
	}

	payload := map[string]any{
		"schema_version": "v1",
		"delivery_id":    delivery.ID,
		"incident_id":    delivery.IncidentID,
		"check_id":       delivery.CheckID,
		"kind":           delivery.Kind,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		s.breaker.ReportFailure(delivery.ChannelID)
		return fmt.Errorf("marshal webhook payload: %w", err)
	}
	if len(body) > 4096 {
		s.breaker.ReportFailure(delivery.ChannelID)
		return errors.New("webhook payload too large")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, delivery.Target, bytes.NewReader(body))
	if err != nil {
		s.breaker.ReportFailure(delivery.ChannelID)
		return fmt.Errorf("new webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.hmacKey != "" {
		req.Header.Set("X-Pulse-Signature", signBody(s.hmacKey, body))
	}

	resp, err := s.client.Do(req)
	if err != nil {
		s.breaker.ReportFailure(delivery.ChannelID)
		return fmt.Errorf("send webhook request: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		s.breaker.ReportSuccess(delivery.ChannelID)
		return nil
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		s.breaker.ReportFailure(delivery.ChannelID)
		return fmt.Errorf("retryable webhook status: %d", resp.StatusCode)
	}

	s.breaker.ReportFailure(delivery.ChannelID)
	return nonRetryableError{cause: fmt.Errorf("non-retryable webhook status: %d", resp.StatusCode)}
}

type nonRetryableError struct {
	cause error
}

func (e nonRetryableError) Error() string { return e.cause.Error() }

func IsNonRetryable(err error) bool {
	var target nonRetryableError
	return errors.As(err, &target)
}

func signBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func validateWebhookTarget(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid webhook url: %w", err)
	}
	if u.Scheme != "https" {
		return errors.New("webhook URL must use https")
	}
	host := u.Hostname()
	if host == "" {
		return errors.New("webhook URL host is required")
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("resolve webhook host: %w", err)
	}
	for _, ip := range ips {
		addr, ok := netip.AddrFromSlice(ip)
		if !ok {
			continue
		}
		if isPrivateOrLocalAddress(addr) {
			return fmt.Errorf("webhook host resolves to private/local address: %s", addr.String())
		}
	}
	return nil
}

func isPrivateOrLocalAddress(addr netip.Addr) bool {
	if addr.IsLoopback() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsPrivate() {
		return true
	}
	if addr.Is4() {
		s := addr.String()
		return strings.HasPrefix(s, "169.254.")
	}
	return false
}

type breakerState struct {
	consecutiveFailures int
	openedAt            time.Time
}

type circuitBreaker struct {
	mu        sync.Mutex
	states    map[string]breakerState
	openAfter int
	coolDown  time.Duration
}

func newCircuitBreaker(openAfter int, coolDown time.Duration) *circuitBreaker {
	if openAfter <= 0 {
		openAfter = 5
	}
	if coolDown <= 0 {
		coolDown = 2 * time.Minute
	}
	return &circuitBreaker{
		states:    make(map[string]breakerState),
		openAfter: openAfter,
		coolDown:  coolDown,
	}
}

func (c *circuitBreaker) Allow(channelID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	state, ok := c.states[channelID]
	if !ok {
		return true
	}
	if state.consecutiveFailures < c.openAfter {
		return true
	}
	if time.Since(state.openedAt) >= c.coolDown {
		state.consecutiveFailures = c.openAfter - 1
		state.openedAt = time.Time{}
		c.states[channelID] = state
		return true
	}
	return false
}

func (c *circuitBreaker) ReportFailure(channelID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.states[channelID]
	state.consecutiveFailures++
	if state.consecutiveFailures >= c.openAfter && state.openedAt.IsZero() {
		state.openedAt = time.Now()
	}
	c.states[channelID] = state
}

func (c *circuitBreaker) ReportSuccess(channelID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.states, channelID)
}
