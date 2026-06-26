# Twin Switch Escape - Server (Phase 2: REST API)

仕様書 `../Twin_Switch_Escape_SPEC.md` の **Phase 2（Go REST API作成）** に対応するバックエンド。

- 標準ライブラリ中心（`net/http` + Go 1.22 `ServeMux`）
- 認証は JWT（`golang-jwt/jwt/v5`）+ bcrypt
- データは **インメモリ**保持（Phase 3 で PostgreSQL に差し替え予定）

> 本ディレクトリは**学習用のひな型**です。各ファイルの `// TODO:` を埋めると動作します。
> ハンドラ→サービス→リポジトリの依存方向と、各 `panic("TODO...")` を上から潰していくのがおすすめ。

## ディレクトリ

```
cmd/api/main.go          エントリポイント（DI とルーティング）
internal/
  config/                環境変数の読込・ゲーム設定既定値
  httpx/                 JSON / エラーレスポンス共通ヘルパー
  models/                User / Match などのドメインモデル
  repository/            UserRepository / MatchRepository（インターフェース + インメモリ実装）
  auth/                  jwt / service / handler（register・login・me）
  middleware/            Bearer トークン検証ミドルウェア
  gameconfig/            GET /api/game-config
  ranking/               GET /api/ranking
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

## スコープ外（後続フェーズ）

- PostgreSQL / マイグレーション（Phase 3）
- マッチング・WebSocket・チャット・GameState・再接続（Phase 4〜7）
