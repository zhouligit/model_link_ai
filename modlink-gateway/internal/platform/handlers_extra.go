package platform

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/modlinkcloud/modlink-gateway/internal/config"
	"github.com/modlinkcloud/modlink-gateway/internal/httpserver"
	"github.com/modlinkcloud/modlink-gateway/internal/shared/envelope"
	"github.com/modlinkcloud/modlink-gateway/internal/store"
)

func listOrgs(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cl, ok := httpserver.ClaimsFrom(r.Context())
		if !ok {
			envelope.Err(w, r, http.StatusUnauthorized, 40101, "UNAUTHORIZED", nil)
			return
		}
		orgs, err := st.ListUserOrgs(r.Context(), cl.UserID)
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
		if _, ok2, _ := st.IsOrgMember(r.Context(), oid, cl.UserID); !ok2 {
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
		if _, ok2, _ := st.IsOrgMember(r.Context(), oid, cl.UserID); !ok2 {
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
			items = append(items, map[string]any{
				"id": k.ID, "name": k.Name, "scope": k.Scope, "key_prefix": k.KeyPrefix, "status": k.Status,
			})
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
			Test  bool   `json:"test"`
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
			if _, okm, _ := st.IsOrgMember(r.Context(), *body.OrgID, cl.UserID); !okm {
				envelope.Err(w, r, http.StatusForbidden, 40301, "NOT_MEMBER", nil)
				return
			}
			orgPtr = body.OrgID
		}
		full, prefix, hash, err := store.GenerateAPIKey(body.Test)
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
			if _, ok2, _ := st.IsOrgMember(r.Context(), oid, cl.UserID); !ok2 {
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
			if _, ok2, _ := st.IsOrgMember(r.Context(), *body.OrgID, cl.UserID); !ok2 {
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
		envelope.OK(w, r, map[string]any{
			"id": o.ID, "status": o.Status, "amount_cents": o.AmountCents, "channel": o.Channel,
		})
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
			`SELECT id, amount_cents, channel, status, created_at FROM orders WHERE user_id = ? ORDER BY id DESC LIMIT 50`, cl.UserID)
		if err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50001, "QUERY_FAILED", nil)
			return
		}
		defer rows.Close()
		var items []map[string]any
		for rows.Next() {
			var id uint64
			var amt int64
			var ch, stt string
			var ct interface{}
			if err := rows.Scan(&id, &amt, &ch, &stt, &ct); err != nil {
				continue
			}
			items = append(items, map[string]any{"id": id, "amount_cents": amt, "channel": ch, "status": stt})
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
			if _, ok2, _ := st.IsOrgMember(r.Context(), oid, cl.UserID); !ok2 {
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
			})
		}
		envelope.OK(w, r, map[string]any{"items": items})
	}
}

func adminChannels(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := st.DB.QueryContext(r.Context(), `SELECT id, name, channel_type, base_url, status FROM channels ORDER BY id`)
		if err != nil {
			envelope.Err(w, r, http.StatusInternalServerError, 50001, "QUERY_FAILED", nil)
			return
		}
		defer rows.Close()
		var items []map[string]any
		for rows.Next() {
			var id uint64
			var name, ctype, base, stt string
			if err := rows.Scan(&id, &name, &ctype, &base, &stt); err != nil {
				continue
			}
			items = append(items, map[string]any{"id": id, "name": name, "type": ctype, "base_url": base, "status": stt})
		}
		envelope.OK(w, r, map[string]any{"items": items})
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
