package matchmaking

import (
	"errors"
	"net/http"

	"twin-switch-escape/server/internal/httpx"
	"twin-switch-escape/server/internal/middleware"
)

// Handler はマッチング系エンドポイント（すべて要認証, spec §7.3）。
type Handler struct {
	mgr *Manager
}

// NewHandler。
func NewHandler(mgr *Manager) *Handler {
	return &Handler{mgr: mgr}
}

// --- レスポンス DTO（spec §7.3 の JSON 形）---

type startResponse struct {
	Status        string `json:"status"`
	MatchmakingID string `json:"matchmakingId"`
}

type statusResponse struct {
	Status string `json:"status"`
	// omitempty: waiting のときは roomId / websocketUrl を出さない（spec §7.3）。
	RoomID       string `json:"roomId,omitempty"`
	WebSocketURL string `json:"websocketUrl,omitempty"`
}

type cancelResponse struct {
	Status string `json:"status"`
}

// Start: POST /api/matchmaking/start（spec §7.3）。
//
// 手順:
//  1. userID := middleware.UserIDFromContext(r.Context())。空なら 401 "INVALID_TOKEN"。
//  2. res, err := h.mgr.Start(userID)。
//  3. errors.Is(err, ErrAlreadyMatching) → 409 "ALREADY_MATCHING"。その他の err → 500。
//  4. 成功 → 200 startResponse{Status: string(res.Status), MatchmakingID: res.MatchmakingID}。
func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid token")
		return
	}

	res, err := h.mgr.Start(userID)
	if err != nil {
		if errors.Is(err, ErrAlreadyMatching) {
			httpx.WriteError(w, http.StatusConflict, "ALREADY_MATCHING", "already in matchmaking")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, startResponse{
		Status:        string(res.Status),
		MatchmakingID: res.MatchmakingID,
	})
}

// Status: GET /api/matchmaking/status（spec §7.3）。
//
// 手順:
//  1. userID を取り出す（空なら 401）。
//  2. res, err := h.mgr.Status(userID)。err → 500。
//  3. 200 statusResponse{Status, RoomID, WebSocketURL}（waiting のときは後2つ空 → omitempty で出ない）。
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid token")
		return
	}

	res, err := h.mgr.Status(userID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, statusResponse{
		Status:       string(res.Status),
		RoomID:       res.RoomID,
		WebSocketURL: res.WebSocketURL,
	})
}

// Cancel: POST /api/matchmaking/cancel（spec §7.3）。
//
// 手順:
//  1. userID を取り出す（空なら 401）。
//  2. res, err := h.mgr.Cancel(userID)。err → 500。
//  3. 200 cancelResponse{Status: string(res.Status)}。
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid token")
		return
	}

	res, err := h.mgr.Cancel(userID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, cancelResponse{
		Status: string(res.Status),
	})
}
