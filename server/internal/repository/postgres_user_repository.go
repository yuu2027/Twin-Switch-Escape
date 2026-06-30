package repository

import (
	"database/sql"
	"errors"

	"twin-switch-escape/server/internal/models"

	"github.com/jackc/pgx/v5/pgconn"
)

// PostgresUserRepository は UserRepository の PostgreSQL 実装（spec §12.1）。
// インメモリ実装（InMemoryUserRepository）と同じインターフェースを満たすので、
// main.go の組み立てを差し替えるだけで上位層（auth など）は無変更で動く。
type PostgresUserRepository struct {
	db *sql.DB
}

// &PostgresUserRepository{db: db} を返す。
func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{
		db: db,
	}
}

// Create はユーザーを1件 INSERT する。
//
//	SQL（プレースホルダは PostgreSQL の $1, $2... を使う）:
//	  INSERT INTO users (id, username, password_hash, created_at, updated_at)
//	  VALUES ($1, $2, $3, $4, $5)
//	手順:
//	  1. r.db.Exec(query, u.ID, u.Username, u.PasswordHash, u.CreatedAt, u.UpdatedAt)
//	  2. username の UNIQUE 制約違反（重複）は ErrUsernameTaken に変換して返す。
//	     - pgx では *pgconn.PgError の Code == "23505"（unique_violation）で判定できる。
//	       errors.As(err, &pgErr) を使う。これにより「DB が一意性を保証」を体感する。
//	  3. 成功なら nil。
//
// 学習ポイント: アプリ側チェックだけでなく DB の UNIQUE 制約で重複を防ぐのが堅牢
// （spec §12.1）。競合状態でも二重登録されない。
func (r *PostgresUserRepository) Create(u *models.User) error {
	const query = `
		INSERT INTO users (id, username, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.Exec(query, u.ID, u.Username, u.PasswordHash, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrUsernameTaken
		}
		return err
	}

	return nil
}

// 1. row := r.db.QueryRow(query, username)
// 2. row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
// 3. err == sql.ErrNoRows なら ErrUserNotFound を返す。
// 4. それ以外の err はそのまま返す。成功なら *u, nil。
func (r *PostgresUserRepository) FindByUsername(username string) (*models.User, error) {
	query := `
		SELECT id, username, password_hash, created_at, updated_at
		FROM users WHERE username = $1
		`

	var u models.User
	err := r.db.QueryRow(query, username).Scan(
		&u.ID,
		&u.Username,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &u, nil
}

// FindByUsername と同様。WHERE id = $1 にするだけ。
func (r *PostgresUserRepository) FindByID(id string) (*models.User, error) {
	const query = `
		SELECT id, username, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1
		`

	var u models.User
	err := r.db.QueryRow(query, id).Scan(
		&u.ID,
		&u.Username,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &u, nil
}

// インターフェース充足のコンパイル時チェック。
var _ UserRepository = (*PostgresUserRepository)(nil)
