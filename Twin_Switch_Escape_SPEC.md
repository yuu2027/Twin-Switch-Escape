# Twin Switch Escape 仕様書

## 1. プロジェクト概要

### 1.1 タイトル

**Twin Switch Escape**

### 1.2 コンセプト

Unityで制作する2Dトップビュー型の2人協力ミニ脱出ゲーム。  
ゲーム本体はシンプルにし、主目的を **GoによるゲームサーバーAPI開発** と **WebSocketを用いたオンライン通信** に置く。

プレイヤー2人は同じルームに参加し、制限時間内に鍵を集め、同時スイッチを押して出口を開放し、脱出を目指す。  
また、ゲーム中は同じルーム内でテキストチャットを行える。

### 1.3 開発目的

本プロジェクトでは、Unityによるゲームクライアント開発に加え、Goによるバックエンド開発を行う。  
特に以下の技術を示すことを目的とする。

- REST API設計
- JWT認証
- マッチング処理
- ルーム管理
- WebSocketによるリアルタイム通信
- プレイヤー位置同期
- ルーム内チャット
- サーバー権威型のゲーム状態管理
- 切断検知
- 再接続処理
- チャット履歴復元
- DB保存
- ランキング機能
- Dockerによる開発環境構築

---

## 2. ゲーム概要

### 2.1 ゲームジャンル

2Dトップビュー型オンライン協力ゲーム

### 2.2 プレイ人数

2人

### 2.3 基本ルール

1. プレイヤーはログイン後、マッチングを開始する。
2. 2人揃うとルームが作成される。
3. UnityクライアントはWebSocketでGoサーバーに接続する。
4. ゲーム開始後、2人は小さなマップ内を移動する。
5. マップ内に配置された鍵を取得する。
6. 2人がそれぞれ別のスイッチを同時に押すと出口が開く。
7. 出口が開いた状態で2人が出口範囲に入るとクリアとなる。
8. クリアタイムはサーバー側で計算され、ランキングに保存される。
9. 制限時間を超えると失敗となる。
10. ゲーム中は同じルーム内でテキストチャットを行える。
11. 通信切断が発生した場合、一定時間以内なら再接続できる。
12. 再接続時には現在のゲーム状態と直近のチャット履歴を復元する。

### 2.4 ゲームの勝敗条件

#### クリア条件

以下をすべて満たした場合、ゲームクリアとする。

- 必要な鍵をすべて取得している
- 出口が開いている
- 2人のプレイヤーが出口範囲内にいる
- 制限時間内である

#### 失敗条件

以下のいずれかに該当した場合、ゲーム失敗とする。

- 制限時間を超過する
- プレイヤーが切断後、再接続猶予時間内に復帰しない
- サーバー側でルームが異常終了する

---

## 3. 想定技術スタック

### 3.1 クライアント

| 項目 | 技術 |
|---|---|
| ゲームエンジン | Unity |
| 言語 | C# |
| REST通信 | UnityWebRequest |
| リアルタイム通信 | WebSocketクライアントライブラリ |
| マップ | Unity Tilemap |
| UI | uGUI / TextMeshPro |
| データ形式 | JSON |

### 3.2 バックエンド

| 項目 | 技術 |
|---|---|
| 言語 | Go (1.22+) |
| HTTPサーバー | net/http（標準ライブラリ。ルーティングは Go 1.22 の `http.ServeMux` で十分。必要なら chi 等の軽量ルーター） |
| WebSocket | coder/websocket（旧 nhooyr/websocket）または gorilla/websocket |
| 認証 | JWT（アクセストークン）+ bcrypt |
| 構造化ログ | log/slog（標準ライブラリ） |
| DB | PostgreSQL |
| マイグレーション | golang-migrate または goose |
| 一時データ管理 | Redis（マッチングキュー用。MVPでは省略可） |
| コンテナ | Docker / Docker Compose |
| API仕様 | OpenAPI |
| テスト | Go testing + testify（任意） |

> **技術選定の補足**
> - **HTTPサーバー**: Gin でもよいが、本規模なら標準 `net/http` + Go 1.22 の改善された `ServeMux`（メソッド・パスパラメータ対応）で依存を増やさず実装できる。「標準ライブラリで組んだ」こと自体がGoの理解を示せる。
> - **WebSocket**: gorilla/websocket は一度アーカイブされたが現在は再びメンテされている。新規なら API がシンプルで `context` 連携の良い coder/websocket（旧 nhooyr/websocket）も有力。どちらでも可。
> - **Redis**: 後述する通り、**進行中ルームの真の状態はGoプロセス内メモリが正本**であり、Redis は主にマッチングキュー（および将来の水平スケール時の pub/sub）用途。単一インスタンス構成のMVPでは Redis なしでも成立するため、最初は省略し Lv.4 以降で導入する判断も可。

### 3.3 通信方式

| 用途 | 通信方式 |
|---|---|
| ユーザー登録 | REST API |
| ログイン | REST API |
| ユーザー情報取得 | REST API |
| マッチング開始 | REST API |
| ランキング取得 | REST API |
| 試合履歴取得 | REST API |
| チャット履歴取得 | REST API |
| ルーム内同期 | WebSocket |
| プレイヤー入力送信 | WebSocket |
| ゲーム状態配信 | WebSocket |
| ルーム内チャット | WebSocket |
| 切断・再接続 | WebSocket + REST API |

---

## 4. システム構成

```text
Unity Client
  ├─ REST API Client
  │   ├─ register
  │   ├─ login
  │   ├─ me
  │   ├─ matchmaking
  │   ├─ ranking
  │   ├─ match history
  │   └─ chat history
  │
  └─ WebSocket Client
      ├─ room join
      ├─ player input
      ├─ state sync
      ├─ chat send / receive
      ├─ event receive
      └─ reconnect

Go Server
  ├─ REST API Server
  ├─ WebSocket Server
  ├─ Auth Service
  ├─ Matchmaking Manager
  ├─ Room Manager
  ├─ GameState Manager
  ├─ Chat Manager
  ├─ Reconnect Manager
  ├─ Ranking Service
  └─ Repository Layer

PostgreSQL
  ├─ users
  ├─ matches          # ランキングは matches(result='cleared') から導出するため専用テーブルは持たない
  ├─ match_players
  └─ chat_messages

Redis（任意 / スケールアウト時に有効）
  ├─ matchmaking queue       # 待機中ユーザーのキュー
  └─ reconnect token index   # 複数インスタンス時にどのインスタンスがルームを保持するか引くため
```

> **状態の正本（source of truth）について**
> 進行中のルーム状態（GameState・接続状況・直近チャット）は、**そのルームを担当する Go プロセスのメモリ上が正本**である。Redis はマッチングキューや、将来複数インスタンスへ分散する際のルーティング情報を持つために使う。
> したがって「Redis を入れたから自動的に水平スケールできる」わけではない点に注意する。WebSocket で状態を保持するサーバーをスケールアウトするには、**同一ルームの2接続を必ず同じインスタンスへ振り分けるルーティング（スティッキー）** と、ルームの所在を引く**ルームディレクトリ**が別途必要になる。MVPは単一インスタンス前提とし、この制約をポートフォリオの「設計上のトレードオフ」として説明できるようにする。

---

## 5. Unity側仕様

### 5.1 シーン構成

| シーン名 | 内容 |
|---|---|
| TitleScene | タイトル画面 |
| LoginScene | ログイン / ユーザー登録 |
| LobbyScene | マッチング・ランキング表示 |
| GameScene | 2D協力ゲーム本体 |
| ResultScene | クリア結果表示 |

### 5.2 マップ仕様

2DトップビューのTilemapを使用する。

#### Tilemapで管理するもの

- 床
- 壁
- 通路
- 装飾
- 移動不可領域

#### GameObjectで管理するもの

- プレイヤー
- 鍵
- スイッチ
- 出口
- チャットUI
- 通信管理オブジェクト

### 5.3 Unityオブジェクト構成

```text
GameScene
├─ Grid
│  ├─ FloorTilemap
│  ├─ WallTilemap
│  └─ StoneTilemap
│
├─ Players
│  ├─ LocalPlayer
│  └─ RemotePlayer
│
├─ Interactables
│  ├─ Key_1
│  ├─ Key_2
│  ├─ Switch_A
│  ├─ Switch_B
│  └─ Exit
│
├─ UI
│  ├─ TimerText
│  ├─ ConnectionStatusText
│  ├─ ChatPanel
│  ├─ ChatMessageList
│  ├─ ChatInputField
│  └─ ChatSendButton
│
└─ Managers
   ├─ GameManager
   ├─ ApiClient
   ├─ WebSocketClient
   ├─ NetworkGameStateView
   ├─ ChatUIController
   └─ UIManager
```

### 5.4 プレイヤー操作

| 操作 | 内容 |
|---|---|
| WASD / 矢印キー | 移動 |
| Eキー | インタラクト |
| Enter | チャット入力欄を開く |
| Enter入力中 | チャット送信 |
| Esc | チャット入力欄またはメニューを閉じる |
| Tab | チャット欄の表示 / 非表示 |

### 5.5 Unity側の責務

Unity側は基本的に **入力と表示** を担当する。

Unity側で行う処理：

- プレイヤー入力（移動ベクトル）の取得
- サーバーへの入力送信
- サーバーから受け取ったGameStateの反映
- **自プレイヤーのクライアント予測（送った入力を先行適用し、サーバー確定値で補正）**
- **相手プレイヤーの位置補間（state_update間をLerp等で滑らかに描画）**
- アニメーション表示
- UI表示
- チャット入力の取得
- チャットメッセージ表示
- 接続状態表示
- 再接続リクエスト

Unity側で行わない処理：

- 鍵取得の最終判定
- 出口開放の最終判定
- クリアタイムの計算
- ランキング登録
- ゲーム結果の確定
- チャット配信先の最終決定
- チャットログの正式保存

これらはGoサーバー側で行う。

---

## 6. バックエンド難易度レベルとゲーム機能対応

本プロジェクトでは、ゲームバックエンド難易度Lv.7相当までを実装対象とする。

| レベル | バックエンド内容 | ゲーム内機能 |
|---|---|---|
| Lv.1 | 単純API通信 | ゲーム設定取得・お知らせ取得 |
| Lv.2 | データ保存API | クリア結果保存・ランキング取得 |
| Lv.3 | 認証・ユーザー管理 | ログイン・JWT認証 |
| Lv.4 | マッチング・ルーム管理 | 2人マッチング・roomId発行 |
| Lv.5 | リアルタイム通信 | 位置同期・状態同期・ルーム内チャット |
| Lv.6 | サーバー権威型状態管理 | 鍵・スイッチ・出口・クリア判定 |
| Lv.7 | 切断・再接続 | 一定時間内の復帰・GameState復元・チャット履歴復元 |

---

## 7. REST API仕様

### 7.1 認証API

#### ユーザー登録

```http
POST /api/register
```

リクエスト例：

```json
{
  "username": "player001",
  "password": "password123"
}
```

レスポンス例：

```json
{
  "userId": "user_001",
  "username": "player001"
}
```

#### ログイン

```http
POST /api/login
```

リクエスト例：

```json
{
  "username": "player001",
  "password": "password123"
}
```

レスポンス例：

```json
{
  "accessToken": "xxxxx.yyyyy.zzzzz",
  "userId": "user_001",
  "username": "player001"
}
```

#### 自分の情報取得

```http
GET /api/me
Authorization: Bearer {accessToken}
```

レスポンス例：

```json
{
  "userId": "user_001",
  "username": "player001",
  "bestClearTime": 74.2,
  "clearCount": 5
}
```

---

### 7.2 ゲーム設定API

#### ゲーム設定取得

```http
GET /api/game-config
```

レスポンス例：

```json
{
  "stageId": "stage_01",
  "stageName": "Twin Switch Lab",
  "timeLimitSec": 180,
  "requiredKeys": 2,
  "reconnectTimeoutSec": 30,
  "chatMaxLength": 100,
  "chatRateLimitSec": 1
}
```

---

### 7.3 マッチングAPI

#### マッチング開始

```http
POST /api/matchmaking/start
Authorization: Bearer {accessToken}
```

レスポンス例：

```json
{
  "status": "waiting",
  "matchmakingId": "mm_001"
}
```

#### マッチング状態取得

```http
GET /api/matchmaking/status
Authorization: Bearer {accessToken}
```

マッチング待機中：

```json
{
  "status": "waiting"
}
```

マッチング成立時：

```json
{
  "status": "matched",
  "roomId": "room_001",
  "websocketUrl": "ws://localhost:8080/ws/rooms/room_001"
}
```

#### マッチングキャンセル

```http
POST /api/matchmaking/cancel
Authorization: Bearer {accessToken}
```

レスポンス例：

```json
{
  "status": "cancelled"
}
```

> **マッチング設計の補足**
> - `GET /api/matchmaking/status` のポーリング間隔は 1〜2 秒を推奨。WebSocket常設前なのでポーリングで十分だが、間隔が短すぎると無駄な負荷になる。
> - **マッチングタイムアウト**を設ける（例: 60秒で `status: "timeout"`）。相手が見つからないまま待ち続ける状態を避ける。
> - 待機キューへの出し入れは競合しやすい。**マッチング処理を1つのゴルーチンに集約（キューへの操作を単一所有者に）** するか、Redis を使う場合は `LPOP`/`RPUSH` の原子性を利用して、2人を取り出す処理が二重に走らないようにする。
> - キャンセルやクライアント離脱で「片方だけキューに残る」ケースを必ず処理する。

---

### 7.4 ランキングAPI

#### ランキング取得

```http
GET /api/ranking
```

レスポンス例：

```json
{
  "rankings": [
    {
      "rank": 1,
      "players": ["player001", "player002"],
      "clearTime": 72.4,
      "playedAt": "2026-06-20T12:00:00Z"
    },
    {
      "rank": 2,
      "players": ["player003", "player004"],
      "clearTime": 81.7,
      "playedAt": "2026-06-20T12:10:00Z"
    }
  ]
}
```

#### 自分の試合履歴取得

```http
GET /api/matches/me
Authorization: Bearer {accessToken}
```

レスポンス例：

```json
{
  "matches": [
    {
      "matchId": "match_001",
      "result": "cleared",
      "clearTime": 72.4,
      "playedAt": "2026-06-20T12:00:00Z"
    }
  ]
}
```

---

### 7.5 チャット履歴API

#### 試合のチャット履歴取得

```http
GET /api/matches/{matchId}/chats
Authorization: Bearer {accessToken}
```

レスポンス例：

```json
{
  "messages": [
    {
      "messageId": "chat_001",
      "senderId": "user_001",
      "senderName": "player001",
      "message": "switch_a に向かいます",
      "sentAt": "2026-06-20T12:00:10Z"
    },
    {
      "messageId": "chat_002",
      "senderId": "user_002",
      "senderName": "player002",
      "message": "了解です",
      "sentAt": "2026-06-20T12:00:14Z"
    }
  ]
}
```

このAPIは試合後の履歴確認用とする。  
ゲーム中のリアルタイムチャットはWebSocketで行う。

---

## 8. WebSocket仕様

### 8.1 接続URL

```text
ws://localhost:8080/ws/rooms/{roomId}
```

接続時にはJWTを送信する。

> **識別子の統一**: 本仕様では `playerId` と `userId` は同一の値（例: `user_001`）を指す。REST/DB側は `userId`、WebSocketのペイロード上の表記は歴史的に `playerId` だが、実体は同じユーザーIDである。実装では片方（`userId`）に寄せて混乱を避けることを推奨する。
>
> **プロトコルバージョン**: 全WebSocketメッセージに `"v": 1` のようなバージョンフィールドを持たせ、将来のメッセージ形式変更に備える（任意だが推奨）。

例：

```json
{
  "type": "join_room",
  "payload": {
    "accessToken": "xxxxx.yyyyy.zzzzz",
    "roomId": "room_001"
  }
}
```

---

### 8.2 クライアントからサーバーへ送信するメッセージ

#### ルーム参加

```json
{
  "type": "join_room",
  "payload": {
    "roomId": "room_001",
    "accessToken": "xxxxx.yyyyy.zzzzz"
  }
}
```

#### プレイヤー入力

移動入力のみを送る。インタラクト（鍵・出口）は専用メッセージ（`try_pickup_key` / `try_exit`）に一本化し、判定経路の二重化を避ける。

```json
{
  "type": "player_input",
  "payload": {
    "seq": 1024,
    "moveX": 1.0,
    "moveY": 0.0
  }
}
```

- `moveX` / `moveY` は正規化済みの**入力ベクトル（向き）**であり、座標そのものではない。実際の移動量はサーバーが `moveSpeed × dt` で計算する（後述の§9.2移動モデル）。
- `seq` はクライアント側の連番。サーバーは `state_update` で最後に処理した `seq` を返し、クライアント予測の補正に用いる。
- 送信レートはクライアントの入力Tick（例: 30Hz）に合わせる。同一フレームの重複入力はまとめてよい。

#### 鍵取得試行

```json
{
  "type": "try_pickup_key",
  "payload": {
    "keyId": "key_1"
  }
}
```

#### スイッチ押下（補足）

スイッチの「押されている／いない」は**クライアントから送らない**。サーバーが各Tickでプレイヤー座標を見て「スイッチ範囲内にプレイヤーがいるか」を判定し、`pressedBy` を導出する（§9.3）。これにより、範囲外から `isPressed: true` を送りつけるなりすましを構造的に防ぐ。

> 旧版では `switch_state {isPressed}` をクライアントが送る設計だったが、サーバー権威型の方針と矛盾し（クライアント申告を信用してしまう）、なりすまし可能なため廃止した。プレイヤーは「スイッチの上に立つ」ことで押下状態になる。

#### 出口進入試行

```json
{
  "type": "try_exit"
}
```

#### チャット送信

```json
{
  "type": "chat_message",
  "payload": {
    "message": "switch_a に向かいます"
  }
}
```

#### 再接続

```json
{
  "type": "reconnect",
  "payload": {
    "roomId": "room_001",
    "playerId": "user_001",
    "reconnectToken": "reconnect_xxxxx"
  }
}
```

#### Ping

```json
{
  "type": "ping"
}
```

---

### 8.3 サーバーからクライアントへ送信するメッセージ

#### ルーム参加成功

```json
{
  "type": "join_success",
  "payload": {
    "roomId": "room_001",
    "playerId": "user_001",
    "reconnectToken": "reconnect_xxxxx"
  }
}
```

#### ゲーム開始

```json
{
  "type": "game_start",
  "payload": {
    "startedAt": "2026-06-20T12:00:00Z",
    "timeLimitSec": 180
  }
}
```

#### 状態同期

```json
{
  "type": "state_update",
  "payload": {
    "remainingTimeSec": 128,
    "players": [
      {
        "playerId": "user_001",
        "x": 2.4,
        "y": 1.8,
        "connected": true
      },
      {
        "playerId": "user_002",
        "x": 5.1,
        "y": 3.2,
        "connected": true
      }
    ],
    "keys": [
      {
        "keyId": "key_1",
        "collected": true
      },
      {
        "keyId": "key_2",
        "collected": false
      }
    ],
    "switches": [
      {
        "switchId": "switch_a",
        "pressedBy": "user_001"
      },
      {
        "switchId": "switch_b",
        "pressedBy": null
      }
    ],
    "exitOpen": false,
    "status": "playing"
  }
}
```

#### 鍵取得通知

```json
{
  "type": "key_collected",
  "payload": {
    "keyId": "key_1",
    "playerId": "user_001"
  }
}
```

#### 出口開放通知

```json
{
  "type": "exit_opened"
}
```

#### チャット配信

```json
{
  "type": "chat_broadcast",
  "payload": {
    "messageId": "chat_001",
    "roomId": "room_001",
    "senderId": "user_001",
    "senderName": "player001",
    "message": "switch_a に向かいます",
    "sentAt": "2026-06-20T12:00:10Z"
  }
}
```

#### チャット履歴送信

```json
{
  "type": "chat_history",
  "payload": {
    "messages": [
      {
        "messageId": "chat_001",
        "senderId": "user_001",
        "senderName": "player001",
        "message": "switch_a に向かいます",
        "sentAt": "2026-06-20T12:00:10Z"
      },
      {
        "messageId": "chat_002",
        "senderId": "user_002",
        "senderName": "player002",
        "message": "了解です",
        "sentAt": "2026-06-20T12:00:14Z"
      }
    ]
  }
}
```

#### チャット送信エラー

```json
{
  "type": "chat_error",
  "payload": {
    "code": "MESSAGE_TOO_LONG",
    "message": "Chat message must be 100 characters or less."
  }
}
```

#### ゲームクリア

```json
{
  "type": "game_clear",
  "payload": {
    "clearTimeSec": 82.4,
    "rank": 3
  }
}
```

#### ゲーム失敗

```json
{
  "type": "game_failed",
  "payload": {
    "reason": "time_over"
  }
}
```

#### プレイヤー切断通知

```json
{
  "type": "player_disconnected",
  "payload": {
    "playerId": "user_002",
    "reconnectTimeoutSec": 30
  }
}
```

#### 再接続成功

```json
{
  "type": "reconnect_success",
  "payload": {
    "gameState": {
      "remainingTimeSec": 92,
      "exitOpen": true,
      "status": "playing"
    },
    "chatHistory": [
      {
        "messageId": "chat_001",
        "senderId": "user_001",
        "senderName": "player001",
        "message": "switch_a に向かいます",
        "sentAt": "2026-06-20T12:00:10Z"
      }
    ]
  }
}
```

#### Pong

```json
{
  "type": "pong"
}
```

---

## 9. サーバー権威型設計

### 9.1 基本方針

本プロジェクトでは、クライアントから送られた結果をそのまま信用しない。  
Unityクライアントは入力や試行をサーバーへ送信し、Goサーバーが判定を行う。

### 9.2 移動モデルとサーバーTick（重要）

「サーバー権威型」を成立させるには、**座標もサーバーが計算する**必要がある。クライアントは座標を送らず、入力ベクトル（向き）だけを送る。

サーバーは固定Tickのシミュレーションループを持つ：

```text
- シミュレーションTick: 20Hz（50msごと）を推奨
- state_update配信レート: 15〜20Hz（Tickと同じか間引く）
- 各Tickの処理:
    1. 直近の player_input（moveX, moveY）を取り出す
    2. nextX = x + moveX * moveSpeed * dt
       nextY = y + moveY * moveSpeed * dt
    3. 壁・マップ外との衝突判定（衝突する軸はその場で止める）
    4. GameStateの座標を更新
    5. スイッチ・出口など範囲判定を再評価（§9.3, §9.4）
    6. 接続中の全プレイヤーへ state_update を配信
```

- `moveSpeed`（タイル/秒）はサーバー側の定数またはステージデータで持ち、**クライアントからは渡さない**。
- クライアントは自分の入力を先行適用（予測）して描画し、`state_update` の確定座標で補正する。相手プレイヤーはTick間を補間して滑らかに描画する。
- これにより「座標改ざん」「スピードハック」を構造的に防げる（クライアントが座標を主張できない）。

### 9.3 並行性モデル（Goの肝）

ルーム状態への同時アクセスは、Goでデータ競合を起こしやすい最重要ポイント。**1ルーム = 1ゴルーチン（アクターモデル）** で設計し、ルーム状態（GameState・接続・チャット履歴）はそのゴルーチンだけが触る。

```text
- 各 Room は専用ゴルーチン（room loop）を1本持つ
- 外部（WebSocket受信、Tickタイマー、切断検知）からは
  channel 経由でコマンドを送る
- room loop が select で:
    - 受信メッセージ（input / pickup / chat / reconnect ...）
    - Tickタイマー（time.Ticker）
    - 切断・終了シグナル
  を逐次処理する → GameStateにmutex不要
- RoomManager が持つ「roomId -> *Room」マップだけ sync.RWMutex で保護
```

この設計は「Goの並行処理を理解している」ことを最も分かりやすく示せるため、ポートフォリオの説明ポイントにする。共有状態を mutex で守る素朴な実装よりも、channel + 単一所有者の方がバグりにくい。

### 9.4 鍵取得判定

Unity側：

```text
key_1を取得しようとした
```

Go側：

```text
1. 対象の鍵が存在するか確認
2. 鍵が未取得であるか確認
3. プレイヤーが鍵の取得可能範囲内にいるか確認
4. 条件を満たす場合、鍵を取得済みに変更
5. 同じルームの全プレイヤーへ通知
```

### 9.5 スイッチ判定

スイッチの押下状態は、各Tickでプレイヤー座標から導出する。

```text
1. 各スイッチについて、範囲内にいるプレイヤーを探す
2. いれば pressedBy = そのplayerId、いなければ null（離れたら自動で解除）
3. Switch_A と Switch_B が「同一Tickで両方とも pressedBy != null」か
4. 必要な鍵がすべて取得済みか
5. 条件を満たす場合、出口を開放する（exitOpen = true）
```

> ポイント: 「同時押し」はサーバーが1Tick内で両スイッチの占有を確認することで判定する。プレイヤーがスイッチから離れれば押下は解除されるため、片方を押したまま放置→もう片方へ、では開かない（協力要素が成立する）。

### 9.6 クリア判定

Go側で以下を判定する。

```text
1. 出口が開いているか
2. 2人のプレイヤーが出口範囲内にいるか
3. 制限時間内であるか
4. 条件を満たす場合、クリア確定
5. クリアタイムをサーバー側で計算（ミリ秒精度）
6. DBへ保存
7. ランキングは matches テーブルへの保存により自動的に反映
```

### 9.7 タイマー管理

ゲーム開始時刻はGoサーバーが保持する。

```text
clearTimeMs   = clearedAt - startedAt   （ミリ秒整数で保持）
remainingTime = timeLimitSec - elapsedTime
```

Unity側のタイマー表示は、サーバーから受け取った残り時間を基準とする。  
クリアタイムをミリ秒整数で保持するのは、ランキングの同タイム比較や並べ替えで浮動小数の誤差を避けるため（API応答では秒(小数)に変換して返してよい）。

---

## 10. チャット機能仕様

### 10.1 概要

本プロジェクトでは、2人協力プレイ中にプレイヤー同士がテキストチャットを行える機能を実装する。  
チャットはUnityクライアント同士が直接通信するのではなく、GoサーバーのWebSocketを経由して同じルーム内のプレイヤーへ配信する。

```text
Unity Player A
  ↓ chat_message
Go WebSocket Server
  ↓ chat_broadcast
Unity Player B
```

### 10.2 チャット機能の目的

チャット機能により、以下を実装・説明できる。

- WebSocketによる双方向通信
- ルーム内ブロードキャスト
- プレイヤーIDと発言者名の管理
- メッセージのバリデーション
- 連続投稿制限
- チャットログ保持
- チャットログDB保存
- 切断・再接続時のチャット履歴復元

### 10.3 チャットUI

GameSceneにチャットUIを追加する。

UI要素：

- チャット表示欄
- 入力欄
- 送信ボタン
- 未読通知
- チャット表示 / 非表示切り替え

### 10.4 チャット送信条件

| 項目 | 条件 |
|---|---|
| 空文字 | 不可 |
| 最大文字数 | 100文字 |
| 送信間隔 | 1秒に1回まで |
| 送信対象 | 同じルーム内のみ |
| 送信可能状態 | waiting / playing / cleared / failed |

### 10.5 連続投稿制限

同一プレイヤーは1秒に1回までチャットを送信できる。

違反時のレスポンス：

```json
{
  "type": "chat_error",
  "payload": {
    "code": "RATE_LIMITED",
    "message": "Please wait before sending another message."
  }
}
```

### 10.6 チャットログ保持

チャットログは以下の2種類で管理する。

#### インメモリ保持

ゲーム中の即時表示・再接続復元用として、Room内に直近のチャット履歴を保持する。

#### DB保存

試合履歴としてチャットログを残すため、必要に応じてDBにも保存する。  
最低限の完成ラインではインメモリ保持のみ、標準完成ラインではDB保存まで行う。

---

## 11. Goサーバー内部設計

### 11.1 ディレクトリ構成案

```text
server/
├─ cmd/
│  └─ api/
│     └─ main.go
│
├─ internal/
│  ├─ auth/
│  │  ├─ handler.go
│  │  ├─ service.go
│  │  └─ jwt.go
│  │
│  ├─ matchmaking/
│  │  ├─ handler.go
│  │  └─ manager.go
│  │
│  ├─ room/
│  │  ├─ manager.go
│  │  ├─ room.go
│  │  └─ game_state.go
│  │
│  ├─ websocket/
│  │  ├─ handler.go
│  │  ├─ client.go
│  │  └─ message.go
│  │
│  ├─ chat/
│  │  ├─ manager.go
│  │  ├─ message.go
│  │  └─ service.go
│  │
│  ├─ ranking/
│  │  ├─ handler.go
│  │  └─ service.go
│  │
│  ├─ repository/
│  │  ├─ user_repository.go
│  │  ├─ match_repository.go
│  │  ├─ ranking_repository.go
│  │  └─ chat_repository.go
│  │
│  ├─ middleware/
│  │  └─ auth_middleware.go
│  │
│  └─ config/
│     └─ config.go
│
├─ migrations/
├─ docker-compose.yml
├─ Dockerfile
└─ README.md
```

### 11.2 主要構造体案

#### Room

```go
type Room struct {
    ID          string
    Players     map[string]*PlayerSession
    GameState   *GameState
    ChatHistory []*ChatMessage
    Status      RoomStatus
}
```

#### PlayerSession

```go
type PlayerSession struct {
    UserID         string
    Username       string
    Connected      bool
    LastSeen       time.Time
    ReconnectToken string
    LastChatSentAt time.Time
}
```

#### GameState

```go
type GameState struct {
    RoomID       string
    StartedAt    time.Time
    TimeLimitSec int

    Players  map[string]*PlayerState
    Keys     map[string]*KeyState
    Switches map[string]*SwitchState

    ExitOpen bool
    Status   GameStatus
}
```

#### PlayerState

```go
type PlayerState struct {
    UserID    string
    X         float64
    Y         float64
    Connected bool
}
```

#### KeyState

```go
type KeyState struct {
    KeyID     string
    X         float64
    Y         float64
    Collected bool
}
```

#### SwitchState

```go
type SwitchState struct {
    SwitchID  string
    X         float64
    Y         float64
    PressedBy *string
}
```

#### ChatMessage

```go
type ChatMessage struct {
    ID         string
    RoomID     string
    SenderID   string
    SenderName string
    Message    string
    SentAt     time.Time
}
```

---

## 12. DB設計

### 12.1 users

| カラム | 型 | 内容 |
|---|---|---|
| id | UUID | ユーザーID（PK） |
| username | VARCHAR(32) | ユーザー名（**UNIQUE 制約必須**） |
| password_hash | CHAR(60) | bcryptハッシュ（固定60文字） |
| created_at | TIMESTAMP | 作成日時 |
| updated_at | TIMESTAMP | 更新日時 |

- `username` には UNIQUE 制約を付ける（登録時の重複防止。アプリ側チェックだけだと競合で二重登録され得る）。
- `/api/me` が返す `bestClearTime` / `clearCount` は専用カラムを持たず、`matches` + `match_players` からの集計クエリで導出する。

### 12.2 matches

| カラム | 型 | 内容 |
|---|---|---|
| id | UUID | 試合ID（PK） |
| room_id | VARCHAR | ルームID（揮発。参考情報） |
| result | VARCHAR | cleared / failed |
| clear_time_ms | INTEGER | クリア時間（ミリ秒。failed時はNULL） |
| failed_reason | VARCHAR | 失敗理由（cleared時はNULL） |
| started_at | TIMESTAMP | 開始時刻 |
| ended_at | TIMESTAMP | 終了時刻 |
| created_at | TIMESTAMP | 作成日時 |

- ランキングは専用テーブルを持たず、`SELECT ... FROM matches WHERE result='cleared' ORDER BY clear_time_ms ASC` で導出する（状態の二重管理を避ける）。
- このクエリ用に **部分インデックス** を張る: `CREATE INDEX ON matches (clear_time_ms) WHERE result = 'cleared';`
- クリア時間は浮動小数ではなく**ミリ秒整数**で保持し、並べ替え・同タイム比較を正確にする。

### 12.3 match_players

| カラム | 型 | 内容 |
|---|---|---|
| id | UUID | ID（PK） |
| match_id | UUID | 試合ID（FK → matches.id） |
| user_id | UUID | ユーザーID（FK → users.id） |
| created_at | TIMESTAMP | 作成日時 |

- `UNIQUE (match_id, user_id)` を付け、同一試合への二重登録を防ぐ。
- `/api/matches/me` 用に `user_id` にインデックスを張る。

### 12.4 chat_messages

| カラム | 型 | 内容 |
|---|---|---|
| id | UUID | チャットメッセージID（PK） |
| match_id | UUID | 試合ID（FK → matches.id） |
| sender_id | UUID | 送信者ID（FK → users.id） |
| message | VARCHAR(100) | メッセージ本文 |
| sent_at | TIMESTAMP | 送信日時 |
| created_at | TIMESTAMP | 作成日時 |

- 履歴取得（`/api/matches/{matchId}/chats`）用に `(match_id, sent_at)` の複合インデックスを張る。
- 旧版の `room_id` カラムは削除した。room_id は揮発的でDB上は試合の安定キーにならず、`match_id` があれば履歴をたどれるため冗長だった。

---

## 13. ステージデータ仕様

UnityのTilemapをGoサーバーが直接読むのではなく、Go側では簡略化したステージデータを持つ。

### 13.1 ステージデータ例

```json
{
  "stageId": "stage_01",
  "width": 12,
  "height": 8,
  "timeLimitSec": 180,
  "playerSpawns": [
    { "x": 1, "y": 1 },
    { "x": 2, "y": 1 }
  ],
  "walls": [
    { "x": 0, "y": 0 },
    { "x": 1, "y": 0 },
    { "x": 2, "y": 0 }
  ],
  "keys": [
    { "id": "key_1", "x": 3, "y": 2 },
    { "id": "key_2", "x": 8, "y": 5 }
  ],
  "switches": [
    { "id": "switch_a", "x": 2, "y": 6 },
    { "id": "switch_b", "x": 9, "y": 6 }
  ],
  "exit": {
    "x": 10,
    "y": 3
  }
}
```

### 13.2 Go側で使用する判定

- プレイヤーが壁の中に移動していないか
- 鍵の取得可能範囲内にいるか
- スイッチの範囲内にいるか
- 出口範囲内にいるか

---

## 14. 切断・再接続仕様

### 14.1 切断検知

WebSocket接続が切れた場合、Goサーバーは対象プレイヤーを即座に削除せず、`disconnected` 状態にする。

```text
connected: false
lastSeen: 切断時刻
```

### 14.2 再接続猶予時間

再接続猶予時間は30秒とする。

```text
reconnectTimeoutSec = 30
```

### 14.3 再接続成功条件

以下をすべて満たす場合、再接続成功とする。

- roomIdが存在する
- playerIdがそのルームに所属している
- reconnectTokenが一致する
- 切断から30秒以内である
- ルームがPlaying状態である

### 14.4 再接続失敗条件

以下のいずれかに該当する場合、再接続失敗とする。

- reconnectTokenが不正
- ルームが存在しない
- 再接続猶予時間を超過
- すでにゲームが終了している

### 14.5 切断時のゲーム処理

相手が切断した場合、ゲームは一時的に停止する。

```text
1. Player A が切断
2. Room status は playing のまま保持
3. Player B には「相手の復帰待機中」と表示
4. 30秒以内に復帰した場合、GameStateとChatHistoryを再送信して再開
5. 復帰しなかった場合、game_failed として終了
```

### 14.6 再接続時に復元する情報

再接続成功時には以下を送信する。

- 現在のGameState
- 残り時間
- プレイヤー位置
- 鍵取得状態
- スイッチ状態
- 出口状態
- 直近のチャット履歴
- 接続状態

---

## 15. ランキング仕様

### 15.1 ランキング基準

ランキングはクリアタイムの短い順に表示する。

```text
1位: clearTimeSec が最も短い
```

### 15.2 ランキング対象

ランキングに登録されるのは、クリア成功した試合のみとする。  
失敗した試合は履歴には残すが、ランキング対象外とする。

### 15.3 表示項目

- 順位
- 2人のプレイヤー名
- クリアタイム
- プレイ日時

例：

```json
{
  "rank": 1,
  "players": ["player001", "player002"],
  "clearTime": 72.4,
  "playedAt": "2026-06-20T12:00:00Z"
}
```

---

## 16. エラーハンドリング方針

### 16.1 REST APIエラー例

| ステータス | 内容 |
|---|---|
| 400 | リクエスト形式が不正 |
| 401 | 認証失敗 |
| 403 | 権限なし |
| 404 | リソースが存在しない |
| 409 | すでにマッチング中 |
| 500 | サーバー内部エラー |

### 16.2 WebSocketエラー例

```json
{
  "type": "error",
  "payload": {
    "code": "INVALID_ACTION",
    "message": "This action is not allowed in current game state."
  }
}
```

### 16.3 主なエラーコード

| コード | 内容 |
|---|---|
| INVALID_TOKEN | JWTが不正 |
| ROOM_NOT_FOUND | ルームが存在しない |
| INVALID_ROOM_STATE | 現在のルーム状態では実行不可 |
| INVALID_ACTION | 不正な操作 |
| RECONNECT_FAILED | 再接続失敗 |
| GAME_ALREADY_ENDED | ゲーム終了済み |
| CHAT_EMPTY | チャット本文が空 |
| MESSAGE_TOO_LONG | チャット本文が長すぎる |
| RATE_LIMITED | チャットの連続投稿制限 |

---

## 17. セキュリティ方針

### 17.1 認証

- パスワードは平文保存しない
- bcrypt等でハッシュ化する
- 認証が必要なAPIではJWTを検証する
- **JWTには有効期限（exp）を設定する**（例: 24時間）。リフレッシュトークンは本プロジェクトのスコープ外とし、期限切れは再ログインで対応する旨を明記する。
- **JWT署名鍵・DB接続情報などのシークレットは環境変数（または Docker secrets）から読み込み**、ソースやリポジトリに含めない。
- `reconnectToken` は `crypto/rand` で生成した十分な長さのランダム値とし、サーバー側でルーム・ユーザーに紐付けて保持する。再接続成功時または猶予時間経過時に無効化する（使い回し・総当たりを防ぐ）。

### 17.2 改ざん対策

- クリアタイムはUnity側から送らせない
- 鍵取得やクリア判定はGoサーバー側で行う
- クライアントから送られた座標・結果は常に検証する
- チャットは送信者がルーム参加者か確認してから配信する

### 17.3 チャット対策

- 空文字を拒否する
- 最大文字数を制限する
- 連続投稿を制限する
- DB保存する場合、本文長を制限する

### 17.4 通信

開発環境ではHTTP / wsを使用する。  
本番相当の環境ではHTTPS / wssを使用する。

---

## 18. 実装フェーズ

### Phase 1：Unityローカルゲーム作成

- 2Dトップビューマップ作成
- Tilemap作成
- プレイヤー移動
- 鍵・スイッチ・出口の仮実装
- ローカルでクリア可能にする

### Phase 2：Go REST API作成

- Goサーバー起動
- `/api/game-config`
- `/api/register`
- `/api/login`
- JWT発行
- `/api/me`
- `/api/ranking`

### Phase 3：DB連携

- PostgreSQL導入
- usersテーブル作成
- matchesテーブル作成
- match_playersテーブル作成
- chat_messagesテーブル作成
- ユーザー登録・ログインをDB対応
- ランキング取得をDB対応

### Phase 4：マッチング実装

- マッチング開始API
- マッチング待機キュー
- 2人揃ったらroomId発行
- RoomManager作成

### Phase 5：WebSocket通信

- UnityからWebSocket接続
- GoでWebSocket接続管理
- ルーム参加
- プレイヤー入力送信
- 相手プレイヤー位置同期

### Phase 5.5：チャット機能実装

- UnityにチャットUIを追加
- WebSocketで `chat_message` を送信
- Goサーバーでチャットメッセージを受信
- 送信者がルームに所属しているか検証
- メッセージの空文字・文字数チェック
- 連続投稿制限
- 同一ルーム内のプレイヤーへ `chat_broadcast` を配信
- Room内に直近チャット履歴を保持
- 再接続時にチャット履歴を送信

### Phase 6：サーバー権威型GameState

- GameStateManager実装
- 鍵取得判定
- スイッチ判定
- 出口開放判定
- クリア判定
- ゲーム終了通知
- クリア結果保存

### Phase 7：切断・再接続

- WebSocket切断検知
- reconnectToken発行
- 一定時間Room保持
- 再接続メッセージ対応
- GameState再送信
- ChatHistory再送信
- 復帰失敗時のゲーム終了

### Phase 8：仕上げ

- UI調整
- エラー表示
- チャットUI調整
- チャットログDB保存
- チャット履歴取得API
- README作成
- API仕様書作成
- Docker Compose整備
- マイグレーションツール（golang-migrate / goose）でスキーマ管理
- `context` を用いたgraceful shutdown（進行中ルームへ終了通知 → 接続クローズ）
- 構造化ログ（log/slog）の導入
- テスト追加
- プレイ動画作成

---

## 19. ポートフォリオでの説明ポイント

本プロジェクトでは、Unity側のゲーム性をシンプルにし、Goバックエンド側の実装に重点を置く。

説明時には以下を強調する。

- Unityをクライアント、Goをサーバーとして責務分離した
- REST APIとWebSocketを用途に応じて使い分けた
- 認証、マッチング、ランキングなどのAPIを実装した
- WebSocketでルーム内のリアルタイム同期を実装した
- WebSocketをプレイヤー位置同期だけでなく、ルーム内チャットにも利用した
- 鍵取得、出口開放、クリア判定をサーバー側で行うサーバー権威型にした
- **座標もサーバーがシミュレーション（固定Tick）で計算し、クライアントは入力ベクトルのみ送る構成にして、座標改ざん・スピードハックを防いだ**
- **1ルーム=1ゴルーチンのアクターモデルで状態を単一所有者に閉じ込め、共有状態のロック競合・データ競合を避けた**
- WebSocket切断時に即終了せず、一定時間内の再接続を可能にした
- 再接続時にGameStateとチャット履歴を復元できるようにした
- クリアタイムはクライアントではなくサーバー側で計算し、改ざんを防いだ

---

## 20. 完成目標

### 20.1 最低完成ライン

- Unityで2Dマップを移動できる
- GoのREST APIに接続できる
- ログインできる
- マッチングできる
- WebSocketで2人の位置同期ができる
- WebSocketでチャットを送受信できる
- ゲームクリア結果を保存できる
- ランキングを表示できる

### 20.2 標準完成ライン

- RoomManagerがある
- GameStateManagerがある
- ChatManagerがある
- 鍵・スイッチ・出口をサーバー側で判定する
- チャットメッセージのバリデーションを行う
- クリアタイムをサーバー側で計算する
- DBに試合履歴を保存する

### 20.3 高評価完成ライン

- 切断検知がある
- reconnectTokenによる再接続がある
- 再接続時にGameStateを復元できる
- 再接続時にチャット履歴を復元できる
- チャットログをDB保存できる
- Docker ComposeでGo / PostgreSQL / Redisを起動できる
- OpenAPI仕様書がある
- Goのテストコードがある

---

## 21. プロジェクトの最終説明文

UnityとGoを用いて、2人協力型の2Dオンライン脱出ゲームを開発する。  
Unity側ではTilemapを用いたマップ表示、プレイヤー操作、UI表示、チャットUIを担当し、Go側ではREST API、WebSocket通信、マッチング、ルーム管理、ゲーム状態管理、チャット配信を担当する。

REST APIではログイン、ユーザー情報取得、マッチング、ランキング取得、試合履歴取得を実装する。  
WebSocketでは、ルーム内のプレイヤー入力、位置同期、鍵取得、スイッチ状態、出口開放、ゲーム終了通知、ルーム内チャットを扱う。

また、ゲーム結果やクリアタイムをクライアント側で確定せず、Goサーバー側で判定するサーバー権威型の構成とする。  
さらに、WebSocket切断時には一定時間ルーム状態を保持し、再接続時に現在のGameStateと直近のチャット履歴を復元することで、オンラインゲームに必要な接続管理も実装する。

本プロジェクトにより、Unityによるゲームクライアント開発だけでなく、GoによるゲームサーバーAPI開発、リアルタイム通信、状態同期、認証、DB設計、チャット機能、再接続処理までを総合的に示す。