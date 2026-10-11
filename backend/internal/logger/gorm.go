package logger

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// SlowQueryThreshold を超えた SQL は WARN で記録する
const SlowQueryThreshold = 200 * time.Millisecond

// gormLogger は GORM のログを zerolog（JSON）に流すアダプター。
// リクエストのコンテキストで実行された SQL のログには request_id が付く。
type gormLogger struct {
	// logSQL が true のときだけ SQL 文を出力する（SQL にはユーザーデータが含まれるため本番では出さない）
	logSQL bool
}

// NewGormLogger は GORM 用のロガーを返す。
// logSQL=true のとき全 SQL を DEBUG で出力する。SQL エラー・遅い SQL は常に WARN で記録する。
func NewGormLogger(logSQL bool) gormlogger.Interface {
	return &gormLogger{logSQL: logSQL}
}

func (l *gormLogger) LogMode(gormlogger.LogLevel) gormlogger.Interface { return l }

func (l *gormLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	FromContext(ctx).Info().Msgf(msg, args...)
}

func (l *gormLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	FromContext(ctx).Warn().Msgf(msg, args...)
}

func (l *gormLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	FromContext(ctx).Error().Msgf(msg, args...)
}

// Trace は SQL 1 本ごとに呼ばれる。
// SQL エラーは呼び出し元（ハンドラー）が必要に応じて ERROR で記録するため、ここでは WARN にとどめる
// （重複キーなど業務上想定内のエラーも含まれるため）。レコードなしはエラー扱いしない。
func (l *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	log := FromContext(ctx)

	var event *zerolog.Event
	switch {
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound):
		event = log.Warn().Err(err)
	case elapsed > SlowQueryThreshold:
		event = log.Warn().Bool("slow", true)
	case l.logSQL:
		event = log.Debug()
	default:
		return
	}
	if !event.Enabled() {
		return
	}

	sql, rows := fc()
	if l.logSQL {
		event = event.Str("sql", sql)
	}
	event.Int64("rows", rows).
		Float64("elapsed_ms", float64(elapsed.Microseconds())/1000).
		Msg("sql")
}
