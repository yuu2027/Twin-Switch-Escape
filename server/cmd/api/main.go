// 起動の流れ:
//  1. config.Load() で環境変数を読む
//  2. インメモリ Repository を初期化（Phase 3 で Postgres に差し替え）
//  3. 各サービス・ハンドラを組み立てる（依存注入）
//  4. http.ServeMux にルートを登録（Go 1.22 のメソッド付きパターン）
//  5. http.Server を起動
package main

import (
	"log/slog" // ログ出力用
	"net/http" // HTTPサーバーを作る標準ライブラリ
	"os"       // OS関連の処理

	"twin-switch-escape/server/internal/auth"
	"twin-switch-escape/server/internal/config"
	"twin-switch-escape/server/internal/gameconfig"
	"twin-switch-escape/server/internal/middleware"
	"twin-switch-escape/server/internal/ranking"
	"twin-switch-escape/server/internal/repository"
)

func main() {
	// 構造化ログ（spec §3.2 / §18 Phase8 で本格化。ここでは最小導入）。
	// LOGをJSON形式で出力
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 1. 設定読込。環境変数を読む。
	cfg, err := config.Load()
	if err != nil { // 失敗したら終了
		slog.Error("failed to load config", "err", err)
		os.Exit(1) // 異常終了
	}

	// 2. リポジトリ初期化（インメモリ）。
	// Repository:データの保存・取得を担当する部品
	userRepo := repository.NewInMemoryUserRepository()
	matchRepo := repository.NewInMemoryMatchRepository()
	matchRepo.SeedDemoMatches() // ランキングが空配列にならないようデモ投入（Phase 2 限定）。

	// 3. サービス/ハンドラ組み立て（依存注入）。
	issuer := auth.NewTokenIssuer(cfg.JWTSecret, cfg.JWTExpiry)
	authSvc := auth.NewService(userRepo, matchRepo, issuer)
	authHandler := auth.NewHandler(authSvc)
	gameConfigHandler := gameconfig.NewHandler(cfg.Game)
	rankingHandler := ranking.NewHandler(ranking.NewService(matchRepo))

	// 4. ルーティング（Go 1.22+ の "METHOD /path" パターン）。
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/register", authHandler.Register)    // ユーザ登録
	mux.HandleFunc("POST /api/login", authHandler.Login)          // ログイン
	mux.HandleFunc("GET /api/game-config", gameConfigHandler.Get) // ゲーム設定
	mux.HandleFunc("GET /api/ranking", rankingHandler.Get)        // ランキング
	// /api/me は要認証 → RequireAuth でラップ。
	mux.Handle("GET /api/me", middleware.RequireAuth(issuer, http.HandlerFunc(authHandler.Me))) // JWT解析

	// 5. サーバー起動。
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
		// TODO: ReadTimeout / WriteTimeout / IdleTimeout を設定すると堅牢（学習: net/http のタイムアウト）。
	}
	slog.Info("server starting", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}

	// TODO(Phase 8): context を使った graceful shutdown（SIGINT/SIGTERM 受信→srv.Shutdown）。
}
