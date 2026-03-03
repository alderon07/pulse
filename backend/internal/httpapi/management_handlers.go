package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"pulse/backend/internal/domain"
	"pulse/backend/internal/repo"
	"pulse/backend/internal/service"
)

const (
	scheduleModeManual                 = "manual"
	scheduleModeAuto                   = "auto"
	defaultAutoExpectedIntervalSeconds = 3600
	defaultAutoGraceSeconds            = 180
)

type createCheckRequest struct {
	Name                    string `json:"name"`
	ExpectedIntervalSeconds *int   `json:"expected_interval_seconds"`
	GraceSeconds            *int   `json:"grace_seconds"`
	ScheduleMode            string `json:"schedule_mode"`
}

type patchCheckRequest struct {
	Name                    *string `json:"name"`
	ExpectedIntervalSeconds *int    `json:"expected_interval_seconds"`
	GraceSeconds            *int    `json:"grace_seconds"`
	ScheduleMode            *string `json:"schedule_mode"`
}

type createAlertChannelRequest struct {
	Type   string `json:"type"`
	Target string `json:"target"`
}

type patchAlertChannelRequest struct {
	Enabled *bool   `json:"enabled"`
	Target  *string `json:"target"`
}

func (s *Server) handleCreateCheck(w http.ResponseWriter, r *http.Request) {
	userID, email, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if err := s.store.EnsureUser(r.Context(), userID, email); err != nil {
		s.logger.Error("ensure user", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var req createCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	mode := normalizeScheduleMode(req.ScheduleMode)
	if mode == "" {
		writeError(w, http.StatusBadRequest, "schedule_mode must be manual or auto")
		return
	}

	expectedIntervalSeconds := defaultAutoExpectedIntervalSeconds
	if req.ExpectedIntervalSeconds != nil {
		expectedIntervalSeconds = *req.ExpectedIntervalSeconds
	}
	graceSeconds := defaultAutoGraceSeconds
	if req.GraceSeconds != nil {
		graceSeconds = *req.GraceSeconds
	}
	if req.Name == "" || expectedIntervalSeconds <= 0 || graceSeconds < 0 {
		writeError(w, http.StatusBadRequest, "name is required; expected_interval_seconds must be > 0; grace_seconds must be >= 0")
		return
	}
	if mode == scheduleModeManual && (req.ExpectedIntervalSeconds == nil || req.GraceSeconds == nil) {
		writeError(w, http.StatusBadRequest, "manual mode requires expected_interval_seconds and grace_seconds")
		return
	}
	token, err := service.GenerateSecureToken(32)
	if err != nil {
		s.logger.Error("generate check token", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	check, err := s.store.CreateCheck(
		r.Context(),
		userID,
		req.Name,
		token,
		expectedIntervalSeconds,
		graceSeconds,
		mode,
	)
	if err != nil {
		s.logger.Error("create check", "error", err)
		writeError(w, http.StatusBadRequest, "failed to create check")
		return
	}
	writeJSON(w, http.StatusCreated, check)
}

func (s *Server) handleListChecks(w http.ResponseWriter, r *http.Request) {
	userID, email, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if err := s.store.EnsureUser(r.Context(), userID, email); err != nil {
		s.logger.Error("ensure user", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	limit := intFromQuery(r, "limit", 50)
	if limit > 200 {
		limit = 200
	}
	offset := intFromQuery(r, "offset", 0)
	checks, err := s.store.ListChecks(r.Context(), userID, limit, offset)
	if err != nil {
		s.logger.Error("list checks", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": checks})
}

func (s *Server) handleGetCheck(w http.ResponseWriter, r *http.Request) {
	userID, _, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	id := chi.URLParam(r, "id")
	check, err := s.store.GetCheck(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "check not found")
			return
		}
		s.logger.Error("get check", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, check)
}

func (s *Server) handlePatchCheck(w http.ResponseWriter, r *http.Request) {
	userID, _, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	id := chi.URLParam(r, "id")
	var req patchCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == nil && req.ExpectedIntervalSeconds == nil && req.GraceSeconds == nil && req.ScheduleMode == nil {
		writeError(w, http.StatusBadRequest, "no patch fields provided")
		return
	}
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		req.Name = &trimmed
		if *req.Name == "" {
			writeError(w, http.StatusBadRequest, "name cannot be empty")
			return
		}
	}
	if req.ExpectedIntervalSeconds != nil && *req.ExpectedIntervalSeconds <= 0 {
		writeError(w, http.StatusBadRequest, "expected_interval_seconds must be > 0")
		return
	}
	if req.GraceSeconds != nil && *req.GraceSeconds < 0 {
		writeError(w, http.StatusBadRequest, "grace_seconds must be >= 0")
		return
	}
	if req.ScheduleMode != nil {
		if strings.TrimSpace(*req.ScheduleMode) == "" {
			writeError(w, http.StatusBadRequest, "schedule_mode must be manual or auto")
			return
		}
		normalized := normalizeScheduleMode(*req.ScheduleMode)
		if normalized == "" {
			writeError(w, http.StatusBadRequest, "schedule_mode must be manual or auto")
			return
		}
		req.ScheduleMode = &normalized
	}

	check, err := s.store.UpdateCheck(r.Context(), userID, id, repo.CheckPatch{
		Name:                    req.Name,
		ExpectedIntervalSeconds: req.ExpectedIntervalSeconds,
		GraceSeconds:            req.GraceSeconds,
		ScheduleMode:            req.ScheduleMode,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "check not found")
			return
		}
		s.logger.Error("update check", "error", err)
		writeError(w, http.StatusBadRequest, "failed to update check")
		return
	}
	writeJSON(w, http.StatusOK, check)
}

func (s *Server) handlePauseCheck(w http.ResponseWriter, r *http.Request) {
	userID, _, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	id := chi.URLParam(r, "id")
	check, err := s.store.PauseCheck(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "check not found")
			return
		}
		s.logger.Error("pause check", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, check)
}

func (s *Server) handleResumeCheck(w http.ResponseWriter, r *http.Request) {
	userID, _, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	id := chi.URLParam(r, "id")
	check, err := s.store.ResumeCheck(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "check not found")
			return
		}
		s.logger.Error("resume check", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, check)
}

func (s *Server) handleRotateCheckToken(w http.ResponseWriter, r *http.Request) {
	userID, _, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	id := chi.URLParam(r, "id")
	token, err := service.GenerateSecureToken(32)
	if err != nil {
		s.logger.Error("generate check token", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	check, err := s.store.RotateCheckToken(r.Context(), userID, id, token)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "check not found")
			return
		}
		s.logger.Error("rotate check token", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, check)
}

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	userID, _, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	id := chi.URLParam(r, "id")
	limit := intFromQuery(r, "limit", 50)
	if limit > 200 {
		limit = 200
	}
	events, err := s.store.ListEvents(r.Context(), userID, id, limit)
	if err != nil {
		s.logger.Error("list events", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": events})
}

func (s *Server) handleCreateAlertChannel(w http.ResponseWriter, r *http.Request) {
	userID, _, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	var req createAlertChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Type = strings.TrimSpace(req.Type)
	req.Target = strings.TrimSpace(req.Target)
	if req.Target == "" {
		writeError(w, http.StatusBadRequest, "target is required")
		return
	}
	channelType := domain.AlertChannelType(req.Type)
	if channelType != domain.AlertChannelEmail && channelType != domain.AlertChannelWebhook {
		writeError(w, http.StatusBadRequest, "type must be email or webhook")
		return
	}
	channel, err := s.store.CreateAlertChannel(r.Context(), userID, channelType, req.Target)
	if err != nil {
		s.logger.Error("create alert channel", "error", err)
		writeError(w, http.StatusBadRequest, "failed to create alert channel")
		return
	}
	channel.Target = maskTarget(channel.Target)
	writeJSON(w, http.StatusCreated, channel)
}

func (s *Server) handleListAlertChannels(w http.ResponseWriter, r *http.Request) {
	userID, _, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	limit := intFromQuery(r, "limit", 50)
	if limit > 200 {
		limit = 200
	}
	offset := intFromQuery(r, "offset", 0)
	channels, err := s.store.ListAlertChannels(r.Context(), userID, limit, offset)
	if err != nil {
		s.logger.Error("list alert channels", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	for i := range channels {
		channels[i].Target = maskTarget(channels[i].Target)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": channels})
}

func (s *Server) handlePatchAlertChannel(w http.ResponseWriter, r *http.Request) {
	userID, _, err := userFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	id := chi.URLParam(r, "id")
	var req patchAlertChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Enabled == nil && req.Target == nil {
		writeError(w, http.StatusBadRequest, "no patch fields provided")
		return
	}
	if req.Target != nil {
		trimmed := strings.TrimSpace(*req.Target)
		if trimmed == "" {
			writeError(w, http.StatusBadRequest, "target cannot be empty")
			return
		}
		req.Target = &trimmed
	}

	channel, err := s.store.UpdateAlertChannel(r.Context(), userID, id, req.Enabled, req.Target)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "alert channel not found")
			return
		}
		s.logger.Error("update alert channel", "error", err)
		writeError(w, http.StatusBadRequest, "failed to update alert channel")
		return
	}
	channel.Target = maskTarget(channel.Target)
	writeJSON(w, http.StatusOK, channel)
}

func intFromQuery(r *http.Request, key string, fallback int) int {
	if raw := strings.TrimSpace(r.URL.Query().Get(key)); raw != "" {
		v, err := strconv.Atoi(raw)
		if err == nil && v >= 0 {
			return v
		}
	}
	return fallback
}

func normalizeScheduleMode(raw string) string {
	mode := strings.ToLower(strings.TrimSpace(raw))
	if mode == "" {
		return scheduleModeManual
	}
	if mode == scheduleModeManual || mode == scheduleModeAuto {
		return mode
	}
	return ""
}

func maskTarget(target string) string {
	if target == "" {
		return target
	}
	if strings.Contains(target, "@") {
		parts := strings.Split(target, "@")
		local := parts[0]
		domain := parts[1]
		if len(local) <= 2 {
			return "**@" + domain
		}
		return local[:2] + "***@" + domain
	}
	if len(target) <= 10 {
		return "***"
	}
	return target[:8] + "***"
}
