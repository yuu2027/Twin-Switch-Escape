package repository

import (
	"sort"
	"sync"
	"time"
	"twin-switch-escape/server/internal/models"
)

// MatchRepository は試合結果の永続化の抽象。
// ランキング（spec §15）と /api/me の集計（spec §12.1）の元データを提供する。
//
// 設計意図: 専用のランキングテーブルは持たず、matches から都度導出する（spec §12.2 / §15）。
type MatchRepository interface {
	// Create は試合結果を1件保存する（Phase 6 のクリア確定時に使う想定）。
	Create(m *models.Match) error

	// ListClearedRankedByTime は result="cleared" を ClearTimeMs 昇順で返す（spec §15.1）。
	// limit <= 0 なら全件。
	ListClearedRankedByTime(limit int) ([]*models.Match, error)

	// ListByUserID は指定ユーザーが参加した試合を新しい順で返す（spec §7.4 /api/matches/me 用）。
	ListByUserID(userID string) ([]*models.Match, error)

	// Stats は /api/me 用の集計（spec §7.1, §12.1）。
	//   bestClearTimeMs: 当該ユーザーが参加した cleared 試合の最小 ClearTimeMs。cleared が無ければ 0。
	//   clearCount:      当該ユーザーが参加した cleared 試合数。
	Stats(userID string) (bestClearTimeMs int, clearCount int, err error)
}

// InMemoryMatchRepository は slice ベースのインメモリ実装。
type InMemoryMatchRepository struct {
	mu      sync.RWMutex
	matches []*models.Match
}

// NewInMemoryMatchRepository は空のリポジトリを返す。
// デモデータの投入は Seed か、main から SeedDemoMatches を呼ぶ形にする。
//
// 構造体を初期化して返す。
func NewInMemoryMatchRepository() *InMemoryMatchRepository {
	return &InMemoryMatchRepository{
		matches: make([]*models.Match, 0),
	}
}

// Lock 下で matches へ append。
func (r *InMemoryMatchRepository) Create(m *models.Match) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.matches = append(r.matches, m)

	return nil
}

//  1. RLock 下で cleared だけを別スライスに集める。
//  2. sort.Slice で ClearTimeMs 昇順に並べる。
//  3. limit > 0 なら先頭 limit 件に切る。
func (r *InMemoryMatchRepository) ListClearedRankedByTime(limit int) ([]*models.Match, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cleared := make([]*models.Match, 0)

	for _, m := range r.matches {
		if m.Result == "cleared" {
			cleared = append(cleared, m)
		}
	}

	sort.Slice(cleared, func(i, j int) bool {
		return cleared[i].ClearTimeMs < cleared[j].ClearTimeMs
	})

	if limit > 0 && len(cleared) > limit {
		cleared = cleared[:limit]
	}

	return cleared, nil
}

// PlayerIDs に userID を含む試合を集め、EndedAt 降順で返す。
func (r *InMemoryMatchRepository) ListByUserID(userID string) ([]*models.Match, error) {
	matches := make([]*models.Match, 0)

	for _, m := range r.matches {
		for _, playerID := range m.PlayerIDs {
			if playerID == userID {
				matches = append(matches, m)
				break
			}
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].EndedAt.After(matches[j].EndedAt)
	})

	return matches, nil
}

// Stats。
//
//  1. ListByUserID 相当で当該ユーザーの cleared 試合を走査。
//  2. 件数を数え、ClearTimeMs の最小値を求める。
//  3. cleared が無ければ best=0, count=0。
func (r *InMemoryMatchRepository) Stats(userID string) (int, int, error) {
	matches, err := r.ListByUserID(userID)
	if err != nil {
		return 0, 0, err
	}

	bestClearTimeMs := 0
	clearCount := 0

	for _, m := range matches {
		if m.Result == "cleared" {

			clearCount++

			if bestClearTimeMs == 0 {
				bestClearTimeMs = m.ClearTimeMs
			} else {
				bestClearTimeMs = min(bestClearTimeMs, m.ClearTimeMs)
			}
		}
	}

	return bestClearTimeMs, clearCount, nil
}

// SeedDemoMatches はランキングが空にならないようデモ試合を数件投入する（Phase 2 限定の便宜）。
//
//   - "player001"/"player002" などの userId（実際には register 済みユーザーの id）で
//     result="cleared", ClearTimeMs を 72400, 81700 などにした Match を 2〜3 件 Create する。
//   - 注意: ランキング表示は player 名が要る（spec §15.3）。名前解決のために、
//     ここでは PlayerIDs に「表示名」を入れるか、別途 username を引けるよう設計を決める。
//     → 簡単のため Phase 2 デモでは PlayerIDs にそのまま表示名を入れてもよい（後で整理）。
func (r *InMemoryMatchRepository) SeedDemoMatches() {
	now := time.Now()

	_ = r.Create(&models.Match{
		ID:          "demo_match_001",
		RoomID:      "demo_room_001",
		Result:      "cleared",
		ClearTimeMs: 72400,
		PlayerIDs:   []string{"player001", "player002"},
		StartedAt:   now.Add(-30 * time.Minute),
		EndedAt:     now.Add(-30*time.Minute + 72400*time.Millisecond),
		CreatedAt:   now,
	})

	_ = r.Create(&models.Match{
		ID:          "demo_match_002",
		RoomID:      "demo_room_002",
		Result:      "cleared",
		ClearTimeMs: 81700,
		PlayerIDs:   []string{"player003", "player004"},
		StartedAt:   now.Add(-20 * time.Minute),
		EndedAt:     now.Add(-20*time.Minute + 81700*time.Millisecond),
		CreatedAt:   now,
	})

	_ = r.Create(&models.Match{
		ID:          "demo_match_003",
		RoomID:      "demo_room_003",
		Result:      "cleared",
		ClearTimeMs: 95600,
		PlayerIDs:   []string{"player005", "player006"},
		StartedAt:   now.Add(-10 * time.Minute),
		EndedAt:     now.Add(-10*time.Minute + 95600*time.Millisecond),
		CreatedAt:   now,
	})
}

var _ MatchRepository = (*InMemoryMatchRepository)(nil)
