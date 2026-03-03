package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"pulse/backend/internal/domain"
)

type CheckPatch struct {
	Name                    *string
	ExpectedIntervalSeconds *int
	GraceSeconds            *int
	ScheduleMode            *string
}

type CheckRecord struct {
	ID                      string             `json:"id"`
	UserID                  string             `json:"user_id"`
	Name                    string             `json:"name"`
	Token                   string             `json:"token"`
	ExpectedIntervalSeconds int                `json:"expected_interval_seconds"`
	GraceSeconds            int                `json:"grace_seconds"`
	ScheduleMode            string             `json:"schedule_mode"`
	IntervalSampleCount     int                `json:"interval_sample_count"`
	Status                  domain.CheckStatus `json:"status"`
	LastPingAt              *time.Time         `json:"last_ping_at,omitempty"`
	NextDueAt               *time.Time         `json:"next_due_at,omitempty"`
	CreatedAt               time.Time          `json:"created_at"`
	UpdatedAt               time.Time          `json:"updated_at"`
}

type EventRecord struct {
	ID         int64   `json:"id"`
	Type       string  `json:"type"`
	ReceivedAt string  `json:"received_at"`
	DurationMS *int    `json:"duration_ms,omitempty"`
	OutputSize *int    `json:"output_size,omitempty"`
	Success    *bool   `json:"success,omitempty"`
	MetaJSON   *string `json:"meta_json,omitempty"`
}

type AlertChannelRecord struct {
	ID        string                  `json:"id"`
	Type      domain.AlertChannelType `json:"type"`
	Target    string                  `json:"target"`
	Enabled   bool                    `json:"enabled"`
	CreatedAt time.Time               `json:"created_at"`
}

func (s *Store) EnsureUser(ctx context.Context, userID, email string) error {
	if email == "" {
		email = fmt.Sprintf("%s@placeholder.local", userID)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users(id, email)
		VALUES ($1::uuid, $2)
		ON CONFLICT (id) DO UPDATE SET email = EXCLUDED.email
	`, userID, email)
	if err != nil {
		return fmt.Errorf("ensure user: %w", err)
	}
	return nil
}

func (s *Store) CreateCheck(ctx context.Context, userID, name, token string, expectedIntervalSeconds, graceSeconds int, scheduleMode string) (CheckRecord, error) {
	var rec CheckRecord
	if err := s.db.QueryRowContext(ctx, `
		INSERT INTO checks(user_id, name, token, expected_interval_seconds, grace_seconds, schedule_mode)
		VALUES ($1::uuid, $2, $3, $4, $5, $6)
		RETURNING id, user_id, name, token, expected_interval_seconds, grace_seconds, schedule_mode, interval_sample_count,
			status, last_ping_at, next_due_at, created_at, updated_at
	`, userID, name, token, expectedIntervalSeconds, graceSeconds, scheduleMode).Scan(
		&rec.ID,
		&rec.UserID,
		&rec.Name,
		&rec.Token,
		&rec.ExpectedIntervalSeconds,
		&rec.GraceSeconds,
		&rec.ScheduleMode,
		&rec.IntervalSampleCount,
		&rec.Status,
		&rec.LastPingAt,
		&rec.NextDueAt,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	); err != nil {
		return CheckRecord{}, fmt.Errorf("create check: %w", err)
	}
	return rec, nil
}

func (s *Store) ListChecks(ctx context.Context, userID string, limit, offset int) ([]CheckRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, name, token, expected_interval_seconds, grace_seconds, schedule_mode, interval_sample_count,
			status, last_ping_at, next_due_at, created_at, updated_at
		FROM checks
		WHERE user_id = $1::uuid AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list checks: %w", err)
	}
	defer rows.Close()

	checks := make([]CheckRecord, 0)
	for rows.Next() {
		var rec CheckRecord
		if err := rows.Scan(
			&rec.ID,
			&rec.UserID,
			&rec.Name,
			&rec.Token,
			&rec.ExpectedIntervalSeconds,
			&rec.GraceSeconds,
			&rec.ScheduleMode,
			&rec.IntervalSampleCount,
			&rec.Status,
			&rec.LastPingAt,
			&rec.NextDueAt,
			&rec.CreatedAt,
			&rec.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan check: %w", err)
		}
		checks = append(checks, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate checks: %w", err)
	}
	return checks, nil
}

func (s *Store) GetCheck(ctx context.Context, userID, checkID string) (CheckRecord, error) {
	var rec CheckRecord
	if err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, name, token, expected_interval_seconds, grace_seconds, schedule_mode, interval_sample_count,
			status, last_ping_at, next_due_at, created_at, updated_at
		FROM checks
		WHERE id = $1::uuid AND user_id = $2::uuid AND deleted_at IS NULL
	`, checkID, userID).Scan(
		&rec.ID,
		&rec.UserID,
		&rec.Name,
		&rec.Token,
		&rec.ExpectedIntervalSeconds,
		&rec.GraceSeconds,
		&rec.ScheduleMode,
		&rec.IntervalSampleCount,
		&rec.Status,
		&rec.LastPingAt,
		&rec.NextDueAt,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return CheckRecord{}, domain.ErrNotFound
		}
		return CheckRecord{}, fmt.Errorf("get check: %w", err)
	}
	return rec, nil
}

func (s *Store) UpdateCheck(ctx context.Context, userID, checkID string, patch CheckPatch) (CheckRecord, error) {
	sets := []string{"updated_at = now()"}
	args := []any{checkID, userID}
	argIdx := 3

	if patch.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, strings.TrimSpace(*patch.Name))
		argIdx++
	}
	if patch.ExpectedIntervalSeconds != nil {
		sets = append(sets, fmt.Sprintf("expected_interval_seconds = $%d", argIdx))
		args = append(args, *patch.ExpectedIntervalSeconds)
		argIdx++
	}
	if patch.GraceSeconds != nil {
		sets = append(sets, fmt.Sprintf("grace_seconds = $%d", argIdx))
		args = append(args, *patch.GraceSeconds)
		argIdx++
	}
	if patch.ScheduleMode != nil {
		sets = append(sets, fmt.Sprintf("schedule_mode = $%d", argIdx))
		args = append(args, *patch.ScheduleMode)
		argIdx++
	}

	query := fmt.Sprintf(`
		UPDATE checks
		SET %s
		WHERE id = $1::uuid AND user_id = $2::uuid AND deleted_at IS NULL
		RETURNING id, user_id, name, token, expected_interval_seconds, grace_seconds, schedule_mode, interval_sample_count,
			status, last_ping_at, next_due_at, created_at, updated_at
	`, strings.Join(sets, ", "))

	var rec CheckRecord
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&rec.ID,
		&rec.UserID,
		&rec.Name,
		&rec.Token,
		&rec.ExpectedIntervalSeconds,
		&rec.GraceSeconds,
		&rec.ScheduleMode,
		&rec.IntervalSampleCount,
		&rec.Status,
		&rec.LastPingAt,
		&rec.NextDueAt,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return CheckRecord{}, domain.ErrNotFound
		}
		return CheckRecord{}, fmt.Errorf("update check: %w", err)
	}
	return rec, nil
}

func (s *Store) PauseCheck(ctx context.Context, userID, checkID string) (CheckRecord, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return CheckRecord{}, fmt.Errorf("begin pause tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE incidents
		SET status = 'resolved', resolved_at = now()
		WHERE check_id = $1::uuid AND status = 'open'
	`, checkID); err != nil {
		return CheckRecord{}, fmt.Errorf("resolve incident on pause: %w", err)
	}

	var rec CheckRecord
	if err := tx.QueryRowContext(ctx, `
		UPDATE checks
		SET status = 'paused', updated_at = now()
		WHERE id = $1::uuid AND user_id = $2::uuid AND deleted_at IS NULL
		RETURNING id, user_id, name, token, expected_interval_seconds, grace_seconds, schedule_mode, interval_sample_count,
			status, last_ping_at, next_due_at, created_at, updated_at
	`, checkID, userID).Scan(
		&rec.ID,
		&rec.UserID,
		&rec.Name,
		&rec.Token,
		&rec.ExpectedIntervalSeconds,
		&rec.GraceSeconds,
		&rec.ScheduleMode,
		&rec.IntervalSampleCount,
		&rec.Status,
		&rec.LastPingAt,
		&rec.NextDueAt,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return CheckRecord{}, domain.ErrNotFound
		}
		return CheckRecord{}, fmt.Errorf("pause check: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return CheckRecord{}, fmt.Errorf("commit pause tx: %w", err)
	}
	return rec, nil
}

func (s *Store) ResumeCheck(ctx context.Context, userID, checkID string) (CheckRecord, error) {
	var rec CheckRecord
	if err := s.db.QueryRowContext(ctx, `
		UPDATE checks
		SET status = 'up',
			next_due_at = now() + make_interval(secs => expected_interval_seconds),
			updated_at = now()
		WHERE id = $1::uuid AND user_id = $2::uuid AND deleted_at IS NULL
		RETURNING id, user_id, name, token, expected_interval_seconds, grace_seconds, schedule_mode, interval_sample_count,
			status, last_ping_at, next_due_at, created_at, updated_at
	`, checkID, userID).Scan(
		&rec.ID,
		&rec.UserID,
		&rec.Name,
		&rec.Token,
		&rec.ExpectedIntervalSeconds,
		&rec.GraceSeconds,
		&rec.ScheduleMode,
		&rec.IntervalSampleCount,
		&rec.Status,
		&rec.LastPingAt,
		&rec.NextDueAt,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return CheckRecord{}, domain.ErrNotFound
		}
		return CheckRecord{}, fmt.Errorf("resume check: %w", err)
	}
	return rec, nil
}

func (s *Store) RotateCheckToken(ctx context.Context, userID, checkID, token string) (CheckRecord, error) {
	var rec CheckRecord
	if err := s.db.QueryRowContext(ctx, `
		UPDATE checks
		SET token = $3, updated_at = now()
		WHERE id = $1::uuid AND user_id = $2::uuid AND deleted_at IS NULL
		RETURNING id, user_id, name, token, expected_interval_seconds, grace_seconds, schedule_mode, interval_sample_count,
			status, last_ping_at, next_due_at, created_at, updated_at
	`, checkID, userID, token).Scan(
		&rec.ID,
		&rec.UserID,
		&rec.Name,
		&rec.Token,
		&rec.ExpectedIntervalSeconds,
		&rec.GraceSeconds,
		&rec.ScheduleMode,
		&rec.IntervalSampleCount,
		&rec.Status,
		&rec.LastPingAt,
		&rec.NextDueAt,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return CheckRecord{}, domain.ErrNotFound
		}
		return CheckRecord{}, fmt.Errorf("rotate check token: %w", err)
	}
	return rec, nil
}

func (s *Store) ListEvents(ctx context.Context, userID, checkID string, limit int) ([]EventRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT e.id, e.type, e.received_at, e.duration_ms, e.output_size, e.success, e.meta_json
		FROM events e
		JOIN checks c ON c.id = e.check_id
		WHERE c.id = $1::uuid AND c.user_id = $2::uuid AND c.deleted_at IS NULL
		ORDER BY e.received_at DESC
		LIMIT $3
	`, checkID, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	events := make([]EventRecord, 0, limit)
	for rows.Next() {
		var rec EventRecord
		var receivedAt time.Time
		var metaRaw []byte
		if err := rows.Scan(&rec.ID, &rec.Type, &receivedAt, &rec.DurationMS, &rec.OutputSize, &rec.Success, &metaRaw); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		t := receivedAt.UTC().Format(time.RFC3339)
		rec.ReceivedAt = t
		if len(metaRaw) > 0 {
			meta := string(metaRaw)
			rec.MetaJSON = &meta
		}
		events = append(events, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}
	return events, nil
}

func (s *Store) CreateAlertChannel(ctx context.Context, userID string, channelType domain.AlertChannelType, target string) (AlertChannelRecord, error) {
	var rec AlertChannelRecord
	if err := s.db.QueryRowContext(ctx, `
		INSERT INTO alert_channels(user_id, type, target, enabled)
		VALUES ($1::uuid, $2, $3, true)
		RETURNING id, type, target, enabled, created_at
	`, userID, channelType, target).Scan(&rec.ID, &rec.Type, &rec.Target, &rec.Enabled, &rec.CreatedAt); err != nil {
		return AlertChannelRecord{}, fmt.Errorf("create alert channel: %w", err)
	}
	return rec, nil
}

func (s *Store) ListAlertChannels(ctx context.Context, userID string, limit, offset int) ([]AlertChannelRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, type, target, enabled, created_at
		FROM alert_channels
		WHERE user_id = $1::uuid
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list alert channels: %w", err)
	}
	defer rows.Close()

	channels := make([]AlertChannelRecord, 0)
	for rows.Next() {
		var rec AlertChannelRecord
		if err := rows.Scan(&rec.ID, &rec.Type, &rec.Target, &rec.Enabled, &rec.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan alert channel: %w", err)
		}
		channels = append(channels, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate alert channels: %w", err)
	}
	return channels, nil
}

func (s *Store) UpdateAlertChannel(ctx context.Context, userID, channelID string, enabled *bool, target *string) (AlertChannelRecord, error) {
	sets := []string{"id = id"}
	args := []any{channelID, userID}
	argIdx := 3
	if enabled != nil {
		sets = append(sets, fmt.Sprintf("enabled = $%d", argIdx))
		args = append(args, *enabled)
		argIdx++
	}
	if target != nil {
		sets = append(sets, fmt.Sprintf("target = $%d", argIdx))
		args = append(args, strings.TrimSpace(*target))
		argIdx++
	}
	query := fmt.Sprintf(`
		UPDATE alert_channels
		SET %s
		WHERE id = $1::uuid AND user_id = $2::uuid
		RETURNING id, type, target, enabled, created_at
	`, strings.Join(sets, ", "))

	var rec AlertChannelRecord
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&rec.ID, &rec.Type, &rec.Target, &rec.Enabled, &rec.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return AlertChannelRecord{}, domain.ErrNotFound
		}
		return AlertChannelRecord{}, fmt.Errorf("update alert channel: %w", err)
	}
	return rec, nil
}
