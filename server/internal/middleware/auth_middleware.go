// Package middleware は HTTP ミドルウェアを提供する。
package middleware

import (
	"context"
	"net/http"
	"strings"

	"twin-switch-escape/server/internal/httpx"
)

// TokenParser は JWT などのトークンを検証して userID を返すもの。
type TokenParser interface {
	Parse(token string) (string, error)
}

// contextKey は context のキー衝突を避けるための非公開型（Go の定番イディオム）。
type contextKey string

const userIDKey contextKey = "userId"

// Authorization: Bearer トークンを検証するミドルウェア。
// 検証成功時のみ next を呼び、context に userId を載せる（spec §16.3 INVALID_TOKEN）。
//  1. r.Header.Get("Authorization") を取得。"Bearer " で始まらなければ 401 "INVALID_TOKEN"。
//  2. prefix を除いてトークン文字列を取り出す（strings.TrimPrefix / strings.CutPrefix）。
//  3. issuer.Parse(token) で userID を得る。失敗→401 "INVALID_TOKEN"。
//  4. ctx := context.WithValue(r.Context(), userIDKey, userID)
//  5. next.ServeHTTP(w, r.WithContext(ctx)) を呼ぶ。
//
// ミドルウェアは「http.Handler を受け取り http.Handler を返す」関数として書くと
// ServeMux と組み合わせやすい。
func RequireAuth(issuer TokenParser, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization") // ヘッダーを取得

		tokenStr, ok := strings.CutPrefix(authHeader, "Bearer ") // 文字列の先頭
		if !ok || tokenStr == "" {
			httpx.WriteError(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid token")
			return
		}

		userID, err := issuer.Parse(tokenStr)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid token")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID) // contextに書き込む
		next.ServeHTTP(w, r.WithContext(ctx))                    // 次のハンドに渡す
	})
}

// RequireAuth が格納した userId を取り出す。
// ctx.Value(userIDKey) を string へ型アサートして返す。無ければ ""。
func UserIDFromContext(ctx context.Context) string {
	userID, ok := ctx.Value(userIDKey).(string) // 文字列型でcontextに入れたuserIDを取得
	if !ok {
		return ""
	}

	return userID
}
