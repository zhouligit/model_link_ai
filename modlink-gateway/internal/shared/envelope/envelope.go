package envelope

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

type Body struct {
	Code      int             `json:"code"`
	Message   string          `json:"message"`
	Data      any             `json:"data,omitempty"`
	Detail    any             `json:"detail,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
}

func JSON(w http.ResponseWriter, r *http.Request, status int, b Body) {
	if b.RequestID == "" {
		b.RequestID = middleware.GetReqID(r.Context())
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(b)
}

func OK(w http.ResponseWriter, r *http.Request, data any) {
	JSON(w, r, http.StatusOK, Body{Code: 0, Message: "ok", Data: data})
}

func Err(w http.ResponseWriter, r *http.Request, status int, code int, msg string, detail any) {
	JSON(w, r, status, Body{Code: code, Message: msg, Detail: detail})
}
