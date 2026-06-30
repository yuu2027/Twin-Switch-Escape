# Twin Switch Escape — Phase 3 仕様書（DB 連携）

本書は親仕様書 `Twin_Switch_Escape_SPEC.md`（以下「本仕様」）の **§18 Phase 3「DB連携」** を
実装するための詳細仕様である。Phase 2（Go REST API・インメモリ）を前提に、データ永続化を
PostgreSQL へ移行する範囲を定義する。

- 関連する本仕様の節: §3.2（技術スタック）, §11（内部設計）, §12（DB設計）, §15（ランキング）, §17（セキュリティ）
- 対応コード: `server/`（Phase 2 で構築済み。本フェーズで Postgres 実装を追加）

---

## 1. 目的とゴール

### 1.1 目的

Phase 2 ではユーザー・試合データをプロセスメモリに保持していた（再起動で消える）。
Phase 3 では PostgreSQL に永続化し、以下を実現する。

- ユーザー登録・ログインの DB 対応（`users` テーブル）
- ランキング・試合履歴・自分情報の DB 対応（`matches` / `match_players` から集計導出）
- 将来のチャット履歴保存に備えたスキーマ整備（`chat_messages` テーブルは作成のみ）

### 1.2 完了条件（Definition of Done）

- [ ] PostgreSQL が docker-compose で起動できる
- [ ] マイグレーションで 4 テーブル（users / matches / match_players / chat_messages）が作成できる
- [ ] `DATABASE_URL` 設定時、`/api/register` → `/api/login` → `/api/me` が DB 経由で動作する
- [ ] 同一 username の二重登録が DB の UNIQUE 制約で 409 になる
- [ ] `/api/ranking` が `matches`(cleared) をクリアタイム昇順で返す
- [ ] サーバー再起動後もユーザー・試合データが保持されている
- [ ] `DATABASE_URL` 未設定時は従来どおりインメモリで動作する（Phase 2 互換を壊さない）

### 1.3 スコープ外（後続フェーズ）

- マッチング / RoomManager（Phase 4）
- WebSocket / 位置同期 / チャット配信（Phase 5・5.5）
- サーバー権威型 GameState・クリア判定・**試合結果の実書き込み経路**（Phase 6）
- 切断・再接続（Phase 7）
- Go アプリの Dockerfile 化・graceful shutdown・OpenAPI・テスト網羅（Phase 8）

> 注: `matches` への INSERT が実際に走るのは Phase 6（クリア確定時）。Phase 3 では
> リポジトリの `Create` を実装しておくが、本番の呼び出し経路はまだ無い。動作確認は
> `migrations/seed.sql` の手動投入で行う。

---

## 2. 全体方針

### 2.1 Repository インターフェースによる差し替え

データアクセスは `repository.UserRepository` / `repository.MatchRepository` インターフェースに
集約済み（Phase 2）。Phase 3 では**同じインターフェースを満たす PostgreSQL 実装を追加**するだけで、
上位層（`auth` サービス・`ranking` サービス・各ハンドラ）は一切変更しない。

```text
auth.Service / ranking.Service
        │ （インターフェースにのみ依存）
        ▼
repository.UserRepository / MatchRepository
        ├─ InMemory*Repository    （Phase 2, DATABASE_URL 未設定時）
        └─ Postgres*Repository    （Phase 3, DATABASE_URL 設定時）  ← 本フェーズで実装
```

### 2.2 実行時の切り替え（`DATABASE_URL`）

`cmd/api/main.go` の `buildRepositories(cfg)` が切替の単一窓口。

| `DATABASE_URL` | 使用する実装 | デモデータ |
|---|---|---|
| 空（未設定） | インメモリ | `SeedDemoMatches()` をコードで投入 |
| 設定あり | PostgreSQL | `migrations/seed.sql` を手動投入 |

この一点で切り替わるため、開発初期はインメモリ、DB 確認時のみ Postgres、と使い分けられる。

---

## 3. 技術スタック（本仕様 §3.2 準拠）

| 項目 | 採用 | 備考 |
|---|---|---|
| DB | PostgreSQL 16 | docker-compose で起動 |
| DB アクセス | 標準 `database/sql` | 汎用 API（Query/QueryRow/Exec/Tx）を学習 |
| ドライバ | `github.com/jackc/pgx/v5/stdlib` | `sql.Open("pgx", dsn)` でドライバ名 `pgx` を使用（blank import で登録） |
| マイグレーション | golang-migrate（CLI） | `migrations/*.up.sql` / `*.down.sql` |
| コンテナ | Docker Compose | DB のみ（アプリの Dockerfile 化は Phase 8） |

> ドライバに pgx を選びつつ `database/sql` 経由で使うことで、「標準インターフェースの上に
> ドライバを差す」Go の DB アクセスの基本形を学べる。pgx ネイティブ API（pgxpool）に
> 切り替える発展も可能。

---

## 4. DB スキーマ仕様（本仕様 §12 を実装に落とす）

実体は `server/migrations/000001_init_schema.up.sql`。本章はその設計意図を述べる。

### 4.1 ID 型の統一（**実装前に必ず決定すること**）

本仕様 §12 は各テーブルの `id` を **UUID** とする。一方 Phase 2 の `auth.newUserID()` は
`"user_" + hex(16バイト)` という **UUID ではない**文字列を返す。Phase 3 で DB に載せる前に、
次のいずれかへ統一する。

| 選択 | 内容 | 影響 | 推奨 |
|---|---|---|---|
| **(A) UUID 化** | `id` を UUID 型のままにし、Go 側の ID 生成を本物の UUID に変更（`github.com/google/uuid` 等）。`newUserID()` / 試合・参加者 ID を書き換え | 本仕様準拠。ポートフォリオで「DB設計どおり」を示せる | ◎ |
| (B) 文字列維持 | `"user_xxxx"` を維持し、マイグレーションの `id`/外部キー列を `VARCHAR(64)` に変更 | 既存コードの変更が少ない | △ |

本書および `000001_init_schema.up.sql` は **(A)** を既定として記述している。(B) を採る場合は
スキーマの `UUID` を `VARCHAR(64)` に読み替え、seed.sql の UUID リテラルも該当形式へ変更する。

### 4.2 テーブル定義

#### users（本仕様 §12.1）

| カラム | 型 | 制約 / 既定 | 意味 |
|---|---|---|---|
| id | UUID | PK | ユーザーID |
| username | VARCHAR(32) | **NOT NULL / UNIQUE** | ユーザー名。重複防止の要 |
| password_hash | CHAR(60) | NOT NULL | bcrypt ハッシュ（固定60文字） |
| created_at | TIMESTAMPTZ | NOT NULL / DEFAULT now() | 作成日時 |
| updated_at | TIMESTAMPTZ | NOT NULL / DEFAULT now() | 更新日時 |

- `username` の UNIQUE はアプリ側チェックに加え DB でも保証する（競合時の二重登録を防ぐ）。

#### matches（本仕様 §12.2）

| カラム | 型 | 制約 | 意味 |
|---|---|---|---|
| id | UUID | PK | 試合ID |
| room_id | VARCHAR(64) | NULL可 | 揮発・参考情報 |
| result | VARCHAR(16) | NOT NULL | `cleared` / `failed` |
| clear_time_ms | INTEGER | NULL可 | クリア時間（ミリ秒）。failed は NULL |
| failed_reason | VARCHAR(64) | NULL可 | 失敗理由。cleared は NULL |
| started_at | TIMESTAMPTZ | NOT NULL | 開始時刻 |
| ended_at | TIMESTAMPTZ | NOT NULL | 終了時刻 |
| created_at | TIMESTAMPTZ | NOT NULL / DEFAULT now() | 作成日時 |

- ランキング用の**部分インデックス**: `CREATE INDEX ... ON matches (clear_time_ms) WHERE result='cleared'`
- クリア時間は浮動小数でなく**ミリ秒整数**（同タイム比較・並べ替えの正確性, 本仕様 §9.7）。

#### match_players（本仕様 §12.3）

| カラム | 型 | 制約 | 意味 |
|---|---|---|---|
| id | UUID | PK | ID |
| match_id | UUID | FK → matches(id) ON DELETE CASCADE | 試合ID |
| user_id | UUID | FK → users(id) ON DELETE CASCADE | ユーザーID |
| created_at | TIMESTAMPTZ | NOT NULL / DEFAULT now() | 作成日時 |

- `UNIQUE (match_id, user_id)`：同一試合への二重登録防止。
- `user_id` にインデックス（`/api/matches/me` 用）。

#### chat_messages（本仕様 §12.4。Phase 3 ではテーブル作成のみ）

| カラム | 型 | 制約 | 意味 |
|---|---|---|---|
| id | UUID | PK | メッセージID |
| match_id | UUID | FK → matches(id) ON DELETE CASCADE | 試合ID |
| sender_id | UUID | FK → users(id) ON DELETE CASCADE | 送信者ID |
| message | VARCHAR(100) | NOT NULL | 本文（長さ制限 §10.4 / §17.3） |
| sent_at | TIMESTAMPTZ | NOT NULL | 送信日時 |
| created_at | TIMESTAMPTZ | NOT NULL / DEFAULT now() | 作成日時 |

- `(match_id, sent_at)` の複合インデックス（履歴取得用）。

---

## 5. マイグレーション運用

- ツール: golang-migrate CLI（goose でも可）。
- 命名: `NNNNNN_説明.up.sql` / `NNNNNN_説明.down.sql`。
- 適用 / ロールバック:

```bash
export DATABASE_URL="postgres://twin:twin_pass@localhost:5432/twin_switch?sslmode=disable"
migrate -path ./migrations -database "$DATABASE_URL" up      # 適用
migrate -path ./migrations -database "$DATABASE_URL" down 1  # 1つ戻す
```

- `down.sql` は外部キー依存の子テーブルから先に DROP する（chat_messages → match_players → matches → users）。
- `migrations/seed.sql` は**マイグレーションではない**動作確認用データ（手動 `psql -f`）。

---

## 6. リポジトリ実装仕様

対応ファイル:
`server/internal/db/db.go`,
`server/internal/repository/postgres_user_repository.go`,
`server/internal/repository/postgres_match_repository.go`。
各メソッドの「正しい振る舞い」はインメモリ実装（同パッケージ）がリファレンスになる。

### 6.1 接続（`internal/db/db.go`）

`Open(dsn) (*sql.DB, error)`：

1. `sql.Open("pgx", dsn)`（遅延接続）
2. プール設定（推奨）: `SetMaxOpenConns(10)` / `SetMaxIdleConns(5)` / `SetConnMaxLifetime(time.Hour)`
3. `Ping`（or `PingContext` + タイムアウト）で疎通確認。失敗時は `Close` して error
4. `*sql.DB` を返す（アプリ全体で 1 インスタンス共有）

### 6.2 PostgresUserRepository

プレースホルダは PostgreSQL の `$1, $2, ...` を使う。

| メソッド | SQL（要旨） | エラー方針 |
|---|---|---|
| `Create(u)` | `INSERT INTO users (id, username, password_hash, created_at, updated_at) VALUES ($1..$5)` | UNIQUE 違反（PgError `Code=="23505"`, `errors.As` で判定）→ `ErrUsernameTaken` |
| `FindByUsername(name)` | `SELECT ... FROM users WHERE username=$1` | `sql.ErrNoRows` → `ErrUserNotFound` |
| `FindByID(id)` | `SELECT ... FROM users WHERE id=$1` | 同上 |

- 既存の `ErrUsernameTaken` / `ErrUserNotFound`（`repository` パッケージ）をそのまま使う。
  これにより `auth.Service` のエラー分岐（409 / 401）が無変更で機能する。

### 6.3 PostgresMatchRepository

**`PlayerIDs` の扱い（重要）**: インメモリでは表示名を入れて簡略化していた。Postgres では
`match_players(user_id)` に UUID が入るため、ランキング表示（本仕様 §15.3 はプレイヤー名）に
合わせて **JOIN で `users.username` を取得し `Match.PlayerIDs` に詰める**。これで `ranking` 層は無変更。

| メソッド | SQL（要旨） | 備考 |
|---|---|---|
| `Create(m)` | `matches` と `match_players` へ INSERT | **トランザクション**（`db.Begin` → 複数 Exec → `Commit`/`Rollback`）。Phase 6 で使用 |
| `ListClearedRankedByTime(limit)` | `matches JOIN match_players JOIN users` を `GROUP BY m.id`、`array_agg(u.username)` で players 集約、`WHERE result='cleared' ORDER BY clear_time_ms ASC LIMIT $1` | `limit<=0` は LIMIT を外す等で分岐。配列カラムは `[]string` へスキャン |
| `ListByUserID(userID)` | `matches m JOIN match_players mp ON mp.match_id=m.id WHERE mp.user_id=$1 ORDER BY m.ended_at DESC` | `/api/matches/me` 用 |
| `Stats(userID)` | `SELECT COUNT(*) FILTER (WHERE result='cleared'), COALESCE(MIN(clear_time_ms) FILTER (WHERE result='cleared'),0) FROM matches m JOIN match_players mp ... WHERE mp.user_id=$1` | `/api/me` の bestClearTime/clearCount を 1 クエリで導出（本仕様 §12.1） |

---

## 7. 設定・環境変数

| 変数 | 必須 | 既定 | 用途 |
|---|---|---|---|
| `JWT_SECRET` | ✅ | — | JWT 署名鍵（本仕様 §17.1） |
| `PORT` | — | 8080 | リッスンポート |
| `JWT_EXPIRY_HOURS` | — | 24 | アクセストークン有効期限 |
| `DATABASE_URL` | — | 空=インメモリ | PostgreSQL 接続文字列 |

接続文字列例:
`postgres://twin:twin_pass@localhost:5432/twin_switch?sslmode=disable`
（`docker-compose.yml` の認証情報と一致させる。シークレットはソースに埋め込まない）

---

## 8. セットアップ・実行手順

```bash
cd server

# 1. DB 起動
docker compose up -d

# 2. マイグレーション適用
export DATABASE_URL="postgres://twin:twin_pass@localhost:5432/twin_switch?sslmode=disable"
migrate -path ./migrations -database "$DATABASE_URL" up

# 3. （任意）デモデータ投入
psql "$DATABASE_URL" -f migrations/seed.sql

# 4. Postgres モードで起動
JWT_SECRET=dev DATABASE_URL="$DATABASE_URL" go run ./cmd/api
```

---

## 9. 受け入れテスト（手動 / curl）

```bash
BASE=http://localhost:8080

# 登録（DB へ INSERT）
curl -s -X POST $BASE/api/register -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"password123"}'

# 二重登録 → 409（DB の UNIQUE 制約）
curl -s -i -X POST $BASE/api/register -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"password123"}' | head -1

# ログイン → accessToken
TOKEN=$(curl -s -X POST $BASE/api/login -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"password123"}' | sed -n 's/.*"accessToken":"\([^"]*\)".*/\1/p')

# 自分情報（matches からの集計。試合が無ければ best=0,count=0）
curl -s $BASE/api/me -H "Authorization: Bearer $TOKEN"

# ランキング（seed.sql 投入後、cleared を時間昇順で）
curl -s $BASE/api/ranking

# 永続性確認: サーバー再起動 → 再度 /api/login が成功すること
```

合格条件は §1.2「完了条件」を参照。

---

## 10. 実装チェックリスト

- [ ] ID 型の方針を (A)/(B) で決定（§4.1）。(A) なら `newUserID()` 等を UUID 生成へ変更
- [ ] `docker compose up -d` で DB 起動を確認
- [ ] `migrate ... up` で 4 テーブル + インデックス作成を確認
- [ ] `internal/db/db.go` `Open` 実装（プール + Ping）
- [ ] `postgres_user_repository.go` 3 メソッド実装（UNIQUE→`ErrUsernameTaken`, NoRows→`ErrUserNotFound`）
- [ ] `postgres_match_repository.go` 4 メソッド実装（JOIN・`array_agg`・`FILTER` 集計・トランザクション）
- [ ] `DATABASE_URL` 設定で register→login→me→ranking が通る
- [ ] 再起動後もデータが残る（永続性）
- [ ] `DATABASE_URL` 未設定でインメモリ動作が維持されている（回帰確認）
- [ ] `go build ./... && go vet ./...` がパス

---

## 11. 設計上のトレードオフ（ポートフォリオ説明用）

- **Repository パターン**で永続化を抽象化し、インメモリ ⇄ DB を 1 箇所で差し替え可能にした。
  上位のビジネスロジック・HTTP 層を変更せずにストレージを移行できる。
- **ランキング専用テーブルを持たず** `matches` から都度導出（部分インデックスで高速化）。
  状態の二重管理を避ける設計（本仕様 §12.2 / §15）。
- **`database/sql` + pgx ドライバ**で、標準インターフェースの上にドライバを差す Go の基本形を採用。
  将来 pgxpool ネイティブへ移行する余地も残す。
- **UNIQUE 制約・外部キー・トランザクション**を DB 側で効かせ、整合性をアプリ任せにしない。
