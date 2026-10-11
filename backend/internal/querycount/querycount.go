// Package querycount はリクエスト単位で発行されたSQLクエリ数を数える。
// N+1クエリの検出・性能計測のための開発／テスト環境専用の仕組み。
package querycount

import (
	"context"
	"sync/atomic"

	"gorm.io/gorm"
)

type counterKey struct{}

// WithCounter はクエリ数を数えるカウンターを埋め込んだコンテキストを返す
func WithCounter(ctx context.Context) context.Context {
	return context.WithValue(ctx, counterKey{}, new(atomic.Int64))
}

// Count はコンテキストのカウンターが数えたクエリ数を返す。カウンターが無ければ0
func Count(ctx context.Context) int64 {
	if c, ok := ctx.Value(counterKey{}).(*atomic.Int64); ok {
		return c.Load()
	}
	return 0
}

// Register はGORMの全種別（SELECT / INSERT / UPDATE / DELETE / Row / Raw）のコールバックに
// カウント処理を登録する。Preloadで発行されるクエリもSELECTとして1本ずつ数えられる。
func Register(db *gorm.DB) error {
	cb := db.Callback()
	registrations := []struct {
		name     string
		register func(string, func(*gorm.DB)) error
	}{
		{"querycount:query", cb.Query().After("gorm:query").Register},
		{"querycount:create", cb.Create().After("gorm:create").Register},
		{"querycount:update", cb.Update().After("gorm:update").Register},
		{"querycount:delete", cb.Delete().After("gorm:delete").Register},
		{"querycount:row", cb.Row().After("gorm:row").Register},
		{"querycount:raw", cb.Raw().After("gorm:raw").Register},
	}
	for _, r := range registrations {
		if err := r.register(r.name, increment); err != nil {
			return err
		}
	}
	return nil
}

func increment(tx *gorm.DB) {
	if tx.Statement == nil || tx.Statement.Context == nil {
		return
	}
	if c, ok := tx.Statement.Context.Value(counterKey{}).(*atomic.Int64); ok {
		c.Add(1)
	}
}
