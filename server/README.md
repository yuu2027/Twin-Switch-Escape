# Twin Switch Escape - Server (Phase 2: REST API / Phase 3: DB連携)

仕様書 `../Twin_Switch_Escape_SPEC.md` の **Phase 2（Go REST API作成）** および
**Phase 3（DB連携）** に対応するバックエンド。

- 標準ライブラリ中心（`net/http` + Go 1.22 `ServeMux`, `database/sql`）
- 認証は JWT（`golang-jwt/jwt/v5`）+ bcrypt
- データ層は Repository インターフェースで抽象化し、**インメモリ ⇄ PostgreSQL** を `DATABASE_URL` で切替
- DB ドライバは pgx の `database/sql` アダプタ（`github.com/jackc/pgx/v5/stdlib`）

> 本ディレクトリは**学習用のひな型**です。各ファイルの `// TODO:` を埋めると動作します。
> ハンドラ→サービス→リポジトリの依存方向と、各 `panic("TODO...")` を上から潰していくのがおすすめ。

## ディレクトリ

```
cmd/api/main.go          エントリポイント（DI とルーティング, リポジトリ切替）
internal/
  config/                環境変数の読込・ゲーム設定既定値（DATABASE_URL 含む）
  db/                    PostgreSQL 接続（database/sql プール, Phase 3）
  httpx/                 JSON / エラーレスポンス共通ヘルパー
  models/                User / Match などのドメインモデル
  repository/            UserRepository / MatchRepository
                         ├ インメモリ実装（Phase 2）
                         └ PostgreSQL 実装（Phase 3）← 同じインターフェースを満たす
  auth/                  jwt / service / handler（register・login・me）
  middleware/            Bearer トークン検証ミドルウェア
  gameconfig/            GET /api/game-config
  ranking/               GET /api/ranking
migrations/              DB スキーマ（golang-migrate 形式）+ seed.sql
docker-compose.yml       開発用 PostgreSQL
```

## 実装の進め方（TODO を埋める順番の目安）

1. `config.Load`（環境変数）
2. `httpx.WriteJSON` / `WriteError`
3. `repository` のインメモリ実装（map / slice + sync.RWMutex）
4. `auth/jwt.go`（Issue / Parse）
5. `auth/service.go`（Register / Login / Me, bcrypt）
6. `auth/handler.go`・`middleware`・`gameconfig`・`ranking`

## 起動

```bash
# 依存取得（初回のみ）
go mod tidy

# 起動（JWT_SECRET は必須）
JWT_SECRET=dev-secret go run ./cmd/api
#   PowerShell: $env:JWT_SECRET="dev-secret"; go run ./cmd/api
```

## 動作確認（curl）

```bash
# ゲーム設定（認証不要, spec §7.2）
curl localhost:8080/api/game-config

# 登録（spec §7.1）
curl -X POST localhost:8080/api/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"player001","password":"password123"}'

# ログイン → accessToken を取得
curl -X POST localhost:8080/api/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"player001","password":"password123"}'

# 自分の情報（要 Bearer, spec §7.1）
curl localhost:8080/api/me -H "Authorization: Bearer <accessToken>"

# ランキング（spec §7.4）
curl localhost:8080/api/ranking
```

## Phase 3: DB 連携（PostgreSQL）

Repository インターフェースはそのままに、実装をインメモリ → PostgreSQL に差し替える。
`DATABASE_URL` を設定すると Postgres を、空ならインメモリを使う（`main.go` の `buildRepositories`）。

### 1. DB を起動

```bash
docker compose up -d        # PostgreSQL 16 が localhost:5432 に立つ
```

### 2. マイグレーション適用（golang-migrate CLI を利用）

```bash
# CLI 導入（未導入なら）: https://github.com/golang-migrate/migrate
export DATABASE_URL="postgres://twin:twin_pass@localhost:5432/twin_switch?sslmode=disable"
migrate -path ./migrations -database "$DATABASE_URL" up

# （任意）動作確認用デモデータ
psql "$DATABASE_URL" -f migrations/seed.sql
```

### 3. Postgres モードで起動

```bash
JWT_SECRET=dev DATABASE_URL="postgres://twin:twin_pass@localhost:5432/twin_switch?sslmode=disable" \
  go run ./cmd/api
```

### 実装する TODO（Phase 3 の埋める順番）

1. `internal/db/db.go` … `Open`（`sql.Open("pgx", dsn)` + プール設定 + `Ping`）
2. `internal/repository/postgres_user_repository.go` … Create / FindByUsername / FindByID
   - UNIQUE 違反（`23505`）を `ErrUsernameTaken` に変換するのがポイント
3. `internal/repository/postgres_match_repository.go` … ランキング/集計の JOIN クエリ、Create はトランザクション

> 各ファイルの `// TODO:` に SQL 例と手順を記載済み。インメモリ実装（同パッケージ）が
> 「期待する振る舞い」のリファレンスになる。

### 設計メモ（重要）

- **ID 型**: spec §12 は id を UUID とする。Phase 2 の `newUserID()` は `"user_xxxx"` を返すため、
  Phase 3 では (A) 本物の UUID 生成に切替（`github.com/google/uuid` 等, spec 準拠・推奨）か、
  (B) スキーマの id 列を `VARCHAR(64)` にするか、どちらかへ統一する。
  `migrations/000001_init_schema.up.sql` の冒頭コメント参照。
- **ランキングの players**: Postgres では `match_players`→`users` を JOIN して
  ユーザー名を `Match.PlayerIDs` に詰める（`ranking` 層を無変更に保つため）。

## Phase 4: マッチング実装

詳細は `../Twin_Switch_Escape_Phase4_SPEC.md`。2人を待機キューで突き合わせ、揃ったら `roomId` を発行する。
キューは **アクターモデル**（単一ゴルーチン + channel）で扱い、`RoomManager` は RWMutex で保護する（spec §9.3）。

### エンドポイント（すべて要認証, spec §7.3）

```
POST /api/matchmaking/start    → { "status": "waiting", "matchmakingId": "mm_..." }
GET  /api/matchmaking/status   → { "status": "waiting" } / { "status":"matched", "roomId","websocketUrl" } / { "status":"timeout" }
POST /api/matchmaking/cancel   → { "status": "cancelled" }
```

### 追加の環境変数

| 変数 | 既定 | 用途 |
|---|---|---|
| `MATCHMAKING_TIMEOUT_SEC` | 60 | 待機タイムアウト秒 |
| `WS_BASE_URL` | `ws://localhost:8080` | `websocketUrl` の基底 |

### 実装する TODO（Phase 4 の埋める順番）

1. `internal/room/manager.go` … `Create`（roomId 発行）/ `Get`（RWMutex 保護）
2. `internal/matchmaking/manager.go` … `handleStart` / `handleStatus` / `handleCancel` / `handleTick`
   - **アクターの配線（`Run`/`send`/公開メソッド）と handler は実装済み**。マッチングのドメインロジックだけ埋める。

> ポイント: `handleX` は `Run` の単一ゴルーチンからのみ呼ばれるので、`byUser` をロックなしで読み書きしてよい。

### 動作確認

2人分のトークンを用意し、A→start（waiting）/ B→start（2人成立）/ 各 status（matched で同じ roomId）を確認。
詳細な curl 手順は Phase4 SPEC §8 を参照。競合検証は `go test -race ./internal/matchmaking/...`。

## スコープ外（後続フェーズ）

- WebSocket・チャット・GameState・再接続（Phase 5〜7）
- Go アプリの Dockerfile 化・graceful shutdown・OpenAPI・テスト（Phase 8）
