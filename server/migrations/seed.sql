-- 開発用デモデータ（任意）。マイグレーションではなく手動投入用。
--   psql "$DATABASE_URL" -f migrations/seed.sql
--
-- 注意: これは「マイグレーション」ではないので golang-migrate の管理対象外。
-- ランキング/ /api/me を Postgres で動作確認するための種データ。
-- インメモリ実装の SeedDemoMatches（repository/match_repository.go）と同じ意図。
--
-- ID は spec 準拠の UUID リテラルで記述している。Go 側で "user_xxxx" 形式を採用した場合は
-- id 列を VARCHAR にし、ここの値もその形式に合わせること（init schema のコメント参照）。

-- デモユーザー（password_hash は "password123" を bcrypt 化した例。実値は各自で置き換える）
INSERT INTO users (id, username, password_hash) VALUES
  ('11111111-1111-1111-1111-111111111111', 'player001', '$2a$10$REPLACE_WITH_REAL_BCRYPT_HASH______________________'),
  ('22222222-2222-2222-2222-222222222222', 'player002', '$2a$10$REPLACE_WITH_REAL_BCRYPT_HASH______________________')
ON CONFLICT (username) DO NOTHING;

-- デモ試合（cleared）
INSERT INTO matches (id, room_id, result, clear_time_ms, started_at, ended_at) VALUES
  ('aaaaaaaa-0000-0000-0000-000000000001', 'demo_room_001', 'cleared', 72400, now() - interval '30 minutes', now() - interval '29 minutes')
ON CONFLICT (id) DO NOTHING;

-- 参加者
INSERT INTO match_players (id, match_id, user_id) VALUES
  ('bbbbbbbb-0000-0000-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111'),
  ('bbbbbbbb-0000-0000-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', '22222222-2222-2222-2222-222222222222')
ON CONFLICT (match_id, user_id) DO NOTHING;
