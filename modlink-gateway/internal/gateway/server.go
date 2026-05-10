package gateway

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/modlinkcloud/modlink-gateway/internal/config"
	"github.com/modlinkcloud/modlink-gateway/internal/shared/envelope"
	"github.com/modlinkcloud/modlink-gateway/internal/shared/requestid"
	"github.com/modlinkcloud/modlink-gateway/internal/store"
)

func NewRouter(cfg *config.Config, st *store.Store) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/mlk/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/mlk/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := st.Ping(r.Context()); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/mlk/v1", func(r chi.Router) {
		r.Post("/chat/completions", chatCompletions(cfg, st))
		r.Get("/models", listModels(st))
		r.Get("/models/*", getModel(st))
		r.Post("/embeddings", embeddings(cfg, st))
	})

	return r
}

func apiKeyAuth(st *store.Store, r *http.Request) (*store.APIKeyRow, error) {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return nil, fmt.Errorf("unauthorized")
	}
	raw := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "bearer "))
	row, err := st.ResolveAPIKey(r.Context(), raw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("unauthorized")
		}
		return nil, err
	}
	return row, nil
}

func chatCompletions(cfg *config.Config, st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := requestid.New()
		keyRow, err := apiKeyAuth(st, r)
		if err != nil {
			envelope.Err(w, r, http.StatusUnauthorized, 40101, "INVALID_API_KEY", nil)
			return
		}
		_ = st.TouchAPIKeyUsed(r.Context(), keyRow.ID)

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			envelope.Err(w, r, http.StatusBadRequest, 40001, "READ_BODY", nil)
			return
		}
		var payload map[string]json.RawMessage
		if err := json.Unmarshal(bodyBytes, &payload); err != nil {
			envelope.Err(w, r, http.StatusBadRequest, 40002, "BAD_JSON", nil)
			return
		}
		rawModel, ok := payload["model"]
		if !ok {
			envelope.Err(w, r, http.StatusBadRequest, 40003, "MODEL_REQUIRED", nil)
			return
		}
		var modelName string
		if err := json.Unmarshal(rawModel, &modelName); err != nil {
			envelope.Err(w, r, http.StatusBadRequest, 40003, "MODEL_INVALID", nil)
			return
		}

		stream := false
		if raw, ok := payload["stream"]; ok {
			_ = json.Unmarshal(raw, &stream)
		}

		route, err := st.GetModelRoute(r.Context(), modelName)
		if err != nil || route == nil || !route.Enabled {
			envelope.Err(w, r, http.StatusNotFound, 40401, "MODEL_NOT_FOUND", map[string]any{"model": modelName})
			return
		}
		ch, err := st.GetChannel(r.Context(), route.ChannelID)
		if err != nil || ch == nil || ch.Status != "active" {
			envelope.Err(w, r, http.StatusServiceUnavailable, 50301, "CHANNEL_UNAVAILABLE", nil)
			return
		}

		inPer1k, outPer1k, _ := st.CurrentPricing(r.Context(), modelName)

		var wa *store.WalletAccount
		if keyRow.Scope == "org" && keyRow.OrgID != nil {
			wa, err = st.EnsureWallet(r.Context(), "org", *keyRow.OrgID)
		} else {
			wa, err = st.EnsureWallet(r.Context(), "user", keyRow.UserID)
		}
		if err != nil || wa.BalanceCents <= 0 {
			envelope.Err(w, r, http.StatusPaymentRequired, 40201, "INSUFFICIENT_BALANCE", map[string]any{"request_id": reqID})
			return
		}

		mm, _ := json.Marshal(route.UpstreamModelID)
		payload["model"] = json.RawMessage(mm)
		outBody, _ := json.Marshal(payload)

		upstreamKey, err := store.DecodeChannelAPIKey(ch.APIKeyCipher)
		if err != nil || upstreamKey == "" {
			upstreamKey = strings.TrimSpace(cfg.Upstream.OpenRouterAPIKey)
		}

		mode := strings.ToLower(strings.TrimSpace(cfg.Upstream.Mode))
		if mode == "mock" {
			handleMockChat(w, r, cfg, st, keyRow, route, modelName, reqID, stream, inPer1k, outPer1k, wa)
			return
		}

		if upstreamKey == "" {
			envelope.Err(w, r, http.StatusInternalServerError, 50001, "UPSTREAM_KEY_MISSING", map[string]any{"hint": "set upstream.openrouter_api_key or channels.api_key_cipher"})
			return
		}

		uclient := newUpstreamClient(cfg, upstreamKey, ch.BaseURL)
		if stream {
			uclient.proxySSE(w, r, outBody, reqID, st, keyRow, route, modelName, ch.ID, wa, inPer1k, outPer1k)
			return
		}
		uclient.proxyJSON(w, r, outBody, reqID, st, keyRow, route, modelName, ch.ID, wa, inPer1k, outPer1k)
	}
}

func handleMockChat(w http.ResponseWriter, r *http.Request, cfg *config.Config, st *store.Store, keyRow *store.APIKeyRow, route *store.ModelRoute, clientModel, reqID string, stream bool, inPer1k, outPer1k int64, wa *store.WalletAccount) {
	start := time.Now()
	inTok := 10
	outTok := 20
	cost := store.EstimateCostCents(inTok, outTok, inPer1k, outPer1k)
	if int64(wa.BalanceCents) < cost {
		envelope.Err(w, r, http.StatusPaymentRequired, 40201, "INSUFFICIENT_BALANCE", map[string]any{"request_id": reqID})
		return
	}
	if err := st.DebitInference(r.Context(), wa.ID, int64(cost), reqID, "estimated", clientModel); err != nil {
		envelope.Err(w, r, http.StatusPaymentRequired, 40201, "DEBIT_FAILED", map[string]any{"request_id": reqID})
		return
	}
	var orgID *uint64
	if keyRow.OrgID != nil {
		orgID = keyRow.OrgID
	}
	_ = st.InsertInferenceLog(r.Context(), reqID, keyRow.ID, keyRow.UserID, orgID, clientModel, &route.ChannelID,
		inTok, outTok, int64(cost), "estimated", 200, "mock", int(time.Since(start).Milliseconds()), false)

	if stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		fl, ok := w.(http.Flusher)
		if !ok {
			return
		}
		chunk := map[string]any{
			"id": reqID, "object": "chat.completion.chunk",
			"choices": []map[string]any{{"delta": map[string]any{"content": "mock "}, "index": 0}},
		}
		b, _ := json.Marshal(chunk)
		_, _ = w.Write([]byte("data: " + string(b) + "\n\n"))
		fl.Flush()
		last := map[string]any{
			"id": reqID, "object": "chat.completion.chunk", "choices": []map[string]any{{"delta": map[string]any{}, "finish_reason": "stop"}},
			"usage": map[string]any{"prompt_tokens": inTok, "completion_tokens": outTok, "total_tokens": inTok + outTok},
		}
		b2, _ := json.Marshal(last)
		_, _ = w.Write([]byte("data: " + string(b2) + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n"))
		fl.Flush()
		return
	}

	resp := map[string]any{
		"id":      reqID,
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   clientModel,
		"choices": []map[string]any{{
			"index": 0,
			"message": map[string]any{
				"role": "assistant", "content": "This is a mock completion from ModLinkCloud (upstream.mode=mock). Configure upstream.mode=openrouter and API keys.",
			},
			"finish_reason": "stop",
		}},
		"usage": map[string]any{
			"prompt_tokens": inTok, "completion_tokens": outTok, "total_tokens": inTok + outTok,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-Id", reqID)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func listModels(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := apiKeyAuth(st, r); err != nil {
			envelope.Err(w, r, http.StatusUnauthorized, 40101, "INVALID_API_KEY", nil)
			return
		}
		models, err := st.ListEnabledModels(r.Context())
		if err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50001, "LIST_FAILED", nil)
			return
		}
		items := make([]map[string]any, 0, len(models))
		for _, m := range models {
			items = append(items, map[string]any{"id": m, "object": "model", "owned_by": "modlink"})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "list", "data": items})
	}
}

func getModel(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := apiKeyAuth(st, r); err != nil {
			envelope.Err(w, r, http.StatusUnauthorized, 40101, "INVALID_API_KEY", nil)
			return
		}
		id := strings.Trim(chi.URLParam(r, "*"), "/")
		route, err := st.GetModelRoute(r.Context(), id)
		if err != nil || route == nil {
			envelope.Err(w, r, http.StatusNotFound, 40401, "NOT_FOUND", nil)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "object": "model", "owned_by": "modlink"})
	}
}

func embeddings(cfg *config.Config, st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !cfg.Embeddings.Enabled {
			envelope.Err(w, r, http.StatusServiceUnavailable, 50302, "EMBEDDINGS_DISABLED", nil)
			return
		}
		if _, err := apiKeyAuth(st, r); err != nil {
			envelope.Err(w, r, http.StatusUnauthorized, 40101, "INVALID_API_KEY", nil)
			return
		}
		envelope.Err(w, r, http.StatusNotImplemented, 50101, "EMBEDDINGS_NOT_IMPLEMENTED_USE_MOCK", nil)
	}
}

type upstreamClient struct {
	cfg    *config.Config
	apiKey string
	base   string
	cli    *http.Client
}

func newUpstreamClient(cfg *config.Config, apiKey, base string) *upstreamClient {
	return &upstreamClient{
		cfg:    cfg,
		apiKey: apiKey,
		base:   strings.TrimRight(base, "/"),
		cli: &http.Client{Timeout: time.Duration(cfg.Upstream.TimeoutSeconds) * time.Second},
	}
}

func (u *upstreamClient) proxyJSON(w http.ResponseWriter, r *http.Request, body []byte, reqID string, st *store.Store, keyRow *store.APIKeyRow, route *store.ModelRoute, clientModel string, channelID uint64, wa *store.WalletAccount, inPer1k, outPer1k int64) {
	start := time.Now()
	ctx := r.Context()
	url := u.base + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		envelope.Err(w, r, http.StatusInternalServerError, 50001, "UPSTREAM_BUILD", nil)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+u.apiKey)

	res, err := u.cli.Do(req)
	if err != nil {
		envelope.Err(w, r, http.StatusBadGateway, 50201, "UPSTREAM_ERROR", map[string]any{"error": err.Error()})
		return
	}
	defer res.Body.Close()
	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		envelope.Err(w, r, http.StatusBadGateway, 50202, "UPSTREAM_READ", nil)
		return
	}

	var parsed struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	_ = json.Unmarshal(respBody, &parsed)
	inTok := parsed.Usage.PromptTokens
	outTok := parsed.Usage.CompletionTokens
	if inTok == 0 && outTok == 0 {
		inTok = 50
		outTok = 50
	}
	billing := "actual"
	cost := store.EstimateCostCents(inTok, outTok, inPer1k, outPer1k)
	if err := st.DebitInference(ctx, wa.ID, cost, reqID, billing, clientModel); err != nil {
		envelope.Err(w, r, http.StatusPaymentRequired, 40201, "INSUFFICIENT_BALANCE", map[string]any{"request_id": reqID})
		return
	}
	var orgID *uint64
	if keyRow.OrgID != nil {
		orgID = keyRow.OrgID
	}
	chID := channelID
	_ = st.InsertInferenceLog(ctx, reqID, keyRow.ID, keyRow.UserID, orgID, clientModel, &chID,
		inTok, outTok, cost, billing, res.StatusCode, "ok", int(time.Since(start).Milliseconds()), false)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-Id", reqID)
	w.WriteHeader(res.StatusCode)
	_, _ = w.Write(respBody)
}

func (u *upstreamClient) proxySSE(w http.ResponseWriter, r *http.Request, body []byte, reqID string, st *store.Store, keyRow *store.APIKeyRow, route *store.ModelRoute, clientModel string, channelID uint64, wa *store.WalletAccount, inPer1k, outPer1k int64) {
	ctx := r.Context()
	url := u.base + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		envelope.Err(w, r, http.StatusInternalServerError, 50001, "UPSTREAM_BUILD", nil)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+u.apiKey)
	req.Header.Set("Accept", "text/event-stream")

	res, err := u.cli.Do(req)
	if err != nil {
		envelope.Err(w, r, http.StatusBadGateway, 50201, "UPSTREAM_ERROR", nil)
		return
	}
	defer res.Body.Close()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Request-Id", reqID)
	w.WriteHeader(res.StatusCode)
	fl, ok := w.(http.Flusher)
	if !ok {
		io.Copy(w, res.Body)
		return
	}

	var buf bytes.Buffer
	tee := io.TeeReader(res.Body, &buf)
	raw, _ := io.ReadAll(tee)

	inTok, outTok := parseSSEUsage(raw)
	if inTok == 0 && outTok == 0 {
		inTok, outTok = 50, 50
	}
	cost := store.EstimateCostCents(inTok, outTok, inPer1k, outPer1k)
	if err := st.DebitInference(ctx, wa.ID, cost, reqID, "actual", clientModel); err != nil {
		envelope.Err(w, r, http.StatusPaymentRequired, 40201, "INSUFFICIENT_BALANCE", map[string]any{"request_id": reqID})
		return
	}
	var orgID *uint64
	if keyRow.OrgID != nil {
		orgID = keyRow.OrgID
	}
	chID := channelID
	_ = st.InsertInferenceLog(ctx, reqID, keyRow.ID, keyRow.UserID, orgID, clientModel, &chID,
		inTok, outTok, cost, "actual", res.StatusCode, "stream", 0, false)

	_, _ = w.Write(buf.Bytes())
	fl.Flush()
}

func parseSSEUsage(raw []byte) (inTok, outTok int) {
	lines := bytes.Split(raw, []byte("\n"))
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if !bytes.HasPrefix(line, []byte("data:")) {
			continue
		}
		payload := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		if bytes.Equal(payload, []byte("[DONE]")) {
			continue
		}
		var m map[string]any
		if json.Unmarshal(payload, &m) != nil {
			continue
		}
		if u, ok := m["usage"].(map[string]any); ok {
			if v, ok := u["prompt_tokens"].(float64); ok {
				inTok = int(v)
			}
			if v, ok := u["completion_tokens"].(float64); ok {
				outTok = int(v)
			}
		}
	}
	return
}
