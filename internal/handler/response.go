package handler

import (
	"encoding/json"
	"net/http"
)

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type meta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

type successEnvelope struct {
	Data any   `json:"data"`
	Meta *meta `json:"meta,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code string, msg string) {
	writeJSON(w, status, errorEnvelope{Error: errorBody{Code: code, Message: msg}})
}

func writeSuccess(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, successEnvelope{Data: data})
}

func writeSuccessWithMeta(w http.ResponseWriter, status int, data any, page, perPage, total int) {
	writeJSON(w, status, successEnvelope{Data: data, Meta: &meta{Page: page, PerPage: perPage, Total: total}})
}
