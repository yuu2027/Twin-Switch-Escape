package auth

import (
	"fmt"
	"time"

	// 学習ポイント: JWT は標準ライブラリに無いので golang-jwt/jwt/v5 を使う。
	// 署名アルゴリズムは HS256（共有鍵）で十分（spec §17.1）。
	"github.com/golang-jwt/jwt/v5"
)

// 実装で使う主なシンボルのアンカー（go.mod に依存を保持させるため & 学習の手掛かり）。
// 実装時にはこれらを Issue / Parse の中で使う:
//   - jwt.NewWithClaims, jwt.SigningMethodHS256, jwt.RegisteredClaims, jwt.NewNumericDate
//   - jwt.ParseWithClaims
//
// 実装後はこの行は削除してよい。
var _ = jwt.SigningMethodHS256

// TokenIssuer は JWT の発行・検証を担う。
// 署名鍵と有効期限は config から渡す（spec §17.1: シークレットは環境変数経由）。
type TokenIssuer struct {
	secret []byte
	expiry time.Duration
}

func NewTokenIssuer(secret []byte, expiry time.Duration) *TokenIssuer {
	return &TokenIssuer{
		secret: secret,
		expiry: expiry,
	}
}

// Issue は userID を sub に持つアクセストークンを発行する。
//
// TODO:
//  1. jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{...}) を作る。
//     RegisteredClaims に Subject=userID, ExpiresAt=jwt.NewNumericDate(time.Now().Add(t.expiry)),
//     IssuedAt も入れておくとよい。
//  2. token.SignedString(t.secret) で署名済み文字列を得る。
//  3. (string, error) を返す。
func (t *TokenIssuer) Issue(userID string) (string, error) {
	// TODO: 実装する。
	claims := jwt.RegisteredClaims{ // JWTでよく使われる標準項目
		Subject:   userID,                                       // トークンの対象者
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(t.expiry)), // 有効期限
		IssuedAt:  jwt.NewNumericDate(time.Now()),               // 発行時刻
	}

	// 指定した署名方式とクレームを使ってJWTオブジェクトを作る
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString(t.secret) // 署名済みの文字列に変換
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

// Parse はトークン文字列を検証し、userID(sub) を取り出す。
//
//  1. jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, keyFunc) を呼ぶ。
//     keyFunc 内で署名方式が HS256 か確認し、t.secret を返す（アルゴリズム混同攻撃を防ぐ）。
//  2. 検証失敗（期限切れ・改ざん）なら error を返す。
//  3. claims.Subject を userID として返す。
func (t *TokenIssuer) Parse(tokenStr string) (userID string, err error) {
	claims := &jwt.RegisteredClaims{}

	// JWT文字列を解析・検証
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 { // 署名方式が一致している判定
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return t.secret, nil
	})
	if err != nil {
		return "", err
	}

	if !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	if claims.Subject == "" {
		return "", fmt.Errorf("subject is empty")
	}

	return claims.Subject, nil
}
