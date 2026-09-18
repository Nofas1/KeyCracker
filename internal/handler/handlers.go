package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"key_cracker/internal/service"
	"key_cracker/internal"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

type SolveRequest struct {
	InitialState string  `json:"initial_state"`
	Relations    [][]int `json:"relations"`
}

type StepResponse struct {
	Pos       int `json:"pos"`
	Direction int `json:"direction"`
}

type SolveResponse struct {
	Success bool           `json:"success"`
	Steps   []StepResponse `json:"steps,omitempty"`
	Error   string         `json:"error,omitempty"`
}

type Handler struct {
	service *service.KeyCrackerService
	logger  *slog.Logger
}

func NewHandler(service *service.KeyCrackerService, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func RegisterHandlers(mux *http.ServeMux, svc *service.KeyCrackerService, logger *slog.Logger) {
	mux.HandleFunc("POST /solve", func(w http.ResponseWriter, r *http.Request) {
		
		var req SolveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		key_cracker := internal.Constructor(req.Relations, req.InitialState)
		key_cracker.BuildTree()
		ans := key_cracker.Answer()
		
		steps := make([]StepResponse, len(ans))
		for i, step := range ans {
			steps[i] = StepResponse{
				Pos:       step.Pos,
				Direction: step.Dir,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SolveResponse{
			Success: true,
			Steps:   steps,
		})
	})
}

func Ping() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		w.WriteHeader(http.StatusOK)
	}
}

type KeyCrackerClient struct {
	baseURL string
	logger  *slog.Logger
	client  *http.Client
}

func NewKeyCrackerClient(baseURL string, logger *slog.Logger) *KeyCrackerClient {
	if logger == nil {
		logger = slog.Default()
	}
	return &KeyCrackerClient{
		baseURL: baseURL,
		logger: logger,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (kc *KeyCrackerClient) Solve(ctx context.Context, initialState string, relations [][]int) (*SolveResponse, error) {
	kc.logger.InfoContext(ctx, "starting solve request",
		slog.String("initial_state", initialState),
		slog.Int("relations_count", len(relations)),
	)

	reqBody := SolveRequest{
		InitialState: initialState,
		Relations:    relations,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		kc.logger.ErrorContext(ctx, "failed to marshal request", slog.Any("error", err))
		return nil, fmt.Errorf("solve: failed to marshal request: %w", err)
	}

	joinURL, err := url.JoinPath(kc.baseURL, "/solve")
	if err != nil {
		kc.logger.ErrorContext(ctx, "failed to build URL", slog.Any("error", err))
		return nil, fmt.Errorf("solve: failed to build URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL, bytes.NewReader(body))
	if err != nil {
		kc.logger.ErrorContext(ctx, "failed to create request", slog.Any("error", err))
		return nil, fmt.Errorf("solve: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := kc.client.Do(req)
	if err != nil {
		kc.logger.ErrorContext(ctx, "server unavailable", slog.Any("error", err))
		return nil, fmt.Errorf("solve: server unavailable: %w", err)
	}
	defer resp.Body.Close()

	var result SolveResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		kc.logger.ErrorContext(ctx, "failed to decode response", slog.Any("error", err))
		return nil, fmt.Errorf("solve: failed to decode response: %w", err)
	}

	if !result.Success {
		kc.logger.WarnContext(ctx, "solve returned error",
			slog.String("error", result.Error),
		)
		return nil, fmt.Errorf("solve: server returned error: %s", result.Error)
	}

	kc.logger.InfoContext(ctx, "solve completed successfully",
		slog.Int("steps_count", len(result.Steps)),
	)

	return &result, nil
}

