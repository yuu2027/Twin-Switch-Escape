// Package auth は認証（登録・ログイン・自分情報取得）のロジックとハンドラを持つ。
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	// パスワードハッシュは golang.org/x/crypto/bcrypt を使う（spec §17.1）。
	"golang.org/x/crypto/bcrypt"

	"twin-switch-escape/server/internal/models"
	"twin-switch-escape/server/internal/repository"
)

// 実装で使う主なシンボルのアンカー（go.mod に依存を保持させるため & 学習の手掛かり）。
// Register で bcrypt.GenerateFromPassword(pw, bcrypt.DefaultCost)、
// Login で bcrypt.CompareHashAndPassword(hash, pw) を使う。実装後はこの行は削除してよい。
var _ = bcrypt.DefaultCost

// 入力バリデーション用の定数（spec §17.3 等に倣う最小限）。
const (
	minUsernameLen = 3
	maxUsernameLen = 32 // spec §12.1 VARCHAR(32)
	minPasswordLen = 8
)

// サービス層が返すドメインエラー。ハンドラがこれを見て HTTP ステータスへ変換する。
var (
	ErrInvalidInput       = errors.New("invalid input")       // → 400
	ErrUsernameTaken      = errors.New("username taken")      // → 409
	ErrInvalidCredentials = errors.New("invalid credentials") // → 401
)

// Service は認証ロジックの中心。リポジトリと JWT 発行器に依存する。
type Service struct {
	users   repository.UserRepository
	matches repository.MatchRepository
	issuer  *TokenIssuer
}

// 受け取った依存を保持した *Service を返す。
func NewService(users repository.UserRepository, matches repository.MatchRepository, issuer *TokenIssuer) *Service {
	return &Service{
		users:   users,
		matches: matches,
		issuer:  issuer,
	}
}

// 新規ユーザーを作成する（spec §7.1, §17.1）。
//  1. username/password のバリデーション（長さ・空チェック）。NG なら ErrInvalidInput。
//  2. bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost) でハッシュ化。
//  3. models.User を組み立てる。ID は newUserID()（crypto/rand）で採番。CreatedAt/UpdatedAt=time.Now()。
//  4. users.Create(u)。repository.ErrUsernameTaken なら ErrUsernameTaken を返す。
//  5. 作成した *models.User を返す。
//
// bcrypt は golang.org/x/crypto/bcrypt。平文は保持せずハッシュのみ保存（spec §17.1）。
func (s *Service) Register(username, password string) (userID, outUsername string, err error) {
	if username == "" || password == "" {
		return "", "", ErrInvalidInput
	}

	if len(username) < minUsernameLen || len(username) > maxUsernameLen {
		return "", "", ErrInvalidInput
	}

	if len(password) < minPasswordLen {
		return "", "", ErrInvalidInput
	}

	// パスワードをハッシュ化
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}

	u := &models.User{
		ID:           newUserID(),
		Username:     username,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.users.Create(u); err != nil {
		if errors.Is(err, repository.ErrUsernameTaken) {
			return "", "", ErrUsernameTaken
		}
		return "", "", err
	}

	return u.ID, u.Username, nil
}

// 認証してアクセストークンを発行する
//  1. users.FindByUsername(username)。見つからなければ ErrInvalidCredentials
//     （「ユーザーが無い」と「パスワード違い」を区別しない＝列挙攻撃対策）。
//  2. bcrypt.CompareHashAndPassword(hash, []byte(password))。不一致なら ErrInvalidCredentials。
//  3. issuer.Issue(user.ID) でトークン発行。
//  4. (accessToken, userID, username, nil) を返す。
func (s *Service) Login(username, password string) (accessToken, userID, outUsername string, err error) {
	u, err := s.users.FindByUsername(username)
	if err != nil {
		return "", "", "", ErrInvalidCredentials
	}

	// 保存しているハッシュ化されたものと入力されたパスワードが一致するか判定
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", "", "", ErrInvalidCredentials
	}

	accessToken, err = s.issuer.Issue(u.ID) // トークンの発行
	if err != nil {
		return "", "", "", err
	}

	return accessToken, u.ID, u.Username, nil
}

// 自分の情報を返す
// bestClearTime / clearCount は専用カラムでなく matches からの集計で導出（spec §12.1）。
//  1. users.FindByID(userID)。無ければ ErrInvalidCredentials 相当（実運用では起きにくい）。
//  2. matches.Stats(userID) で bestClearTimeMs, clearCount を取得。
//  3. bestClearTimeMs はミリ秒→秒(小数)へ変換
func (s *Service) Me(userID string) (username string, bestClearTimeSec float64, clearCount int, err error) {
	u, err := s.users.FindByID(userID)
	if err != nil {
		return "", 0.0, 0, ErrInvalidCredentials
	}

	bestClearTimeMs, clearCount, err := s.matches.Stats(u.ID)
	if err != nil {
		return "", 0.0, 0, err
	}

	bestClearTimeSec = float64(bestClearTimeMs) / 1000.0

	if clearCount == 0 {
		return u.Username, 0.0, 0, nil
	}

	return u.Username, bestClearTimeSec, clearCount, nil
}

// "user_xxxx" 形式の ID を生成する。
//   - crypto/rand で 8〜16 バイト読み、encoding/hex か base64(URLエンコード) で文字列化。
//   - "user_" を prefix に付けて返す。
//
// ID やトークンの乱数は math/rand ではなく crypto/rand を使う（予測困難にする）。
func newUserID() string {
	b := make([]byte, 16)

	if _, err := rand.Read(b); err != nil { // 乱数生成
		panic(err)
	}

	return "user_" + hex.EncodeToString(b) // 文字列に変換
}
