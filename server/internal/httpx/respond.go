// Package httpx は JSON レスポンスとエラーレスポンスの共通ヘルパーを提供する。
// エラー形式・ステータスは spec §16 に統一する。
package httpx

import (
	"encoding/json"
	"net/http"
)

// REST では spec §16.1 のステータスコード（400/401/409/500 等）と
// この body を組み合わせて返す。
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// 1. w.Header().Set("Content-Type", "application/json")
// 2. w.WriteHeader(status)
// 3. json.NewEncoder(w).Encode(v) でエンコード。エラーは握りつぶさずログだけ出してよい。
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status) // HTTPステータスをクライアントへ送る
	json.NewEncoder(w).Encode(v)
}

// WriteJSON(w, status, ErrorBody{Code: code, Message: message}) を呼ぶだけ。
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorBody{Code: code, Message: message})
}
