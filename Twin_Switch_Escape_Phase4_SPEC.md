# Twin Switch Escape — Phase 4 仕様書（マッチング実装）

本書は親仕様書 `Twin_Switch_Escape_SPEC.md`（以下「本仕様」）の **§18 Phase 4「マッチング実装」** を
実装するための詳細仕様である。Phase 2（REST API）・Phase 3（DB連携）を前提に、2人をマッチングして
ルームを発行する範囲を定義する。

- 関連する本仕様の節: §7.3（マッチングAPI）, §9.3（並行性モデル）, §11（内部設計）, §16（エラー）
- 対応コード: `server/internal/matchmaking/`, `server/internal/room/`

---

## 1. 目的とゴール

### 1.1 目的

ログイン済みのプレイヤー2人を待機キューで突き合わせ、揃ったら `roomId` を発行して
WebSocket 接続先（Phase 5 で使用）を返す。ここで **Go の並行処理（アクターモデル）** を実装し、
共有状態をロックではなく「単一ゴルーチン所有 + channel」で安全に扱う設計を学ぶ（本仕様 §9.3）。

### 1.2 完了条件（Definition of Done）

- [ ] `POST /api/matchmaking/start` で待機キューに入り `waiting` を返す（要認証）
- [ ] 2人目が入った時点で両者が `matched` になり、同じ `roomId` / `websocketUrl` を受け取る
- [ ] `GET /api/matchmaking/status` で現在状態（waiting / matched / timeout）をポーリング取得できる
- [ ] `POST /api/matchmaking/cancel` でキューから抜けられる（`cancelled`）
- [ ] 既にマッチング中のユーザーが再度 start すると `409`（重複登録しない）
- [ ] 一定時間（既定60秒）相手が来なければ `timeout` になる
- [ ] キュー操作が**単一ゴルーチン**に集約され、データ競合が起きない（`go test -race` で確認）
- [ ] `RoomManager` が `roomId → *Room` を `sync.RWMutex` で安全に管理する

### 1.3 スコープ外（後続フェーズ）

- WebSocket 接続・ルーム参加・位置同期・チャット（Phase 5・5.5）
- GameState・鍵/スイッチ/出口・クリア判定（Phase 6）
- 切断・再接続（Phase 7）
- Redis によるキュー外出し / 水平スケール（本仕様 §3.2。MVP は単一インスタンスのインメモリ）

> 本フェーズで作る `Room` は「ID と参加者を持つ箱」までで十分。ゲームループ・GameState・
> チャット履歴・専用ゴルーチンは Phase 5/6 で `Room` に足していく。

---

## 2. API 仕様（本仕様 §7.3）

すべて `Authorization: Bearer {accessToken}` が必要（認証ミドルウェアで `userId` を取得）。

### 2.1 マッチング開始

```
POST /api/matchmaking/start
```

レスポンス（200）:

```json
{ "status": "waiting", "matchmakingId": "mm_xxxx" }
```

- 既にマッチング中（waiting / matched）の場合は `409`（コード `ALREADY_MATCHING`）。

### 2.2 マッチング状態取得（ポーリング）

```
GET /api/matchmaking/status
```

待機中（200）:

```json
{ "status": "waiting" }
```

成立時（200）:

```json
{
  "status": "matched",
  "roomId": "room_xxxx",
  "websocketUrl": "ws://localhost:8080/ws/rooms/room_xxxx"
}
```

タイムアウト時（200）:

```json
{ "status": "timeout" }
```

- ポーリング間隔はクライアント側 1〜2 秒を推奨（本仕様 §7.3）。
- キューにいないユーザーは `idle`（または 404）を返す。本実装は `idle` を返す方針。

### 2.3 マッチングキャンセル

```
POST /api/matchmaking/cancel
```

レスポンス（200）:

```json
{ "status": "cancelled" }
```

### 2.4 エラー（本仕様 §16.1）

| 状況 | ステータス | コード |
|---|---|---|
| 未認証 / トークン不正 | 401 | `INVALID_TOKEN` |
| 既にマッチング中で再 start | 409 | `ALREADY_MATCHING` |
| サーバー内部エラー | 500 | `INTERNAL_ERROR` |

---

## 3. 並行性モデル（本フェーズの肝・本仕様 §9.3）

待機キューは複数の HTTP ハンドラ（＝複数ゴルーチン）から同時に触られるため、最も競合しやすい。
本実装は **「キューは1本のゴルーチンだけが触る（アクターモデル）」** で設計する。

```text
HTTP handler (goroutine A) ─┐
HTTP handler (goroutine B) ─┼─→ commands channel ─→ Manager.Run（単一ゴルーチン）
time.Ticker（タイムアウト） ─┘                          │ queue / byUser を直接操作（ロック不要）
                                                        └─→ reply channel で各 handler へ返す
```

- ハンドラは `Manager.Start/Status/Cancel` を呼ぶ。内部で**コマンドを channel に送り、reply を待つ**。
- `Manager.Run` が `select` で「コマンド」「Tick（タイムアウト走査）」を逐次処理する。
- キュー（`[]*entry`）と索引（`map[userID]*entry`）は Run ゴルーチンだけが触るので **mutex 不要**。
- 対照的に `RoomManager` の `map[roomId]*Room` は複数ゴルーチンから引かれるので `sync.RWMutex` で保護する
  （こちらは「読み取り主体の共有マップ」なので素直にロックでよい。使い分けを学ぶ）。

> ポイント: 「共有状態をロックで守る」より「状態を1ゴルーチンに閉じ込めて channel で会話する」方が
> デッドロックや競合が起きにくい。Phase 6 の Room も同じアクターモデルで作る布石になる。

---

## 4. マッチング状態遷移

```text
        start
 idle ───────────▶ waiting ──(2人揃う)──▶ matched
   ▲                  │
   │   cancel         │ 60秒経過
   └───── cancelled ◀─┘─────────▶ timeout
```

- `waiting` 中に2人目が来たら、待機していた1人目と新規の2人目をペアにして両者 `matched`。
- `matched` / `timeout` / `cancelled` になったエントリは、status が一度取得されたら掃除してよい
  （メモリリーク防止。タイムアウト走査時に古いものを削除）。

---

## 5. データ構造

### 5.1 matchmaking パッケージ

```go
type Status string
const (
    StatusIdle      Status = "idle"
    StatusWaiting   Status = "waiting"
    StatusMatched   Status = "matched"
    StatusTimeout   Status = "timeout"
    StatusCancelled Status = "cancelled"
)

// entry は待機中/成立済みの1ユーザー分の状態（Manager.Run だけが触る）。
type entry struct {
    userID        string
    matchmakingID string
    status        Status
    roomID        string
    websocketURL  string
    enqueuedAt    time.Time
}

// Result はハンドラへ返す結果。
type Result struct {
    Status        Status
    MatchmakingID string
    RoomID        string
    WebSocketURL  string
}
```

### 5.2 room パッケージ

```go
// Room は Phase 4 時点では「ID と参加者の箱」。Phase 5/6 で GameState 等を足す。
type Room struct {
    ID        string
    PlayerIDs []string
    CreatedAt time.Time
}

// Manager は roomId → *Room を保持。共有マップなので RWMutex で保護。
type Manager struct {
    mu    sync.RWMutex
    rooms map[string]*Room
}
```

---

## 6. 設定・環境変数（追加分）

| 変数 | 既定 | 用途 |
|---|---|---|
| `MATCHMAKING_TIMEOUT_SEC` | 60 | 待機タイムアウト秒（本仕様 §7.3） |
| `WS_BASE_URL` | `ws://localhost:8080` | `websocketUrl` の組み立てに使う基底 URL |

`websocketUrl = WS_BASE_URL + "/ws/rooms/" + roomId`（本仕様 §8.1）。

---

## 7. 実装タスク（埋める順番）

1. `internal/room/room.go` … `Room` 構造体
2. `internal/room/manager.go` … `Manager`（`Create` で roomId 発行、`Get`。RWMutex 保護）
3. `internal/matchmaking/manager.go`
   - コマンド型・`entry`・`Result`・`Status`
   - `Run(ctx)` の `select` ループ（**プラミングは提供済み**。中の `handleX` を実装）
   - `handleStart`（重複なら ALREADY_MATCHING、キュー投入、2人揃えばペアリング → `room.Create`）
   - `handleStatus`（現在状態を返し、終了状態は掃除）
   - `handleCancel`（キューから除去）
   - `handleTick`（enqueuedAt から timeout 判定、古いエントリ掃除）
4. `internal/matchmaking/handler.go` … 3 エンドポイント（DTO とエラー分岐）
5. `cmd/api/main.go` … RoomManager / Manager を生成し `go manager.Run(ctx)`、ルート登録
6. `internal/config/config.go` … `MatchmakingTimeout` / `WebSocketBaseURL` を追加

---

## 8. 受け入れテスト（手動 / curl）

2人分のトークンを用意してポーリングを再現する。

```bash
BASE=http://localhost:8080

# 事前に2ユーザー登録＆ログインして TOKEN_A, TOKEN_B を取得しておく

# A が開始 → waiting
curl -s -X POST $BASE/api/matchmaking/start -H "Authorization: Bearer $TOKEN_A"
# A が再度開始 → 409 ALREADY_MATCHING
curl -s -i -X POST $BASE/api/matchmaking/start -H "Authorization: Bearer $TOKEN_A" | head -1

# B が開始 → ここで2人揃う
curl -s -X POST $BASE/api/matchmaking/start -H "Authorization: Bearer $TOKEN_B"

# A / B の status → 両方 matched で同じ roomId
curl -s $BASE/api/matchmaking/status -H "Authorization: Bearer $TOKEN_A"
curl -s $BASE/api/matchmaking/status -H "Authorization: Bearer $TOKEN_B"

# 別ユーザー C が開始 → 相手が来ないので 60 秒後 timeout
curl -s $BASE/api/matchmaking/status -H "Authorization: Bearer $TOKEN_C"

# キャンセル
curl -s -X POST $BASE/api/matchmaking/cancel -H "Authorization: Bearer $TOKEN_C"
```

### 競合テスト（推奨）

`go test -race ./internal/matchmaking/...` で、多数のゴルーチンから同時に Start/Status/Cancel を
呼んでもデータ競合が検出されないことを確認する（アクターモデルの効果）。

---

## 9. 設計上のトレードオフ（ポートフォリオ説明用）

- **アクターモデル**: マッチングキューを単一ゴルーチンに閉じ込め、channel でコマンドを送る構成にした。
  共有状態のロック競合・データ競合を構造的に回避（本仕様 §9.3）。Phase 6 の Room も同方式で作る。
- **使い分け**: 「頻繁に書き換わる単一の真実（キュー）」はアクター、「読み取り主体の共有マップ
  （RoomManager）」は RWMutex、と適材適所でロック戦略を変えた。
- **片方だけ残るケース**を必ず処理: cancel / timeout で「相手待ちのまま放置」を掃除する。
- **Redis は使わない**（MVP 単一インスタンス）。水平スケール時にキュー外出し + スティッキールーティングが
  必要になる点を「将来の拡張」として説明できるようにする（本仕様 §4 補足）。
