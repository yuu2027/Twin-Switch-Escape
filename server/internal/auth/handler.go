package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"twin-switch-escape/server/internal/httpx"
	"twin-switch-escape/server/internal/middleware"
	// TODO: 実装時に有効化:
)

// Handler は認証系エンドポイントの HTTP ハンドラ群。
type Handler struct {
	svc *Service
}

// *Handler{svc: svc} を返す。
func NewHandler(svc *Service) *Handler {
	return &Handler{
		svc: svc,
	}
}

// --- リクエスト/レスポンス DTO（spec §7.1 のJSON形に対応） ---

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type registerResponse struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string `json:"accessToken"`
	UserID      string `json:"userId"`
	Username    string `json:"username"`
}

type meResponse struct {
	UserID        string  `json:"userId"`
	Username      string  `json:"username"`
	BestClearTime float64 `json:"bestClearTime"`
	ClearCount    int     `json:"clearCount"`
}

// Register: POST /api/register（spec §7.1）。
//  1. json.NewDecoder(r.Body).Decode(&req)。失敗→ httpx.WriteError(w, 400, "INVALID_REQUEST", ...)。
//  2. svc.Register(req.Username, req.Password)。
//  3. errors.Is で分岐:
//     ErrInvalidInput  → 400
//     ErrUsernameTaken → 409 "USERNAME_TAKEN"
//     その他           → 500
//  4. 成功→ httpx.WriteJSON(w, 200, registerResponse{...})。
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	userID, userName, err := h.svc.Register(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request input") // 400
			return
		}
		if errors.Is(err, ErrUsernameTaken) {
			httpx.WriteError(w, http.StatusConflict, "USERNAME_TAKEN", "username is already taken") // 409
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error") // 500
		return
	}

	httpx.WriteJSON(w, http.StatusOK, registerResponse{
		UserID:   userID,
		Username: userName,
	})
}

// Login: POST /api/login（spec §7.1）。
//  1. body をデコード。
//  2. svc.Login(...)。ErrInvalidCredentials → 401 "INVALID_CREDENTIALS"、その他→500。
//  3. 成功→ 200 loginResponse{accessToken, userId, username}。
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body") // 400
		return
	}

	token, userID, userName, err := h.svc.Login(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.WriteError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid username or password") // 401
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, loginResponse{
		AccessToken: token,
		UserID:      userID,
		Username:    userName,
	})
}

// Me: GET /api/me（要認証, spec §7.1）。
// 認証ミドルウェアが context に入れた userId を取り出して使う。
//  1. userID := middleware.UserIDFromContext(r.Context())。空なら 401。
//  2. svc.Me(userID)。エラー→500（または401）。
//  3. 成功→ 200 meResponse{...}。
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "INTERNAL_ERROR", "internal server error") // 401
		return
	}

	userName, bestClearTimeSec, clearCount, err := h.svc.Me(userID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, meResponse{
		UserID:        userID,
		Username:      userName,
		BestClearTime: bestClearTimeSec,
		ClearCount:    clearCount,
	})
}
