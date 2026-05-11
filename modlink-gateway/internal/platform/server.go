package platform

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/modlinkcloud/modlink-gateway/internal/auth"
	"github.com/modlinkcloud/modlink-gateway/internal/config"
	"github.com/modlinkcloud/modlink-gateway/internal/httpserver"
	"github.com/modlinkcloud/modlink-gateway/internal/shared/envelope"
	"github.com/modlinkcloud/modlink-gateway/internal/store"
)

func hashRefresh(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func randToken() string {
	var b [32]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

type tokenOut struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        int    `json:"expires_in"`
	TokenType        string `json:"token_type"`
	RefreshExpiresIn int    `json:"refresh_expires_in,omitempty"`
}

func issueTokens(cfg *config.Config, st *store.Store, u *store.User, orgID *uint64) (*tokenOut, error) {
	cl := &auth.Claims{
		UserID:       u.ID,
		Email:        u.Email,
		Role:         u.Role,
		CurrentOrgID: orgID,
	}
	ttl := time.Duration(cfg.JWT.AccessTTLMinutes) * time.Minute
	at, err := auth.SignAccess(cfg.JWT.Secret, cfg.JWT.Issuer, cl, ttl)
	if err != nil {
		return nil, err
	}
	rt := randToken()
	rth := hashRefresh(rt)
	exp := time.Now().Add(time.Duration(cfg.JWT.RefreshTTLDays) * 24 * time.Hour)
	if err := st.InsertRefreshToken(context.Background(), u.ID, rth, exp, ""); err != nil {
		return nil, err
	}
	return &tokenOut{
		AccessToken:      at,
		RefreshToken:     rt,
		ExpiresIn:        int(ttl.Seconds()),
		TokenType:        "Bearer",
		RefreshExpiresIn: int(time.Until(exp).Seconds()),
	}, nil
}

// NewRouter mounts /mlk/health, /mlk/platform/v1/*
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
			envelope.Err(w, r, http.StatusServiceUnavailable, 50301, "NOT_READY", nil)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/mlk/platform/v1", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			if err := st.Ping(r.Context()); err != nil {
				envelope.Err(w, r, http.StatusServiceUnavailable, 50301, "DB_UNAVAILABLE", map[string]any{"error": err.Error()})
				return
			}
			envelope.OK(w, r, map[string]any{"database": "up"})
		})

		// --- public auth ---
		r.Get("/auth/login", func(w http.ResponseWriter, r *http.Request) {
			envelope.Err(w, r, http.StatusMethodNotAllowed, 40501, "METHOD_NOT_ALLOWED", map[string]any{
				"hint": "登录必须使用 POST；Content-Type: application/json；body: {\"email\":\"...\",\"password\":\"...\"}",
			})
		})

		r.Post("/auth/register", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Email        string `json:"email"`
				Password     string `json:"password"`
				DisplayName  string `json:"display_name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				envelope.Err(w, r, http.StatusBadRequest, 40001, "BAD_JSON", nil)
				return
			}
			if body.Email == "" || len(body.Password) < 8 {
				envelope.Err(w, r, http.StatusBadRequest, 40002, "INVALID_PARAMS", nil)
				return
			}
			if u, _ := st.GetUserByEmail(r.Context(), body.Email); u != nil {
				envelope.Err(w, r, http.StatusConflict, 40901, "EMAIL_EXISTS", nil)
				return
			}
			ph, err := auth.HashPassword(body.Password)
			if err != nil {
				envelope.Err(w, r, http.StatusInternalServerError, 50001, "HASH_ERROR", nil)
				return
			}
			role := "user"
			for _, e := range cfg.Security.BootstrapAdminEmails {
				if strings.EqualFold(strings.TrimSpace(e), strings.TrimSpace(body.Email)) {
					role = "admin"
					break
				}
			}
			uid, err := st.CreateUser(r.Context(), body.Email, ph, body.DisplayName, role)
			if err != nil {
				envelope.Err(w, r, http.StatusInternalServerError, 50002, "CREATE_USER_FAILED", map[string]any{"error": err.Error()})
				return
			}
			if _, err := st.EnsureWallet(r.Context(), "user", uid); err != nil {
				envelope.Err(w, r, http.StatusInternalServerError, 50003, "WALLET_INIT_FAILED", nil)
				return
			}
			u := &store.User{ID: uid, Email: strings.ToLower(strings.TrimSpace(body.Email)), Role: role, Status: "active"}
			tok, err := issueTokens(cfg, st, u, nil)
			if err != nil {
				envelope.Err(w, r, http.StatusInternalServerError, 50004, "TOKEN_ISSUE_FAILED", map[string]any{"error": err.Error()})
				return
			}
			envelope.OK(w, r, tok)
		})

		r.Post("/auth/login", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				envelope.Err(w, r, http.StatusBadRequest, 40001, "BAD_JSON", nil)
				return
			}
			body.Email = strings.TrimSpace(strings.ToLower(body.Email))
			u, err := st.GetUserByEmail(r.Context(), body.Email)
			if err != nil || u == nil || !auth.CheckPassword(u.PasswordHash, body.Password) {
				envelope.Err(w, r, http.StatusUnauthorized, 40103, "BAD_CREDENTIALS", nil)
				return
			}
			if u.Status != "active" {
				envelope.Err(w, r, http.StatusForbidden, 40302, "USER_DISABLED", nil)
				return
			}
			_ = st.TouchLogin(r.Context(), u.ID)
			var curOrg *uint64
			if orgs, err := st.ListUserOrgs(r.Context(), u.ID); err == nil && len(orgs) > 0 {
				oid := orgs[0].ID
				curOrg = &oid
			}
			tok, err := issueTokens(cfg, st, u, curOrg)
			if err != nil {
				envelope.Err(w, r, http.StatusInternalServerError, 50004, "TOKEN_ISSUE_FAILED", map[string]any{"error": err.Error()})
				return
			}
			envelope.OK(w, r, tok)
		})

		r.Post("/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				RefreshToken string `json:"refresh_token"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RefreshToken == "" {
				envelope.Err(w, r, http.StatusBadRequest, 40001, "BAD_JSON", nil)
				return
			}
			h := hashRefresh(body.RefreshToken)
			uid, exp, rev, err := st.FindRefreshToken(r.Context(), h)
			if err != nil || rev.Valid {
				envelope.Err(w, r, http.StatusUnauthorized, 40104, "INVALID_REFRESH", nil)
				return
			}
			if time.Now().After(exp) {
				envelope.Err(w, r, http.StatusUnauthorized, 40105, "REFRESH_EXPIRED", nil)
				return
			}
			u, err := st.GetUserByID(r.Context(), uid)
			if err != nil || u == nil || u.Status != "active" {
				envelope.Err(w, r, http.StatusUnauthorized, 40106, "USER_INVALID", nil)
				return
			}
			var curOrg *uint64
			if orgs, err := st.ListUserOrgs(r.Context(), u.ID); err == nil && len(orgs) > 0 {
				oid := orgs[0].ID
				curOrg = &oid
			}
			tok, err := issueTokens(cfg, st, u, curOrg)
			if err != nil {
				envelope.Err(w, r, http.StatusInternalServerError, 50004, "TOKEN_ISSUE_FAILED", map[string]any{"error": err.Error()})
				return
			}
			_ = st.RevokeRefreshToken(r.Context(), h)
			envelope.OK(w, r, tok)
		})

		r.Post("/auth/password/reset-request", func(w http.ResponseWriter, r *http.Request) {
			envelope.OK(w, r, map[string]any{"sent": cfg.SMS.Mode == "mock"})
		})
		r.Post("/auth/password/reset-confirm", func(w http.ResponseWriter, r *http.Request) {
			envelope.OK(w, r, map[string]any{"ok": true})
		})

		r.Group(func(r chi.Router) {
			r.Use(httpserver.BearerJWT(cfg))

			r.Post("/auth/logout", func(w http.ResponseWriter, r *http.Request) {
				cl, _ := httpserver.ClaimsFrom(r.Context())
				if cl != nil {
					_ = st.RevokeAllRefreshTokens(r.Context(), cl.UserID)
				}
				envelope.OK(w, r, map[string]any{"ok": true})
			})

			r.Get("/auth/me", meHandler(st))
			r.Get("/users/me", meHandler(st))
			r.Patch("/users/me", func(w http.ResponseWriter, r *http.Request) {
				cl, ok := httpserver.ClaimsFrom(r.Context())
				if !ok {
					envelope.Err(w, r, http.StatusUnauthorized, 40101, "UNAUTHORIZED", nil)
					return
				}
				var body struct {
					DisplayName string `json:"display_name"`
					AvatarURL   string `json:"avatar_url"`
				}
				_ = json.NewDecoder(r.Body).Decode(&body)
				if err := st.UpdateProfile(r.Context(), cl.UserID, body.DisplayName, body.AvatarURL); err != nil {
					envelope.Err(w, r, http.StatusInternalServerError, 50001, "UPDATE_FAILED", nil)
					return
				}
				envelope.OK(w, r, map[string]any{"ok": true})
			})

			r.Get("/orgs", listOrgs(st))
			r.Post("/orgs", createOrg(st))
			r.Get("/orgs/{org_id}", getOrg(st))
			r.Post("/orgs/{org_id}/switch", switchOrg(cfg, st))

			r.Get("/api-keys", listKeys(st))
			r.Post("/api-keys", createKey(st))
			r.Delete("/api-keys/{key_id}", deleteKey(st))

			r.Get("/wallet", walletBalance(st))
			r.Post("/orders/recharge", recharge(cfg, st))
			r.Get("/orders/{order_id}", getOrder(st))
			r.Get("/orders", listOrders(st))
			r.Post("/payment/mock/complete", mockPay(cfg, st))

			r.Get("/usage/summary", usageSummary(st))

			r.Group(func(r chi.Router) {
				r.Use(httpserver.RequireAdmin)
				r.Get("/admin/users", adminUsers(st))
				r.Get("/admin/users/{user_id}", adminUserDetail(st))
				r.Get("/admin/channels", adminChannels(st))
				r.Get("/admin/channels/{channel_id}", adminGetChannel(st))
				r.Patch("/admin/channels/{channel_id}", adminPatchChannel(st))
				r.Get("/admin/models", adminModels(st))
				r.Post("/admin/models", adminCreateModel(st))
			})
		})
	})

	return r
}

func meHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cl, ok := httpserver.ClaimsFrom(r.Context())
		if !ok {
			envelope.Err(w, r, http.StatusUnauthorized, 40101, "UNAUTHORIZED", nil)
			return
		}
		u, err := st.GetUserByID(r.Context(), cl.UserID)
		if err != nil || u == nil {
			envelope.Err(w, r, http.StatusNotFound, 40401, "NOT_FOUND", nil)
			return
		}
		dn := ""
		if u.DisplayName.Valid {
			dn = u.DisplayName.String
		}
		var cur any
		if cl.CurrentOrgID != nil {
			cur = *cl.CurrentOrgID
		}
		envelope.OK(w, r, map[string]any{
			"id":             u.ID,
			"email":          u.Email,
			"display_name":   dn,
			"role":           u.Role,
			"current_org_id": cur,
		})
	}
}
