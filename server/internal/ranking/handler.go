package ranking

import (
	"net/http"
	"strconv"

	"twin-switch-escape/server/internal/httpx"
)

// Handler はランキング取得エンドポイント。認証不要（spec §7.4）。
type Handler struct {
	svc *Service
}

// *Handler{svc: svc} を返す。
func NewHandler(svc *Service) *Handler {
	return &Handler{
		svc: svc,
	}
}

// rankingResponse は spec §7.4 のレスポンス形 {"rankings": [...]}。
type rankingResponse struct {
	Rankings []Entry `json:"rankings"`
}

// Get: GET /api/ranking（spec §7.4）。
//  1. limit を決める（例: 上位 50。クエリ ?limit= を読むなら strconv.Atoi）。
//  2. svc.Top(limit)。エラー→ httpx.WriteError(w, 500, ...)。
//  3. 成功→ httpx.WriteJSON(w, 200, rankingResponse{Rankings: entries})。
//     entries が nil の場合は空配列 []Entry{} にしておくと JSON が null にならず親切。
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	limit := 50

	limitStr := r.URL.Query().Get("limit") // クライアントからの要求
	if limitStr != "" {
		n, err := strconv.Atoi(limitStr) // 整数に変換
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_LIMIT", "limit must be a number")
			return
		}
		limit = n
	}

	entries, err := h.svc.Top(limit)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get ranking")
		return
	}

	if entries == nil {
		entries = []Entry{}
	}

	httpx.WriteJSON(w, http.StatusOK, rankingResponse{
		Rankings: entries,
	})
}
