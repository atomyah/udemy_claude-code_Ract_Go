// Package logger はアプリケーション全体の構造化ログ（zerolog・JSON 形式）を提供する。
//
//   - すべてのログは 1 行 1 JSON で標準出力に書き出す（Cloud Run / Docker のログ収集がそのまま扱える形式）。
//   - リクエスト処理中のログは FromContext(ctx) で取得したロガーで出す。RequestID ミドルウェアが
//     コンテキストに request_id 付きのロガーを入れるため、ログに自動で request_id が含まれる。
//   - レベルの使い分け: DEBUG=SQL など開発時の詳細 / INFO=アクセスログ・起動などの通常イベント /
//     WARN=4xx・遅いリクエスト・設定値の誤りなど想定内の異常 / ERROR=5xx・panic など要調査の障害（スタックトレース付き）。
package logger

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	// Init が呼ばれる前（設定の読み込み中など）のログも JSON で出す
	setup(os.Stdout, zerolog.InfoLevel)
}

// Init は環境（ENV）とログレベル（LOG_LEVEL）に応じてグローバルロガーを設定する。
// level が空のときは開発環境なら DEBUG、それ以外は INFO にする。
func Init(env, level string) {
	setup(os.Stdout, ParseLevel(env, level))
}

// ParseLevel は LOG_LEVEL の文字列（debug / info / warn / error）をレベルに変換する。
// 空・不正な値のときは環境ごとの既定値（開発: DEBUG、テスト・本番: INFO）を返す。
func ParseLevel(env, level string) zerolog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	}
	if env == "development" {
		return zerolog.DebugLevel
	}
	return zerolog.InfoLevel
}

// SetOutput はグローバルロガーの出力先とレベルを差し替える（テストでログ内容を検証するときに使う）
func SetOutput(w io.Writer, level zerolog.Level) {
	setup(w, level)
}

func setup(w io.Writer, level zerolog.Level) {
	l := zerolog.New(w).Level(level).With().Timestamp().Logger()
	log.Logger = l
	// コンテキストにロガーが無いとき zerolog.Ctx はこのロガーを返す
	zerolog.DefaultContextLogger = &log.Logger
}

// L はグローバルロガーを返す。リクエストに紐づかない処理（起動・設定読み込み・バッチなど）で使う
func L() *zerolog.Logger {
	return &log.Logger
}

// FromContext はコンテキストに紐づいたロガー（request_id 付き）を返す。無ければグローバルロガーを返す
func FromContext(ctx context.Context) *zerolog.Logger {
	return zerolog.Ctx(ctx)
}

type requestIDKey struct{}

// WithRequestID はリクエストIDと、そのIDを必ず出力するロガーを埋め込んだコンテキストを返す
func WithRequestID(ctx context.Context, requestID string) context.Context {
	l := FromContext(ctx).With().Str("request_id", requestID).Logger()
	ctx = context.WithValue(ctx, requestIDKey{}, requestID)
	return l.WithContext(ctx)
}

// RequestID はコンテキストのリクエストIDを返す。無ければ空文字
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}
