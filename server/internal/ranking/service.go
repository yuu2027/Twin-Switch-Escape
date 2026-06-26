// Package ranking はランキング取得（spec §7.4 / §15）を提供する。
// 専用テーブルを持たず matches から導出する（spec §12.2）。
package ranking

import (
	"time"
	"twin-switch-escape/server/internal/repository"
)

// Entry は spec §15.3 の表示項目。
type Entry struct {
	Rank      int      `json:"rank"`
	Players   []string `json:"players"`   // 2人のプレイヤー名（spec §15.3）
	ClearTime float64  `json:"clearTime"` // 秒(小数)。内部はミリ秒整数→秒へ変換（spec §9.7）
	PlayedAt  string   `json:"playedAt"`  // RFC3339（例: time.RFC3339）
}

// Service はランキング集計ロジック。
type Service struct {
	matches repository.MatchRepository
	// 本来は userId→username 解決のため UserRepository も要る。
	// Phase 2 デモでは Match.PlayerIDs に表示名を入れて簡略化する場合は不要（設計判断）。
}

// *Service{matches: matches} を返す。
func NewService(matches repository.MatchRepository) *Service {
	return &Service{
		matches: matches,
	}
}

// 上位 limit 件のランキングを返す（spec §15.1: クリアタイム昇順）。
//  1. matches.ListClearedRankedByTime(limit) を呼ぶ（既に昇順）。
//  2. 各 Match を Entry へ変換:
//     Rank = index+1
//     Players = m.PlayerIDs（表示名）
//     ClearTime = float64(m.ClearTimeMs) / 1000.0
//     PlayedAt = m.EndedAt.Format(time.RFC3339)
//  3. []Entry を返す。
func (s *Service) Top(limit int) ([]Entry, error) {
	matches, err := s.matches.ListClearedRankedByTime(limit)
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, 0, len(matches))

	for i, m := range matches {
		entry := Entry{
			Rank:      i + 1,
			Players:   m.PlayerIDs,
			ClearTime: float64(m.ClearTimeMs) / 1000.0,
			PlayedAt:  m.EndedAt.Format(time.RFC3339), // 文字列に変換
		}
		entries = append(entries, entry)
	}

	return entries, nil
}
