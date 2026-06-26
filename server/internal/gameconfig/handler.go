// Package gameconfig は GET /api/game-config を提供する（spec §7.2）。認証不要。
package gameconfig

import (
	"net/http"

	"twin-switch-escape/server/internal/config"
	"twin-switch-escape/server/internal/httpx"
)

// config.GameConfig をそのまま返すだけの単純なハンドラ。
type Handler struct {
	cfg config.GameConfig
}

// *Handler{cfg: cfg} を返す。
func NewHandler(cfg config.GameConfig) *Handler {
	return &Handler{
		cfg: cfg,
	}
}

// Get: GET /api/game-config。
// httpx.WriteJSON(w, 200, h.cfg) を呼ぶだけ。
// config.GameConfig は spec §7.2 の JSON タグ付きなので、そのまま返せる。
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, 200, h.cfg)
}
