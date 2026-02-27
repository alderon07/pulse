package domain

import (
	"errors"
	"time"
)

type CheckStatus string

const (
	StatusUp     CheckStatus = "up"
	StatusLate   CheckStatus = "late"
	StatusDown   CheckStatus = "down"
	StatusPaused CheckStatus = "paused"
)

type DeliveryKind string

const (
	DeliveryDown      DeliveryKind = "down"
	DeliveryRepeat    DeliveryKind = "repeat"
	DeliveryRecovered DeliveryKind = "recovered"
	DeliveryWarning   DeliveryKind = "warning"
)

type AlertChannelType string

const (
	AlertChannelEmail   AlertChannelType = "email"
	AlertChannelWebhook AlertChannelType = "webhook"
)

type PingInput struct {
	DurationMS     *int   `json:"duration_ms"`
	OutputSize     *int   `json:"output_size"`
	Success        *bool  `json:"success"`
	IdempotencyKey string `json:"-"`
	SourceIP       string `json:"-"`
	UserAgent      string `json:"-"`
}

type PingResult struct {
	Status     CheckStatus `json:"status"`
	NextDueAt  time.Time   `json:"next_due_at"`
	Idempotent bool        `json:"idempotent"`
}

type Check struct {
	ID                      string
	UserID                  string
	Token                   string
	ExpectedIntervalSeconds int
	GraceSeconds            int
	Status                  CheckStatus
	LastPingAt              *time.Time
	NextDueAt               *time.Time
	LastAlertedAt           *time.Time
	DeletedAt               *time.Time
}

type AlertDelivery struct {
	ID           int64
	IncidentID   *string
	CheckID      string
	ChannelID    string
	Kind         DeliveryKind
	Status       string
	AttemptCount int
	ChannelType  AlertChannelType
	Target       string
}

var (
	ErrNotFound    = errors.New("not found")
	ErrInvalidData = errors.New("invalid data")
)

func ComputeStatus(now, nextDueAt time.Time, graceSeconds int, paused bool) CheckStatus {
	if paused {
		return StatusPaused
	}
	graceDeadline := nextDueAt.Add(time.Duration(graceSeconds) * time.Second)
	if !now.After(nextDueAt) {
		return StatusUp
	}
	if !now.After(graceDeadline) {
		return StatusLate
	}
	return StatusDown
}
