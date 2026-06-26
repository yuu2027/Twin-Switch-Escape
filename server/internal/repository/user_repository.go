// Package repository はデータ永続化の抽象（インターフェース）と、
// Phase 2 用のインメモリ実装を提供する。
//
// 設計意図（spec §11.1）: ハンドラ/サービス層はインターフェースにのみ依存する。
// Phase 3 でインメモリ実装を PostgreSQL 実装に差し替えても上位層を変更しないで済む。
package repository

import (
	"errors"
	"sync"

	"twin-switch-escape/server/internal/models"
)

// ErrUsernameTaken は username の重複時に Create が返す（spec §12.1 の UNIQUE 制約相当）。
var ErrUsernameTaken = errors.New("username already taken")

// ErrUserNotFound は対象ユーザーが存在しないとき返す。
var ErrUserNotFound = errors.New("user not found")

// UserRepository はユーザーの永続化の抽象。
type UserRepository interface {
	// Create は新規ユーザーを保存する。username 重複なら ErrUsernameTaken。
	Create(u *models.User) error
	// FindByUsername は username で1件取得。なければ ErrUserNotFound。
	FindByUsername(username string) (*models.User, error)
	// FindByID は userId で1件取得。なければ ErrUserNotFound。
	FindByID(id string) (*models.User, error)
}

type InMemoryUserRepository struct {
	// TODO: フィールドを定義する。
	mu     sync.RWMutex // 複数のゴルーチン（HTTP ハンドラは同時実行される）から触られるため sync.RWMutex で保護する。
	byID   map[string]*models.User
	byName map[string]*models.User // username 重複チェックを O(1) にするための索引
}

func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{
		byID:   make(map[string]*models.User),
		byName: make(map[string]*models.User),
	}
}

// Create は spec §17.1 の重複防止を担う。
//  1. mu.Lock() / defer mu.Unlock()
//  2. byName に u.Username が既にあれば ErrUsernameTaken を返す。
//  3. byID[u.ID] = u, byName[u.Username] = u に登録。
//  4. nil を返す。
func (r *InMemoryUserRepository) Create(u *models.User) error {
	// TODO: 実装する。
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byName[u.Username]; exists {
		return ErrUsernameTaken
	}

	r.byID[u.ID] = u
	r.byName[u.Username] = u

	return nil
}

func (r *InMemoryUserRepository) FindByUsername(username string) (*models.User, error) {
	// TODO: 実装する。
	r.mu.RLock()
	defer r.mu.RUnlock()

	name, exists := r.byName[username]
	if exists {
		return name, nil
	} else {
		return nil, ErrUserNotFound
	}
}

func (r *InMemoryUserRepository) FindByID(userid string) (*models.User, error) {
	// TODO: 実装する。
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, exists := r.byID[userid]
	if exists {
		return id, nil
	} else {
		return nil, ErrUserNotFound
	}
}

// コンパイル時にインターフェースを満たしているか検証する定番イディオム。
var _ UserRepository = (*InMemoryUserRepository)(nil)
