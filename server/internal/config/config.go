// Package config はサーバー設定を環境変数から読み込む。
// シークレット（JWT 署名鍵など）はソースに埋め込まず、必ず環境変数から取得する（spec §17.1）。
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config はサーバー全体の設定。main.go で一度だけ生成し、各層へ渡す。
type Config struct {
	Port      string        // リッスンポート（既定 "8080"）
	JWTSecret []byte        // JWT 署名鍵（必須・未設定なら起動失敗させる）
	JWTExpiry time.Duration // アクセストークン有効期限（既定 24h, spec §17.1）

	Game GameConfig // /api/game-config が返す既定値（spec §7.2）
}

// GameConfig は spec §7.2 のゲーム設定。Phase 2 では固定値（環境やステージデータ化は後続）。
//   - JSON タグを付け、gameconfig ハンドラでこの構造体をそのまま返せるようにしておくと楽。
//   - 既定値は Unity 側（GameManager の timeLimitSec=180）と一致させる。
type GameConfig struct {
	StageID             string `json:"stageId"`
	StageName           string `json:"stageName"`
	TimeLimitSec        int    `json:"timeLimitSec"`
	RequiredKeys        int    `json:"requiredKeys"`
	ReconnectTimeoutSec int    `json:"reconnectTimeoutSec"`
	ChatMaxLength       int    `json:"chatMaxLength"`
	ChatRateLimitSec    int    `json:"chatRateLimitSec"`
}

// Load は環境変数から Config を構築する。
//
//  1. os.Getenv("PORT") を読む。空なら "8080" を既定値にする。
//  2. os.Getenv("JWT_SECRET") を読む。空なら error を返す（起動失敗させる）。→ []byte に変換。
//  3. os.Getenv("JWT_EXPIRY_HOURS") を読む。空なら 24。strconv.Atoi で数値化し time.Duration に。
//  4. GameConfig は defaultGameConfig() で埋める。
//  5. 完成した *Config を返す。
func Load() (*Config, error) {
	port := os.Getenv("PORT") // 環境変数を取得
	if port == "" {
		port = "8080"
	}

	jwtSecret := os.Getenv("JWT_SECRET") // 環境変数を取得
	if jwtSecret == "" {
		return nil, errors.New("JWT_SECRET is required") // 固定メッセージだけのエラー
	}

	expiryHoursStr := os.Getenv("JWT_EXPIRY_HOURS") // 環境変数を取得
	if expiryHoursStr == "" {
		expiryHoursStr = "24"
	}

	hours, err := strconv.Atoi(expiryHoursStr) // stringをintに変換
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY_HOURS: %w", err) // 変数や元のエラーを含めたメッセージ
	}

	jwtExpiry := time.Duration(hours) * time.Hour

	gameConfig := defaultGameConfig()

	return &Config{
		Port:      port,
		JWTSecret: []byte(jwtSecret),
		JWTExpiry: jwtExpiry,
		Game:      gameConfig,
	}, nil
}

// defaultGameConfig は spec §7.2 の既定値を返す。
func defaultGameConfig() GameConfig {
	return GameConfig{
		StageID:             "stage_01",
		StageName:           "Twin Switch Lab",
		TimeLimitSec:        180,
		RequiredKeys:        2,
		ReconnectTimeoutSec: 30,
		ChatMaxLength:       100,
		ChatRateLimitSec:    1,
	}
}
