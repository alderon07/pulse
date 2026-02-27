package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"pulse/backend/internal/domain"
)

const defaultEWMAAlpha = 0.2

type Store struct {
	db        *sql.DB
	ewmaAlpha float64
}

func New(db *sql.DB) *Store {
	return &Store{db: db, ewmaAlpha: defaultEWMAAlpha}
}

type checkRow struct {
	ID                  string
	UserID              string
	Status              string
	ExpectedIntervalSec int
	GraceSec            int
	BaselineDuration    sql.NullFloat64
	BaselineOutput      sql.NullFloat64
	SampleDuration      int
	SampleOutput        int
}

func (s *Store) RecordPing(ctx context.Context, token string, input domain.PingInput, idempotencyTTL time.Duration) (domain.PingResult, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return domain.PingResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	row := checkRow{}
	if err := tx.QueryRowContext(ctx, `
		SELECT id, user_id, status, expected_interval_seconds, grace_seconds,
		       baseline_duration_ms, baseline_output_size,
		       sample_count_duration, sample_count_output
		FROM checks
		WHERE token = $1 AND deleted_at IS NULL
		FOR UPDATE
	`, token).Scan(
		&row.ID,
		&row.UserID,
		&row.Status,
		&row.ExpectedIntervalSec,
		&row.GraceSec,
		&row.BaselineDuration,
		&row.BaselineOutput,
		&row.SampleDuration,
		&row.SampleOutput,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.PingResult{}, domain.ErrNotFound
		}
		return domain.PingResult{}, fmt.Errorf("select check by token: %w", err)
	}

	if input.IdempotencyKey != "" {
		var status string
		var nextDue time.Time
		if err := tx.QueryRowContext(ctx, `
			SELECT response_status, response_next_due_at
			FROM ping_idempotency
			WHERE check_id = $1 AND idempotency_key = $2 AND expires_at > now()
		`, row.ID, input.IdempotencyKey).Scan(&status, &nextDue); err == nil {
			if err := tx.Commit(); err != nil {
				return domain.PingResult{}, fmt.Errorf("commit idempotent tx: %w", err)
			}
			return domain.PingResult{Status: domain.CheckStatus(status), NextDueAt: nextDue, Idempotent: true}, nil
		} else if !errors.Is(err, sql.ErrNoRows) {
			return domain.PingResult{}, fmt.Errorf("select idempotency: %w", err)
		}
	}

	meta := map[string]string{
		"source_ip":  truncateString(input.SourceIP, 64),
		"user_agent": truncateString(input.UserAgent, 512),
	}
	if input.IdempotencyKey != "" {
		meta["idempotency_key"] = truncateString(input.IdempotencyKey, 128)
	}
	metaJSON, _ := json.Marshal(meta)

	var eventID int64
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO events (check_id, type, received_at, duration_ms, output_size, success, meta_json)
		VALUES ($1, 'ping', now(), $2, $3, $4, $5)
		RETURNING id
	`, row.ID, input.DurationMS, input.OutputSize, input.Success, metaJSON).Scan(&eventID); err != nil {
		return domain.PingResult{}, fmt.Errorf("insert ping event: %w", err)
	}

	nextDurationBaseline, durationSamples := updatedBaseline(row.BaselineDuration, row.SampleDuration, input.DurationMS, s.ewmaAlpha)
	nextOutputBaseline, outputSamples := updatedBaseline(row.BaselineOutput, row.SampleOutput, input.OutputSize, s.ewmaAlpha)

	var status string
	var nextDue time.Time
	if err := tx.QueryRowContext(ctx, `
		UPDATE checks
		SET
			last_ping_at = now(),
			next_due_at = now() + make_interval(secs => expected_interval_seconds),
			status = CASE WHEN status = 'paused' THEN status ELSE 'up' END,
			last_duration_ms = COALESCE($2, last_duration_ms),
			baseline_duration_ms = $3,
			sample_count_duration = $4,
			last_output_size = COALESCE($5, last_output_size),
			baseline_output_size = $6,
			sample_count_output = $7,
			last_success = COALESCE($8, last_success),
			updated_at = now()
		WHERE id = $1
		RETURNING status, next_due_at
	`, row.ID, input.DurationMS, nextDurationBaseline, durationSamples, input.OutputSize, nextOutputBaseline, outputSamples, input.Success).Scan(&status, &nextDue); err != nil {
		return domain.PingResult{}, fmt.Errorf("update check for ping: %w", err)
	}

	var incidentID string
	if err := tx.QueryRowContext(ctx, `
		SELECT id FROM incidents WHERE check_id = $1 AND status = 'open' FOR UPDATE
	`, row.ID).Scan(&incidentID); err == nil {
		if _, err := tx.ExecContext(ctx, `
			UPDATE incidents
			SET status = 'resolved', resolved_at = now(), resolve_event_id = $2
			WHERE id = $1
		`, incidentID, eventID); err != nil {
			return domain.PingResult{}, fmt.Errorf("resolve incident: %w", err)
		}
		if err := s.enqueueAlertsForAllChannels(ctx, tx, row.UserID, row.ID, &incidentID, domain.DeliveryRecovered, 0); err != nil {
			return domain.PingResult{}, err
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return domain.PingResult{}, fmt.Errorf("select open incident: %w", err)
	}

	if input.IdempotencyKey != "" {
		expiresAt := time.Now().Add(idempotencyTTL)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO ping_idempotency(check_id, idempotency_key, response_status, response_next_due_at, expires_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (check_id, idempotency_key) DO UPDATE
			SET response_status = excluded.response_status,
				response_next_due_at = excluded.response_next_due_at,
				expires_at = excluded.expires_at
		`, row.ID, input.IdempotencyKey, status, nextDue, expiresAt); err != nil {
			return domain.PingResult{}, fmt.Errorf("upsert idempotency key: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return domain.PingResult{}, fmt.Errorf("commit ping tx: %w", err)
	}
	return domain.PingResult{Status: domain.CheckStatus(status), NextDueAt: nextDue}, nil
}

func (s *Store) ProcessOverdueBatch(ctx context.Context, batchSize int, repeatInterval time.Duration) (int, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return 0, fmt.Errorf("begin overdue tx: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT id, user_id, status, next_due_at, grace_seconds, last_alerted_at
		FROM checks
		WHERE deleted_at IS NULL
			AND status <> 'paused'
			AND next_due_at IS NOT NULL
			AND now() > next_due_at
		ORDER BY next_due_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, batchSize)
	if err != nil {
		return 0, fmt.Errorf("select due checks: %w", err)
	}
	defer rows.Close()

	type rowData struct {
		checkID      string
		userID       string
		status       string
		nextDueAt    time.Time
		graceSeconds int
		lastAlerted  sql.NullTime
	}
	var dueRows []rowData
	for rows.Next() {
		var r rowData
		if err := rows.Scan(&r.checkID, &r.userID, &r.status, &r.nextDueAt, &r.graceSeconds, &r.lastAlerted); err != nil {
			return 0, fmt.Errorf("scan due checks: %w", err)
		}
		dueRows = append(dueRows, r)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate due checks: %w", err)
	}

	now := time.Now().UTC()
	updated := 0
	for _, r := range dueRows {
		if now.After(r.nextDueAt.Add(time.Duration(r.graceSeconds) * time.Second)) {
			kind := domain.DeliveryDown
			shouldAlert := false
			if r.status != string(domain.StatusDown) {
				shouldAlert = true
				updated++
			} else if !r.lastAlerted.Valid || now.After(r.lastAlerted.Time.Add(repeatInterval)) {
				kind = domain.DeliveryRepeat
				shouldAlert = true
			}

			var incidentID string
			if err := tx.QueryRowContext(ctx, `
				INSERT INTO incidents(id, check_id, status, opened_at)
				VALUES (gen_random_uuid(), $1, 'open', now())
				ON CONFLICT DO NOTHING
				RETURNING id
			`, r.checkID).Scan(&incidentID); err != nil {
				if !errors.Is(err, sql.ErrNoRows) {
					return updated, fmt.Errorf("insert open incident: %w", err)
				}
				if err := tx.QueryRowContext(ctx, `
					SELECT id FROM incidents WHERE check_id = $1 AND status = 'open'
				`, r.checkID).Scan(&incidentID); err != nil {
					return updated, fmt.Errorf("select open incident fallback: %w", err)
				}
			}

			if _, err := tx.ExecContext(ctx, `
				UPDATE checks
				SET status = 'down',
					last_alerted_at = CASE WHEN $2 THEN now() ELSE last_alerted_at END,
					updated_at = now()
				WHERE id = $1
			`, r.checkID, shouldAlert); err != nil {
				return updated, fmt.Errorf("update check down: %w", err)
			}

			if shouldAlert {
				priority := 0
				if kind == domain.DeliveryRepeat {
					priority = 1
				}
				if err := s.enqueueAlertsForAllChannels(ctx, tx, r.userID, r.checkID, &incidentID, kind, priority); err != nil {
					return updated, err
				}
			}
			continue
		}

		if r.status == string(domain.StatusUp) {
			if _, err := tx.ExecContext(ctx, `
				UPDATE checks
				SET status = 'late', updated_at = now()
				WHERE id = $1
			`, r.checkID); err != nil {
				return updated, fmt.Errorf("update check late: %w", err)
			}
			updated++
		}
	}

	if err := tx.Commit(); err != nil {
		return updated, fmt.Errorf("commit overdue tx: %w", err)
	}
	return updated, nil
}

func (s *Store) ClaimDeliveries(ctx context.Context, batchSize int) ([]domain.AlertDelivery, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("begin claim deliveries tx: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT d.id, d.incident_id, d.check_id, d.channel_id, d.kind, d.status, d.attempt_count,
		       c.type, c.target
		FROM alert_deliveries d
		JOIN alert_channels c ON c.id = d.channel_id
		WHERE d.status IN ('queued', 'failed')
			AND d.next_attempt_at <= now()
			AND c.enabled = true
		ORDER BY d.priority ASC, d.created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, batchSize)
	if err != nil {
		return nil, fmt.Errorf("select deliveries: %w", err)
	}
	defer rows.Close()

	deliveries := make([]domain.AlertDelivery, 0, batchSize)
	ids := make([]int64, 0, batchSize)
	for rows.Next() {
		var d domain.AlertDelivery
		var incidentID sql.NullString
		if err := rows.Scan(&d.ID, &incidentID, &d.CheckID, &d.ChannelID, &d.Kind, &d.Status, &d.AttemptCount, &d.ChannelType, &d.Target); err != nil {
			return nil, fmt.Errorf("scan delivery: %w", err)
		}
		if incidentID.Valid {
			d.IncidentID = &incidentID.String
		}
		deliveries = append(deliveries, d)
		ids = append(ids, d.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deliveries: %w", err)
	}

	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, `
			UPDATE alert_deliveries
			SET status = 'sending', last_attempt_at = now(), attempt_count = attempt_count + 1
			WHERE id = $1
		`, id); err != nil {
			return nil, fmt.Errorf("mark delivery sending: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit claim deliveries: %w", err)
	}
	return deliveries, nil
}

func (s *Store) MarkDeliverySent(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE alert_deliveries
		SET status = 'sent', error_text = NULL
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("mark delivery sent: %w", err)
	}
	return nil
}

func (s *Store) MarkDeliveryFailed(ctx context.Context, id int64, attemptCount int, errText string) error {
	status := "failed"
	nextAttemptMinutes := int(math.Pow(2, float64(attemptCount)))
	if attemptCount >= 8 {
		status = "dead_letter"
		nextAttemptMinutes = 0
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE alert_deliveries
		SET status = $2,
			error_text = $3,
			next_attempt_at = CASE WHEN $4 = 0 THEN next_attempt_at ELSE now() + make_interval(mins => $4) END
		WHERE id = $1
	`, id, status, truncateString(errText, 1024), nextAttemptMinutes)
	if err != nil {
		return fmt.Errorf("mark delivery failed: %w", err)
	}
	return nil
}

func (s *Store) CleanupIdempotencyKeys(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM ping_idempotency WHERE expires_at <= now()
	`)
	if err != nil {
		return 0, fmt.Errorf("cleanup idempotency: %w", err)
	}
	n, _ := result.RowsAffected()
	return n, nil
}

func (s *Store) enqueueAlertsForAllChannels(ctx context.Context, tx *sql.Tx, userID, checkID string, incidentID *string, kind domain.DeliveryKind, priority int) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO alert_deliveries(
			incident_id,
			check_id,
			channel_id,
			kind,
			status,
			priority,
			next_attempt_at
		)
		SELECT $1, $2, id, $3, 'queued', $4, now()
		FROM alert_channels
		WHERE user_id = $5 AND enabled = true
	`, incidentID, checkID, kind, priority, userID)
	if err != nil {
		return fmt.Errorf("enqueue alerts: %w", err)
	}
	return nil
}

func updatedBaseline(current sql.NullFloat64, sampleCount int, value *int, alpha float64) (*float64, int) {
	if value == nil {
		if !current.Valid {
			return nil, sampleCount
		}
		v := current.Float64
		return &v, sampleCount
	}
	incoming := float64(*value)
	if !current.Valid {
		return &incoming, sampleCount + 1
	}
	next := alpha*incoming + (1-alpha)*current.Float64
	return &next, sampleCount + 1
}

func truncateString(v string, max int) string {
	if len(v) <= max {
		return v
	}
	return v[:max]
}
