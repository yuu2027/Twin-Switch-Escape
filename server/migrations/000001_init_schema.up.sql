-- Phase 3: 初期スキーマ（spec §12 に対応）
-- マイグレーションツール: golang-migrate を想定（goose でも可）。
--   適用例: migrate -path ./migrations -database "$DATABASE_URL" up
--
-- 【重要な設計判断: ID 型について】
-- spec §12 は各テーブルの id を UUID 型としている。一方 Phase 2 の Go 実装では
-- newUserID() が "user_xxxx"（16バイトの hex）という UUID ではない文字列を生成している。
-- Phase 3 で DB 対応する際は、次のどちらかに統一する必要がある:
--   (A) spec どおり UUID 型にする → Go 側の ID 生成を github.com/google/uuid 等の
--       本物の UUID に切り替える（newUserID/newMatchID を書き換える）。← spec 準拠でおすすめ。
--   (B) "user_xxxx" 形式を維持する → 下記 id 列の型を UUID ではなく VARCHAR(64) にする。
-- 本ファイルは (A)（spec 準拠の UUID）で記述している。(B) を選ぶ場合は id/外部キーの
-- 型を VARCHAR(64) に読み替えること。
--
-- gen_random_uuid() を DB 側デフォルトに使う場合は pgcrypto 拡張、または PostgreSQL 13+ の
-- 組み込み関数が必要。本プロジェクトでは ID を Go 側で採番して INSERT するため DEFAULT は付けない。

-- ユーザー（spec §12.1）
CREATE TABLE users (
    id            UUID         PRIMARY KEY,
    username      VARCHAR(32)  NOT NULL UNIQUE,   -- 登録時の重複防止（spec §12.1）
    password_hash CHAR(60)     NOT NULL,          -- bcrypt ハッシュ（固定60文字）
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- 試合（spec §12.2）
CREATE TABLE matches (
    id            UUID         PRIMARY KEY,
    room_id       VARCHAR(64),                    -- 揮発・参考情報
    result        VARCHAR(16)  NOT NULL,          -- 'cleared' | 'failed'
    clear_time_ms INTEGER,                        -- cleared のときのみ。failed は NULL
    failed_reason VARCHAR(64),                    -- failed のときのみ。cleared は NULL
    started_at    TIMESTAMPTZ  NOT NULL,
    ended_at      TIMESTAMPTZ  NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- ランキング導出用の部分インデックス（spec §12.2）
--   SELECT ... FROM matches WHERE result='cleared' ORDER BY clear_time_ms ASC
CREATE INDEX idx_matches_cleared_time ON matches (clear_time_ms) WHERE result = 'cleared';

-- 試合参加者（spec §12.3）
CREATE TABLE match_players (
    id         UUID         PRIMARY KEY,
    match_id   UUID         NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    user_id    UUID         NOT NULL REFERENCES users(id)   ON DELETE CASCADE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (match_id, user_id)                    -- 同一試合への二重登録防止（spec §12.3）
);

-- /api/matches/me 用（spec §12.3）
CREATE INDEX idx_match_players_user ON match_players (user_id);

-- チャットメッセージ（spec §12.4。Phase 3 ではテーブルのみ作成、利用は Phase 5.5/8）
CREATE TABLE chat_messages (
    id         UUID         PRIMARY KEY,
    match_id   UUID         NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    sender_id  UUID         NOT NULL REFERENCES users(id)   ON DELETE CASCADE,
    message    VARCHAR(100) NOT NULL,             -- 本文長制限（spec §10.4 / §17.3）
    sent_at    TIMESTAMPTZ  NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- 履歴取得（/api/matches/{matchId}/chats）用の複合インデックス（spec §12.4）
CREATE INDEX idx_chat_messages_match_sent ON chat_messages (match_id, sent_at);
