// Package models はサーバー内部で扱うドメインモデルを定義する。
// Phase 2 ではインメモリで保持し、Phase 3 で PostgreSQL のレコードに対応させる。
// spec §11.2 / §12 を参照。
package models

import "time"

// User は登録ユーザー。spec §12.1 users テーブルに対応。
//
// 学習ポイント:
//   - PasswordHash には bcrypt のハッシュ文字列（60文字）を入れる。平文は絶対に保持しない（spec §17.1）。
//   - JSON タグは付けない方針。API レスポンス用の構造体はハンドラ側で別に定義し、
//     PasswordHash を誤って外部へ出さないようにする（内部モデルと公開モデルを分離する）。
type User struct {
	ID           string // 例: "user_xxxx"（crypto/rand 由来）
	Username     string // UNIQUE（spec §12.1）
	PasswordHash string // bcrypt ハッシュ
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Match は1試合の結果。spec §12.2 matches テーブルに対応。
//
// 学習ポイント:
//   - ClearTimeMs はミリ秒整数で保持する（spec §9.7 / §12.2）。
//     浮動小数だと同タイム比較・並べ替えで誤差が出るため。failed の試合では 0（または別途 nil 表現）。
//   - Result は "cleared" / "failed"。ランキングは cleared のみ対象（spec §15.2）。
type Match struct {
	ID           string // 例: "match_xxxx"
	RoomID       string // 揮発・参考情報（spec §12.2）
	Result       string // "cleared" | "failed"
	ClearTimeMs  int     // cleared のときのみ有効
	FailedReason string  // failed のときのみ
	PlayerIDs    []string // この試合に参加した userId（spec §12.3 match_players 相当をインメモリでは配列で表現）
	StartedAt    time.Time
	EndedAt      time.Time
	CreatedAt    time.Time
}

// TODO(Phase 3): match_players を独立した構造体/テーブルへ分離する。
// Phase 2 では Match.PlayerIDs に内包して簡略化している。
