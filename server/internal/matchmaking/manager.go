package matchmaking

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"twin-switch-escape/server/internal/room"
)

// Status はマッチングの状態（spec §7.3）。
type Status string

const (
	StatusIdle      Status = "idle"      // キューにいない
	StatusWaiting   Status = "waiting"   // 相手待ち
	StatusMatched   Status = "matched"   // 成立
	StatusTimeout   Status = "timeout"   // 時間切れ
	StatusCancelled Status = "cancelled" // 取消
)

// ErrAlreadyMatching は既にマッチング中のユーザーが再度 start したとき返す（→ 409）。
var ErrAlreadyMatching = errors.New("already matching")

// Result はハンドラへ返す結果
// マッチング処理の結果を HTTP ハンドラに返す
type Result struct {
	Status        Status
	MatchmakingID string
	RoomID        string
	WebSocketURL  string
}

// entry は待機中/成立済みの1ユーザー分の状態。Run ゴルーチンだけが触る
// 1人のユーザーのマッチング状態をサーバー内部で管理
type entry struct {
	userID        string
	matchmakingID string
	status        Status
	roomID        string
	websocketURL  string
	enqueuedAt    time.Time
}

// --- コマンド（ハンドラ → Run ゴルーチン）---

type cmdKind int

const (
	cmdStart  cmdKind = iota // 0
	cmdStatus                // 1
	cmdCancel                // 2
)

type cmdResult struct {
	result Result
	err    error
}

type command struct {
	kind   cmdKind
	userID string
	reply  chan cmdResult
}

// Manager はマッチング全体を司る。
type Manager struct {
	rooms     *room.Manager // 成立時に Room を作る
	wsBaseURL string        // websocketUrl の基底（spec §8.1）
	timeout   time.Duration // 待機タイムアウト（spec §7.3）
	commands  chan command  // ハンドラからのコマンド受け口

	// ↓ Run ゴルーチンだけが触る状態（ロック不要）
	byUser map[string]*entry
}

// NewManager は Manager を生成する（まだ Run は始まらない。main で go Run する）。
func NewManager(rooms *room.Manager, wsBaseURL string, timeout time.Duration) *Manager {
	return &Manager{
		rooms:     rooms,
		wsBaseURL: wsBaseURL,
		timeout:   timeout,
		commands:  make(chan command),
		byUser:    make(map[string]*entry),
	}
}

// Run はアクター本体。単一ゴルーチンで commands と Ticker を逐次処理する。
// main から `go mgr.Run(ctx)` で起動する。ctx がキャンセルされたら終了。
func (m *Manager) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second) // 1秒ごとにタイムアウト走査
	defer ticker.Stop()

	for {
		select { // channelの待ち受け
		case <-ctx.Done(): // サーバー終了通知
			return
		case c := <-m.commands:
			var res cmdResult
			switch c.kind {
			case cmdStart:
				res.result, res.err = m.handleStart(c.userID)
			case cmdStatus:
				res.result, res.err = m.handleStatus(c.userID)
			case cmdCancel:
				res.result, res.err = m.handleCancel(c.userID)
			}
			c.reply <- res
		case now := <-ticker.C:
			m.handleTick(now)
		}
	}
}

// --- 公開メソッド（ハンドラが呼ぶ。内部で send して reply を待つ）---

func (m *Manager) Start(userID string) (Result, error)  { return m.send(cmdStart, userID) }
func (m *Manager) Status(userID string) (Result, error) { return m.send(cmdStatus, userID) }
func (m *Manager) Cancel(userID string) (Result, error) { return m.send(cmdCancel, userID) }

// send はコマンドを Run へ送り、reply を待つ（同期呼び出しに見せる）。
func (m *Manager) send(kind cmdKind, userID string) (Result, error) {
	reply := make(chan cmdResult, 1)                                // 返信用channelを作る
	m.commands <- command{kind: kind, userID: userID, reply: reply} // Runにcommandを送る
	r := <-reply                                                    // 結果が返るまで待つ
	return r.result, r.err
}

// handleStart はキュー投入とペアリングを行う（spec §7.3）。
//  1. e := m.byUser[userID]
//     既に存在し e.status == StatusWaiting または StatusMatched なら、
//     return Result{}, ErrAlreadyMatching（二重登録防止 → ハンドラが 409）。
//  2. 新しい entry を作る:
//     e = &entry{userID: userID, matchmakingID: newMatchmakingID(),
//     status: StatusWaiting, enqueuedAt: now}
//     m.byUser[userID] = e
//  3. 相手探し: m.byUser を走査して、自分以外で status==StatusWaiting のエントリを1人探す。
//     見つかったら2人をペアにする:
//     r := m.rooms.Create([]string{other.userID, userID})
//     url := m.websocketURL(r.ID)
//     other と e の両方を status=StatusMatched, roomID=r.ID, websocketURL=url にする。
//  4. 自分（e）の現在状態を Result にして返す:
//     Result{Status: e.status, MatchmakingID: e.matchmakingID,
//     RoomID: e.roomID, WebSocketURL: e.websocketURL}
//
// 2人専用なので「map を走査して待機者を1人見つける」で十分。
// 3人以上の公平な順序が要るなら enqueuedAt でソート、または別途スライスのキューを持つ。
func (m *Manager) handleStart(userID string) (Result, error) {
	if e, ok := m.byUser[userID]; ok && (e.status == StatusWaiting || e.status == StatusMatched) {
		return Result{}, ErrAlreadyMatching
	}

	// 新しい待機エントリを作る
	e := &entry{
		userID:        userID,
		matchmakingID: newMatchmakingID(),
		status:        StatusWaiting,
		enqueuedAt:    time.Now(),
	}
	m.byUser[userID] = e

	// 相手を探す：自分以外でwaitingを探す
	for _, other := range m.byUser {
		if other.userID == userID || other.status != StatusWaiting {
			continue
		}
		// 2人そろったらルーム発行して両者をmatchedにする
		r := m.rooms.Create([]string{other.userID, userID})
		url := m.websocketURL(r.ID)
		for _, p := range []*entry{other, e} {
			p.status = StatusMatched
			p.roomID = r.ID
			p.websocketURL = url
		}
		break
	}

	return Result{
		Status:        e.status,
		MatchmakingID: e.matchmakingID,
		RoomID:        e.roomID,
		WebSocketURL:  e.websocketURL,
	}, nil
}

// handleStatus は現在状態を返す（spec §7.3 のポーリング）。
//  1. e := m.byUser[userID]。無ければ return Result{Status: StatusIdle}, nil。
//  2. e の状態を Result に詰めて返す（matched なら roomID/websocketURL も）。
//  3. 終了状態（matched / timeout / cancelled）を返したら m.byUser から delete して掃除する
//     （メモリリーク防止。matched は「1回返したら削除」で十分。クライアントは roomId を保持して WS へ進む）。
func (m *Manager) handleStatus(userID string) (Result, error) {
	e, ok := m.byUser[userID]
	if !ok {
		return Result{Status: StatusIdle}, nil
	}

	res := Result{
		Status:        e.status,
		MatchmakingID: e.matchmakingID,
		RoomID:        e.roomID,
		WebSocketURL:  e.websocketURL,
	}

	switch e.status {
	case StatusMatched, StatusTimeout, StatusCancelled:
		delete(m.byUser, userID)
	}

	return res, nil
}

// handleCancel はキューから抜ける（spec §7.3）。
//  1. m.byUser[userID] があれば delete する（waiting のときだけでよい）。
//  2. return Result{Status: StatusCancelled}, nil。
func (m *Manager) handleCancel(userID string) (Result, error) {
	if e, ok := m.byUser[userID]; ok && e.status == StatusWaiting {
		delete(m.byUser, userID)
	}

	return Result{Status: StatusCancelled}, nil
}

// handleTick は1秒ごとに呼ばれ、タイムアウトと掃除を行う。
//   - m.byUser を走査し、status==StatusWaiting かつ now.Sub(e.enqueuedAt) >= m.timeout のものを
//     status=StatusTimeout にする（相手が来なかった人）。
//   - 古い終了状態エントリ（cancelled/timeout/matched で一定時間放置）を delete してもよい。
//
// map をループ中に delete しても Go では安全（イテレーション中の delete は許容）。
func (m *Manager) handleTick(now time.Time) {
	for userID, e := range m.byUser {
		switch {
		case e.status == StatusWaiting && now.Sub(e.enqueuedAt) >= m.timeout:
			// 相手が来なかった待機者をタイムアウトにする
			e.status = StatusTimeout
		case e.status == StatusMatched || e.status == StatusTimeout || e.status == StatusCancelled:
			// ポーリングされず放置された終了状態を、念のため一定時間後に掃除（メモリリーク防止）
			if now.Sub(e.enqueuedAt) >= 2*m.timeout {
				delete(m.byUser, userID)
			}
		}
	}
}

// newMatchmakingID は "mm_xxxx" を採番する。
func newMatchmakingID() string {
	return "mm_" + uuid.NewString()
}

// websocketURL は roomId から接続先 URL を組み立てる（spec §8.1）。
func (m *Manager) websocketURL(roomID string) string {
	return m.wsBaseURL + "/ws/rooms/" + roomID
}
