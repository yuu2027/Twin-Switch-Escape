-- 000001_init_schema のロールバック。
-- 外部キーの依存があるため、参照している側（子テーブル）から先に削除する。
DROP TABLE IF EXISTS chat_messages;
DROP TABLE IF EXISTS match_players;
DROP TABLE IF EXISTS matches;
DROP TABLE IF EXISTS users;
