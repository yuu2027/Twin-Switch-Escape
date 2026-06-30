package repository

import (
	"database/sql"
	"strings"
	"time"

	"twin-switch-escape/server/internal/models"

	"github.com/google/uuid"
)

// PostgresMatchRepository は MatchRepository の PostgreSQL 実装（spec §12.2 / §12.3）。
//
//	インメモリ実装では Match.PlayerIDs に「表示名」をそのまま入れて簡略化していた。
//	Postgres では正規化され、match_players(user_id) に UUID が入る。ランキング表示には
//	ユーザー名が必要（spec §15.3）なので、ここでは JOIN で users.username を取り出して
//	Match.PlayerIDs に「ユーザー名」を詰めて返す方針にする（上位の ranking 層を無変更にするため）。
//	※ もし PlayerIDs に user_id を入れたい場合は、ranking 側で名前解決する設計に変える。
type PostgresMatchRepository struct {
	db *sql.DB
}

// &PostgresMatchRepository{db: db} を返す。
func NewPostgresMatchRepository(db *sql.DB) *PostgresMatchRepository {
	return &PostgresMatchRepository{
		db: db,
	}
}

// Create は試合結果 + 参加者を保存する（Phase 6 のクリア確定時に使う想定）。
//
//	matches と match_players の2テーブルにまたがるので「トランザクション」で書く。
//	  1. tx, err := r.db.Begin()
//	  2. tx.Exec("INSERT INTO matches (...) VALUES ($1..$8)", ...)
//	  3. m.PlayerIDs（=参加 userId）ごとに
//	     tx.Exec("INSERT INTO match_players (id, match_id, user_id) VALUES ($1,$2,$3)", ...)
//	     ※ match_players.id も UUID を採番する（newMatchPlayerID 等）。
//	  4. すべて成功したら tx.Commit()。途中で失敗したら tx.Rollback() して error を返す。
//	     （defer で「Commit 済みでなければ Rollback」にしておくと安全）
//
// 複数テーブルへの書き込みは「全部成功か全部失敗か」を保証するため
// トランザクションでまとめる。Go の database/sql では *sql.Tx を使う。
func (r *PostgresMatchRepository) Create(m *models.Match) error {
	tx, err := r.db.Begin() // トランザクション開始処理
	if err != nil {
		return err
	}

	committed := false
	defer func() { // 関数が終了する直前に実行
		if !committed {
			_ = tx.Rollback() // トランザクション中に行ったDB操作を取り消す処理
		}
	}()

	const matchQuery = `
		INSERT INTO matches (id, room_id, result, clear_time_ms, failed_reason, started_at, ended_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
	if _, err := tx.Exec(
		matchQuery,
		m.ID,
		nullString(m.RoomID),
		m.Result,
		nullClearTime(m),
		nullString(m.FailedReason),
		m.StartedAt,
		m.EndedAt,
		m.CreatedAt,
	); err != nil {
		return err
	}

	const playerQuery = `
		INSERT INTO match_players (id, match_id, user_id)
		VALUES ($1, $2, $3)
		`
	for _, userID := range m.PlayerIDs {
		if _, err := tx.Exec(playerQuery, newMatchPlayerID(), m.ID, userID); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil { // トランザクション中のDB操作を正式に確定する処理
		return err
	}
	committed = true
	return nil
}

// ListClearedRankedByTime は cleared 試合をクリアタイム昇順で返す（spec §15.1）。
// PlayerIDs にはユーザー名を詰める（上記設計メモ参照）。
//  1. rows, err := r.db.Query(query, limit)   // limit<=0 のときは LIMIT を外す等の分岐
//  2. defer rows.Close()
//  3. for rows.Next(): rows.Scan(...) で1行ずつ Match へ。
//     - array_agg の players は pq.Array / pgx の配列スキャンで []string に受ける。
//     （database/sql + pgx stdlib では `var players []string` に直接 Scan できる場合がある。
//     うまくいかなければ pgtype を使う。ここは実装時に調べるポイント）
//  4. rows.Err() を確認して返す。
func (r *PostgresMatchRepository) ListClearedRankedByTime(limit int) ([]*models.Match, error) {
	query := `
		SELECT m.id, m.room_id, m.clear_time_ms, m.started_at, m.ended_at, m.created_at,
				string_agg(u.username, ',' ORDER BY mp.created_at) AS players
		FROM matches m
		JOIN match_players mp ON mp.match_id = m.id
		JOIN users u          ON u.id = mp.user_id
		WHERE m.result = 'cleared'
		GROUP BY m.id
		ORDER BY m.clear_time_ms ASC` // 昇順に並べる

	args := []any{}
	if limit > 0 {
		query += ` LIMIT $1` // 先頭から数行取り出す
		args = append(args, limit)
	}

	rows, err := r.db.Query(query, args...) // 可変長引数
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []*models.Match
	for rows.Next() {
		var (
			id        string
			roomID    sql.NullString // NULLを扱えるstring型
			clearMs   int
			startedAt time.Time
			endedAt   time.Time
			createdAt time.Time
			players   string
		)
		if err := rows.Scan(&id, &roomID, &clearMs, &startedAt, &endedAt, &createdAt, &players); err != nil {
			return nil, err
		}
		matches = append(matches, &models.Match{
			ID:          id,
			RoomID:      roomID.String,
			Result:      "cleared",
			ClearTimeMs: clearMs,
			PlayerIDs:   strings.Split(players, ","),
			StartedAt:   startedAt,
			EndedAt:     endedAt,
			CreatedAt:   createdAt,
		})
	}
	return matches, rows.Err()
}

// ListByUserID は指定ユーザーが参加した試合を新しい順で返す（spec §7.4 /api/matches/me 用）。
//
//	（PlayerIDs まで詰めるかは用途次第。/api/me の集計には不要なので空でもよい）
func (r *PostgresMatchRepository) ListByUserID(userID string) ([]*models.Match, error) {
	query := `
		SELECT m.id, m.room_id, m.result, m.clear_time_ms, m.failed_reason, m.started_at, m.ended_at, m.created_at
		FROM matches m
		JOIN match_players mp ON mp.match_id = m.id
		WHERE mp.user_id = $1
		ORDER BY m.ended_at DESC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []*models.Match
	for rows.Next() {
		var (
			id        string
			roomID    sql.NullString // NULLを扱えるstring型
			result    string
			clearMs   sql.NullInt32
			reason    sql.NullString
			startedAt time.Time
			endedAt   time.Time
			createdAt time.Time
		)
		if err := rows.Scan(&id, &roomID, &result, &clearMs, &reason, &startedAt, &endedAt, &createdAt); err != nil {
			return nil, err
		}
		matches = append(matches, &models.Match{
			ID:           id,
			RoomID:       roomID.String,
			Result:       result,
			ClearTimeMs:  int(clearMs.Int32), // NULL → 0
			FailedReason: reason.String,
			StartedAt:    startedAt,
			EndedAt:      endedAt,
			CreatedAt:    createdAt,
		})
	}
	return matches, rows.Err()
}

// Stats は /api/me 用の集計（spec §7.1, §12.1）。専用カラムを持たず集計で導出。
//  1. row := r.db.QueryRow(query, userID)
//  2. row.Scan(&bestMs, &clearCount)
//  3. cleared が無ければ best=0, count=0 になる（COALESCE で 0）。
func (r *PostgresMatchRepository) Stats(userID string) (bestClearTimeMs int, clearCount int, err error) {
	query := `
		SELECT 
		COUNT(*) FILTER (WHERE m.result = 'cleared'),
		COALESCE(MIN(m.clear_time_ms) FILTER (WHERE m.result = 'cleared'), 0)
		FROM matches m
		JOIN match_players mp ON mp.match_id = m.id
		WHERE mp.user_id = $1`

	if err := r.db.QueryRow(query, userID).Scan(&clearCount, &bestClearTimeMs); err != nil {
		return 0, 0, err
	}

	return bestClearTimeMs, clearCount, nil
}

func newMatchPlayerID() string {
	return uuid.NewString()
}

// 空文字なら NULL、それ以外はその値を返す
func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// cleared のときだけクリアタイムを入れ、failed は NULL
func nullClearTime(m *models.Match) any {
	if m.Result == "cleared" {
		return m.ClearTimeMs
	}
	return nil
}

var _ MatchRepository = (*PostgresMatchRepository)(nil)
