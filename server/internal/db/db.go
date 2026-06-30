// Package db は PostgreSQL への接続（database/sql のコネクションプール）を提供する。
//
// 設計方針:
//   - 標準の database/sql を使い、ドライバだけ pgx の stdlib アダプタを使う
//     （github.com/jackc/pgx/v5/stdlib）。これで Query/QueryRow/Exec という
//     どの DB でも通用する API を学べる。
//   - *sql.DB は内部にコネクションプールを持つので、アプリ全体で1つ生成して使い回す。
package db

import (
	"database/sql"
	"time"

	// ドライバ登録のための blank import。これにより database/sql で
	// sql.Open("pgx", dsn) が使えるようになる（init() でドライバ "pgx" を登録）。
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Open は DATABASE_URL から *sql.DB（プール）を作り、疎通確認して返す。
//  1. db, err := sql.Open("pgx", dsn)   // ドライバ名は blank import した stdlib の "pgx"
//     - sql.Open は実接続しない（遅延接続）点に注意。
//  2. プール設定（任意だが推奨）:
//     db.SetMaxOpenConns(10)
//     db.SetMaxIdleConns(5)
//     db.SetConnMaxLifetime(time.Hour)
//  3. db.Ping() で実際に接続できるか確認。失敗したら db.Close() して error を返す。
//     （より丁寧にやるなら context.WithTimeout + db.PingContext）
//  4. *sql.DB を返す。
func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(10)           // 同時に開けるDB接続の最大数
	db.SetMaxIdleConns(5)            // 使い終わった後に待機状態で残しておく接続数
	db.SetConnMaxLifetime(time.Hour) // 1つの接続を最大どれくらい使い続けるか

	if err := db.Ping(); err != nil { // 接続を確認
		db.Close()
		return nil, err
	}

	return db, nil
}
