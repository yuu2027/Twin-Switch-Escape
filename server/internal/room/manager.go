package room

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Manager は roomId → *Room を保持する。
//	この map は複数ゴルーチン（マッチング成立時の Create、後のフェーズで WebSocket ハンドラの Get）
//	から触られる「読み取り主体の共有マップ」なので、素直に sync.RWMutex で保護する。
//	※ マッチングキュー側はアクターモデル（単一ゴルーチン所有）で別物。使い分けを意識する。
type Manager struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

// NewManager は空の RoomManager を返す。
// （main の起動時に呼ばれるため実装済み。Create / Get が実装対象）
func NewManager() *Manager {
	return &Manager{
		rooms: make(map[string]*Room),
	}
}

// Create は参加者から新しい Room を作って登録し、返す。
//  1. roomID を採番する。例: "room_" + uuid.NewString()
//     （roomId は揮発的な識別子。DB の matches.room_id(VARCHAR) に後で入れてもよい）
//  2. &Room{ID: roomID, PlayerIDs: playerIDs, CreatedAt: time.Now()} を作る。
//  3. mu.Lock() / defer Unlock() で rooms[roomID] = room。
//  4. *Room を返す。
func (m *Manager) Create(playerIDs []string) *Room {
	roomID := "room_" + uuid.NewString() // RoomIDの作成

	room := &Room{
		ID:        roomID,
		PlayerIDs: playerIDs,
		CreatedAt: time.Now(),
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.rooms[roomID] = room

	return room
}

// Get は roomID で Room を引く。存在しなければ ok=false。
// mu.RLock() / defer RUnlock() で rooms[roomID] を返す（カンマOKイディオム）。
func (m *Manager) Get(roomID string) (*Room, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	room, ok := m.rooms[roomID]
	return room, ok
}
