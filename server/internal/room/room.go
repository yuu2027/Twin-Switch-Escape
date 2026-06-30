// Package room はルーム（2人が入る部屋）の状態と管理を提供する。
// Phase 4 では「ID と参加者の箱」までを作る。Phase 5/6 で GameState・チャット履歴・
// 専用ゴルーチン（room loop）を Room に足していく（spec §9.3 / §11.2）。
package room

import "time"

// Room は1つのルーム。Phase 4 時点の最小形。
//
// 学習ポイント:
//   - ここに将来 GameState *GameState / ChatHistory []*ChatMessage / コマンド用 channel などが増える。
//   - 今は「誰が参加していて、いつ作られたか」だけ持てばよい。
type Room struct {
	ID        string    // 例: "room_xxxx"（揮発。WebSocket URL に使う）
	PlayerIDs []string  // 参加者の userId（2人）
	CreatedAt time.Time

	// TODO(Phase 5/6): 以下を足していく予定（今は不要）。
	//   GameState   *GameState
	//   ChatHistory []*ChatMessage
	//   Status      RoomStatus
	//   commands    chan command   // room loop へのコマンド送信口
	
}
