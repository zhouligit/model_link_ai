package platform

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/modlinkcloud/modlink-gateway/internal/config"
	"github.com/modlinkcloud/modlink-gateway/internal/httpserver"
	"github.com/modlinkcloud/modlink-gateway/internal/shared/envelope"
	"github.com/modlinkcloud/modlink-gateway/internal/store"
)

// canAccessOrg: 普通用户需为组织成员；管理员视为拥有全部「用户侧」组织权限（查看/切组织/钱包/用量等）。
func canAccessOrg(st *store.Store, r *http.Request, orgID uint64) bool {
	cl, ok := httpserver.ClaimsFrom(r.Context())
	if !ok {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(cl.Role), "admin") {
		return true
	}
	_, mem, err := st.IsOrgMember(r.Context(), orgID, cl.UserID)
	return err == nil && mem
}

func listOrgs(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cl, ok := httpserver.ClaimsFrom(r.Context())
		if !ok {
			envelope.Err(w, r, http.StatusUnauthorized, 40101, "UNAUTHORIZED", nil)
			return
		}
		var orgs []store.Org
		var err error
		if strings.EqualFold(strings.TrimSpace(cl.Role), "admin") {
			orgs, err = st.ListAllOrgs(r.Context(), 200)
		} else {
			orgs, err = st.ListUserOrgs(r.Context(), cl.UserID)
		}
		if err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50001, "LIST_FAILED", nil)
			return
		}
		out := make([]map[string]any, 0, len(orgs))
		for _, o := range orgs {
			slug := ""
			if o.Slug.Valid {
				slug = o.Slug.String
			}
			out = append(out, map[string]any{"id": o.ID, "name": o.Name, "slug": slug, "status": o.Status})
		}
		envelope.OK(w, r, map[string]any{"items": out})
	}
}

func createOrg(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cl, ok := httpserver.ClaimsFrom(r.Context())
		if !ok {
			envelope.Err(w, r, http.StatusUnauthorized, 40101, "UNAUTHORIZED", nil)
			return
		}
		var body struct {
			Name string `json:"name"`
			Slug string `json:"slug"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
			envelope.Err(w, r, http.StatusBadRequest, 40001, "BAD_JSON", nil)
			return
		}
		oid, err := st.CreateOrg(r.Context(), body.Name, cl.UserID, body.Slug)
		if err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50001, "CREATE_ORG_FAILED", map[string]any{"error": err.Error()})
			return
		}
		if _, err := st.EnsureWallet(r.Context(), "org", oid); err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50002, "ORG_WALLET_FAILED", nil)
			return
		}
		envelope.OK(w, r, map[string]any{"org_id": oid})
	}
}

func getOrg(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := httpserver.ClaimsFrom(r.Context())
		if !ok {
			envelope.Err(w, r, http.StatusUnauthorized, 40101, "UNAUTHORIZED", nil)
			return
		}
		oid, err := strconv.ParseUint(chi.URLParam(r, "org_id"), 10, 64)
		if err != nil {
			envelope.Err(w, r, http.StatusBadRequest, 40001, "BAD_ORG_ID", nil)
			return
		}
		if !canAccessOrg(st, r, oid) {
			envelope.Err(w, r, http.StatusForbidden, 40301, "NOT_MEMBER", nil)
			return
		}
		o, err := st.GetOrg(r.Context(), oid)
		if err != nil || o == nil {
			envelope.Err(w, r, http.StatusNotFound, 40401, "NOT_FOUND", nil)
			return
		}
		slug := ""
		if o.Slug.Valid {
			slug = o.Slug.String
		}
		envelope.OK(w, r, map[string]any{"id": o.ID, "name": o.Name, "slug": slug, "owner_user_id": o.OwnerUserID})
	}
}

func switchOrg(cfg *config.Config, st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cl, ok := httpserver.ClaimsFrom(r.Context())
		if !ok {
			envelope.Err(w, r, http.StatusUnauthorized, 40101, "UNAUTHORIZED", nil)
			return
		}
		oid, err := strconv.ParseUint(chi.URLParam(r, "org_id"), 10, 64)
		if err != nil {
			envelope.Err(w, r, http.StatusBadRequest, 40001, "BAD_ORG_ID", nil)
			return
		}
		if !canAccessOrg(st, r, oid) {
			envelope.Err(w, r, http.StatusForbidden, 40301, "NOT_MEMBER", nil)
			return
		}
		u, err := st.GetUserByID(r.Context(), cl.UserID)
		if err != nil || u == nil {
			envelope.Err(w, r, http.StatusUnauthorized, 40106, "USER_INVALID", nil)
			return
		}
		tok, err := issueTokens(cfg, st, u, &oid)
		if err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50004, "TOKEN_ISSUE_FAILED", nil)
			return
		}
		envelope.OK(w, r, tok)
	}
}

func listKeys(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cl, ok := httpserver.ClaimsFrom(r.Context())
		if !ok {
			envelope.Err(w, r, http.StatusUnauthorized, 40101, "UNAUTHORIZED", nil)
			return
		}
		keys, err := st.ListAPIKeys(r.Context(), cl.UserID)
		if err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50001, "LIST_FAILED", nil)
			return
		}
		items := make([]map[string]any, 0, len(keys))
		for _, k := range keys {
			row := map[string]any{
				"id": k.ID, "name": k.Name, "scope": k.Scope, "key_prefix": k.KeyPrefix, "status": k.Status,
			}
			if k.OrgID != nil {
				row["org_id"] = *k.OrgID
			}
			items = append(items, row)
		}
		envelope.OK(w, r, map[string]any{"items": items})
	}
}

func createKey(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cl, ok := httpserver.ClaimsFrom(r.Context())
		if !ok {
			envelope.Err(w, r, http.StatusUnauthorized, 40101, "UNAUTHORIZED", nil)
			return
		}
		var body struct {
			Name  string `json:"name"`
			Scope string `json:"scope"`
			OrgID *uint64 `json:"org_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || (body.Scope != "personal" && body.Scope != "org") {
			envelope.Err(w, r, http.StatusBadRequest, 40001, "BAD_JSON", nil)
			return
		}
		var orgPtr *uint64
		if body.Scope == "org" {
			if body.OrgID == nil {
				envelope.Err(w, r, http.StatusBadRequest, 40002, "ORG_ID_REQUIRED", nil)
				return
			}
			if !canAccessOrg(st, r, *body.OrgID) {
				envelope.Err(w, r, http.StatusForbidden, 40301, "NOT_MEMBER", nil)
				return
			}
			orgPtr = body.OrgID
		}
		full, prefix, hash, err := store.GenerateAPIKey()
		if err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50001, "KEY_GEN_FAILED", nil)
			return
		}
		id, err := st.InsertAPIKey(r.Context(), cl.UserID, orgPtr, body.Scope, body.Name, prefix, hash)
		if err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50002, "KEY_INSERT_FAILED", map[string]any{"error": err.Error()})
			return
		}
		envelope.OK(w, r, map[string]any{"id": id, "secret": full, "key_prefix": prefix})
	}
}

func deleteKey(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cl, ok := httpserver.ClaimsFrom(r.Context())
		if !ok {
			envelope.Err(w, r, http.StatusUnauthorized, 40101, "UNAUTHORIZED", nil)
			return
		}
		id, err := strconv.ParseUint(chi.URLParam(r, "key_id"), 10, 64)
		if err != nil {
			envelope.Err(w, r, http.StatusBadRequest, 40001, "BAD_ID", nil)
			return
		}
		if err := st.DisableAPIKey(r.Context(), cl.UserID, id); err != nil {
			envelope.Err(w, r, http.StatusNotFound, 40401, "NOT_FOUND", nil)
			return
		}
		envelope.OK(w, r, map[string]any{"ok": true})
	}
}

func walletBalance(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cl, ok := httpserver.ClaimsFrom(r.Context())
		if !ok {
			envelope.Err(w, r, http.StatusUnauthorized, 40101, "UNAUTHORIZED", nil)
			return
		}
		qOrg := r.URL.Query().Get("org_id")
		var wa *store.WalletAccount
		var err error
		if qOrg != "" {
			oid, _ := strconv.ParseUint(qOrg, 10, 64)
			if !canAccessOrg(st, r, oid) {
				envelope.Err(w, r, http.StatusForbidden, 40301, "NOT_MEMBER", nil)
				return
			}
			wa, err = st.EnsureWallet(r.Context(), "org", oid)
		} else {
			wa, err = st.EnsureWallet(r.Context(), "user", cl.UserID)
		}
		if err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50001, "WALLET_ERROR", nil)
			return
		}
		envelope.OK(w, r, map[string]any{
			"balance_cents":  wa.BalanceCents,
			"currency":       wa.Currency,
			"credit_status":  wa.Status,
		})
	}
}

func recharge(cfg *config.Config, st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cl, ok := httpserver.ClaimsFrom(r.Context())
		if !ok {
			envelope.Err(w, r, http.StatusUnauthorized, 40101, "UNAUTHORIZED", nil)
			return
		}
		var body struct {
			AmountCents int64   `json:"amount_cents"`
			Channel       string  `json:"channel"`
			OrgID         *uint64 `json:"org_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.AmountCents < 100 {
			envelope.Err(w, r, http.StatusBadRequest, 40001, "BAD_JSON", nil)
			return
		}
		ch := body.Channel
		if ch == "" {
			ch = "wechat"
		}
		var oid *uint64
		if body.OrgID != nil {
			if !canAccessOrg(st, r, *body.OrgID) {
				envelope.Err(w, r, http.StatusForbidden, 40301, "NOT_MEMBER", nil)
				return
			}
			oid = body.OrgID
		}
		orderID, err := st.CreateOrder(r.Context(), cl.UserID, oid, body.AmountCents, ch)
		if err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50001, "ORDER_CREATE_FAILED", nil)
			return
		}
		params := map[string]any{"mode": cfg.Payment.Mode}
		if cfg.Payment.Mode == "mock" {
			params["hint"] = "POST /mlk/platform/v1/payment/mock/complete with order_id"
		}
		envelope.OK(w, r, map[string]any{
			"order_id":         strconv.FormatUint(orderID, 10),
			"payment_params": params,
		})
	}
}

func getOrder(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cl, ok := httpserver.ClaimsFrom(r.Context())
		if !ok {
			envelope.Err(w, r, http.StatusUnauthorized, 40101, "UNAUTHORIZED", nil)
			return
		}
		id, err := strconv.ParseUint(chi.URLParam(r, "order_id"), 10, 64)
		if err != nil {
			envelope.Err(w, r, http.StatusBadRequest, 40001, "BAD_ID", nil)
			return
		}
		o, err := st.GetOrder(r.Context(), id)
		if err != nil || o == nil || o.UserID != cl.UserID {
			envelope.Err(w, r, http.StatusNotFound, 40401, "NOT_FOUND", nil)
			return
		}
		out := map[string]any{
			"id": o.ID, "status": o.Status, "amount_cents": o.AmountCents, "channel": o.Channel,
			"order_type": o.OrderType, "currency": o.Currency,
			"created_at": o.CreatedAt.UTC().Format(time.RFC3339Nano),
		}
		if o.OrgID != nil {
			out["org_id"] = *o.OrgID
		}
		if o.PaidAt.Valid {
			out["paid_at"] = o.PaidAt.Time.UTC().Format(time.RFC3339Nano)
		}
		if o.ProviderTradeNo.Valid {
			out["provider_trade_no"] = o.ProviderTradeNo.String
		}
		envelope.OK(w, r, out)
	}
}

func listOrders(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cl, ok := httpserver.ClaimsFrom(r.Context())
		if !ok {
			envelope.Err(w, r, http.StatusUnauthorized, 40101, "UNAUTHORIZED", nil)
			return
		}
		rows, err := st.DB.QueryContext(r.Context(),
			`SELECT id, org_id, order_type, amount_cents, currency, channel, status, created_at, paid_at
			 FROM orders WHERE user_id = ? ORDER BY id DESC LIMIT 50`, cl.UserID)
		if err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50001, "QUERY_FAILED", nil)
			return
		}
		defer rows.Close()
		var items []map[string]any
		for rows.Next() {
			var id uint64
			var org sql.NullInt64
			var orderType, ch, stt, curr string
			var amt int64
			var ct time.Time
			var paid sql.NullTime
			if err := rows.Scan(&id, &org, &orderType, &amt, &curr, &ch, &stt, &ct, &paid); err != nil {
				continue
			}
			row := map[string]any{
				"id": id, "order_type": orderType, "amount_cents": amt, "currency": curr,
				"channel": ch, "status": stt, "created_at": ct.UTC().Format(time.RFC3339Nano),
			}
			if org.Valid {
				row["org_id"] = uint64(org.Int64)
			}
			if paid.Valid {
				row["paid_at"] = paid.Time.UTC().Format(time.RFC3339Nano)
			}
			items = append(items, row)
		}
		envelope.OK(w, r, map[string]any{"items": items})
	}
}

func mockPay(cfg *config.Config, st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.Payment.Mode != "mock" {
			envelope.Err(w, r, http.StatusNotFound, 40401, "NOT_AVAILABLE", nil)
			return
		}
		var body struct {
			OrderID string `json:"order_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.OrderID == "" {
			envelope.Err(w, r, http.StatusBadRequest, 40001, "BAD_JSON", nil)
			return
		}
		oid, err := strconv.ParseUint(body.OrderID, 10, 64)
		if err != nil {
			envelope.Err(w, r, http.StatusBadRequest, 40002, "BAD_ORDER_ID", nil)
			return
		}
		order, err := st.GetOrder(r.Context(), oid)
		if err != nil || order == nil {
			envelope.Err(w, r, http.StatusNotFound, 40401, "NOT_FOUND", nil)
			return
		}
		if order.Status != "pending" {
			envelope.OK(w, r, map[string]any{"ok": true, "already_paid": true})
			return
		}
		if err := st.MarkOrderPaid(r.Context(), oid, "mock-"+body.OrderID); err != nil {
			envelope.Err(w, r, http.StatusBadRequest, 40003, "PAY_FAILED", map[string]any{"error": err.Error()})
			return
		}
		var wa *store.WalletAccount
		if order.OrgID != nil {
			wa, err = st.EnsureWallet(r.Context(), "org", *order.OrgID)
		} else {
			wa, err = st.EnsureWallet(r.Context(), "user", order.UserID)
		}
		if err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50001, "WALLET_ERROR", nil)
			return
		}
		if err := st.Credit(r.Context(), wa.ID, order.AmountCents, "recharge", "order", oid, "mock payment"); err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50002, "CREDIT_FAILED", map[string]any{"error": err.Error()})
			return
		}
		envelope.OK(w, r, map[string]any{"ok": true})
	}
}

func usageSummary(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cl, ok := httpserver.ClaimsFrom(r.Context())
		if !ok {
			envelope.Err(w, r, http.StatusUnauthorized, 40101, "UNAUTHORIZED", nil)
			return
		}
		qOrg := r.URL.Query().Get("org_id")
		var wa *store.WalletAccount
		var err error
		if qOrg != "" {
			oid, _ := strconv.ParseUint(qOrg, 10, 64)
			if !canAccessOrg(st, r, oid) {
				envelope.Err(w, r, http.StatusForbidden, 40301, "NOT_MEMBER", nil)
				return
			}
			wa, err = st.EnsureWallet(r.Context(), "org", oid)
		} else {
			wa, err = st.EnsureWallet(r.Context(), "user", cl.UserID)
		}
		if err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50001, "WALLET_ERROR", nil)
			return
		}
		days := 30
		calls, cost, err := st.UsageSummary(r.Context(), wa.ID, days)
		if err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50002, "USAGE_FAILED", nil)
			return
		}
		envelope.OK(w, r, map[string]any{
			"period_days":        days,
			"inference_calls":    calls,
			"total_cost_cents":   cost,
		})
	}
}

func adminUsers(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := st.AdminListUsers(r.Context(), 200)
		if err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50001, "LIST_FAILED", nil)
			return
		}
		items := make([]map[string]any, 0, len(list))
		for _, u := range list {
			items = append(items, map[string]any{
				"id": u.ID, "email": u.Email, "display_name": u.DisplayName, "role": u.Role, "status": u.Status,
				"created_at": u.CreatedAt.UTC().Format(time.RFC3339Nano),
			})
		}
		envelope.OK(w, r, map[string]any{"items": items})
	}
}

func adminUserDetail(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "user_id")
		uid, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil || uid == 0 {
			envelope.Err(w, r, http.StatusBadRequest, 40002, "INVALID_USER_ID", nil)
			return
		}
		u, err := st.AdminGetUserDetail(r.Context(), uid)
		if err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50001, "QUERY_FAILED", nil)
			return
		}
		if u == nil {
			envelope.Err(w, r, http.StatusNotFound, 40401, "USER_NOT_FOUND", nil)
			return
		}
		keys, err := st.AdminListAPIKeysForUser(r.Context(), uid)
		if err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50002, "KEYS_QUERY_FAILED", nil)
			return
		}
		wallet, err := st.AdminGetUserWallet(r.Context(), uid)
		if err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50003, "WALLET_QUERY_FAILED", nil)
			return
		}
		keyItems := make([]map[string]any, 0, len(keys))
		for _, k := range keys {
			row := map[string]any{
				"id": k.ID, "scope": k.Scope, "name": k.Name, "key_prefix": k.KeyPrefix,
				"status": k.Status, "created_at": k.CreatedAt.UTC().Format(time.RFC3339Nano),
			}
			if k.OrgID != nil {
				row["org_id"] = *k.OrgID
			}
			if k.LastUsedAt.Valid {
				row["last_used_at"] = k.LastUsedAt.Time.UTC().Format(time.RFC3339Nano)
			}
			keyItems = append(keyItems, row)
		}
		userOut := map[string]any{
			"id": u.ID, "role": u.Role, "status": u.Status,
			"created_at": u.CreatedAt.UTC().Format(time.RFC3339Nano),
		}
		if u.Email.Valid {
			userOut["email"] = u.Email.String
		}
		if u.Phone.Valid {
			userOut["phone"] = u.Phone.String
		}
		if u.DisplayName.Valid {
			userOut["display_name"] = u.DisplayName.String
		}
		if u.AvatarURL.Valid {
			userOut["avatar_url"] = u.AvatarURL.String
		}
		if u.LastLoginAt.Valid {
			userOut["last_login_at"] = u.LastLoginAt.Time.UTC().Format(time.RFC3339Nano)
		}
		out := map[string]any{"user": userOut, "api_keys": keyItems}
		if wallet != nil {
			out["wallet"] = map[string]any{
				"balance_cents": wallet.BalanceCents, "currency": wallet.Currency, "status": wallet.Status,
			}
		} else {
			out["wallet"] = nil
		}
		out["note"] = "API Key 仅展示前缀；完整密钥仅在用户创建时返回一次，管理员无法查看明文。"
		envelope.OK(w, r, out)
	}
}

func adminChannels(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := st.DB.QueryContext(r.Context(), `SELECT id, name, channel_type, base_url, status, api_key_cipher FROM channels ORDER BY id`)
		if err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50001, "QUERY_FAILED", nil)
			return
		}
		defer rows.Close()
		var items []map[string]any
		for rows.Next() {
			var id uint64
			var name, ctype, base, stt, cipher string
			if err := rows.Scan(&id, &name, &ctype, &base, &stt, &cipher); err != nil {
				continue
			}
			apiKeySet := false
			if k, err := store.DecodeChannelAPIKey(cipher); err == nil && strings.TrimSpace(k) != "" {
				apiKeySet = true
			}
			items = append(items, map[string]any{
				"id": id, "name": name, "type": ctype, "base_url": base, "status": stt,
				"api_key_set": apiKeySet,
			})
		}
		envelope.OK(w, r, map[string]any{"items": items})
	}
}

// adminGetChannel returns channel metadata and decoded upstream api_key (admin only).
func adminGetChannel(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseUint(chi.URLParam(r, "channel_id"), 10, 64)
		if err != nil || id == 0 {
			envelope.Err(w, r, http.StatusBadRequest, 40001, "BAD_CHANNEL_ID", nil)
			return
		}
		ch, err := st.GetChannel(r.Context(), id)
		if err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50001, "QUERY_FAILED", nil)
			return
		}
		if ch == nil {
			envelope.Err(w, r, http.StatusNotFound, 40401, "CHANNEL_NOT_FOUND", nil)
			return
		}
		apiKey := ""
		if k, err := store.DecodeChannelAPIKey(ch.APIKeyCipher); err == nil {
			apiKey = strings.TrimSpace(k)
		}
		envelope.OK(w, r, map[string]any{
			"id": ch.ID, "name": ch.Name, "type": ch.ChannelType,
			"base_url": ch.BaseURL, "status": ch.Status,
			"api_key":     apiKey,
			"api_key_set": apiKey != "",
		})
	}
}

func adminPatchChannel(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseUint(chi.URLParam(r, "channel_id"), 10, 64)
		if err != nil || id == 0 {
			envelope.Err(w, r, http.StatusBadRequest, 40001, "BAD_CHANNEL_ID", nil)
			return
		}
		ch, err := st.GetChannel(r.Context(), id)
		if err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50001, "QUERY_FAILED", nil)
			return
		}
		if ch == nil {
			envelope.Err(w, r, http.StatusNotFound, 40401, "CHANNEL_NOT_FOUND", nil)
			return
		}
		var body struct {
			APIKey string `json:"api_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			envelope.Err(w, r, http.StatusBadRequest, 40001, "BAD_JSON", nil)
			return
		}
		key := strings.TrimSpace(body.APIKey)
		if key == "" {
			envelope.Err(w, r, http.StatusBadRequest, 40002, "API_KEY_REQUIRED", nil)
			return
		}
		cipher := store.EncodeChannelAPIKeyPlain(key)
		if err := st.UpdateChannelAPIKeyCipher(r.Context(), id, cipher); err != nil {
			if errors.Is(err, store.ErrChannelNotFound) {
				envelope.Err(w, r, http.StatusNotFound, 40401, "CHANNEL_NOT_FOUND", nil)
				return
			}
			envelope.Err(w, r, http.StatusInternalServerError, 50001, "UPDATE_FAILED", map[string]any{"error": err.Error()})
			return
		}
		envelope.OK(w, r, map[string]any{"ok": true})
	}
}

func adminModels(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := st.DB.QueryContext(r.Context(), `SELECT id, model_id, display_name, enabled FROM platform_models ORDER BY id`)
		if err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50001, "QUERY_FAILED", nil)
			return
		}
		defer rows.Close()
		var items []map[string]any
		for rows.Next() {
			var id uint64
			var mid, dn string
			var en int
			if err := rows.Scan(&id, &mid, &dn, &en); err != nil {
				continue
			}
			items = append(items, map[string]any{"id": id, "model_id": mid, "display_name": dn, "enabled": en == 1})
		}
		envelope.OK(w, r, map[string]any{"items": items})
	}
}

func adminCreateModel(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ModelID           string `json:"model_id"`
			DisplayName       string `json:"display_name"`
			UpstreamModelID   string `json:"upstream_model_id"`
			ChannelID         uint64 `json:"channel_id"`
			InputPer1kCents   int64  `json:"input_per_1k_cents"`
			OutputPer1kCents  int64  `json:"output_per_1k_cents"`
			Enabled           *bool  `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			envelope.Err(w, r, http.StatusBadRequest, 40001, "BAD_JSON", nil)
			return
		}
		body.ModelID = strings.TrimSpace(body.ModelID)
		if body.ModelID == "" {
			envelope.Err(w, r, http.StatusBadRequest, 40002, "MODEL_ID_REQUIRED", nil)
			return
		}
		enabled := true
		if body.Enabled != nil {
			enabled = *body.Enabled
		}
		err := st.CreatePlatformModelBundle(r.Context(), body.ModelID, strings.TrimSpace(body.DisplayName), strings.TrimSpace(body.UpstreamModelID), body.ChannelID, body.InputPer1kCents, body.OutputPer1kCents, enabled)
		if err != nil {
			switch {
			case errors.Is(err, store.ErrDuplicateModel):
				envelope.Err(w, r, http.StatusConflict, 40901, "DUPLICATE_MODEL", nil)
			case errors.Is(err, store.ErrChannelNotFound):
				envelope.Err(w, r, http.StatusBadRequest, 40003, "CHANNEL_NOT_FOUND", nil)
			case strings.Contains(err.Error(), "model_id required"), strings.Contains(err.Error(), "pricing must be non-negative"):
				envelope.Err(w, r, http.StatusBadRequest, 40002, "INVALID_PARAMS", map[string]any{"error": err.Error()})
			default:
				envelope.Err(w, r, http.StatusInternalServerError, 50001, "CREATE_MODEL_FAILED", map[string]any{"error": err.Error()})
			}
			return
		}
		envelope.OK(w, r, map[string]any{"ok": true, "model_id": body.ModelID})
	}
}
