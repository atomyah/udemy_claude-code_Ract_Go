package querycount

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCount_WithoutCounterReturnsZero(t *testing.T) {
	assert.Equal(t, int64(0), Count(context.Background()))
}

func TestCount_NewCounterStartsAtZero(t *testing.T) {
	assert.Equal(t, int64(0), Count(WithCounter(context.Background())))
}

// TestRegister_CountsQueriesPerContext はテスト用DBに実際にクエリを発行し、
// カウンター付きコンテキストのクエリだけが数えられることを確認する
func TestRegister_CountsQueriesPerContext(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL が未設定のためスキップ（make test-backend で実行してください）")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, Register(db))

	ctx := WithCounter(context.Background())
	var n int
	require.NoError(t, db.WithContext(ctx).Raw("SELECT 1").Scan(&n).Error)
	require.NoError(t, db.WithContext(ctx).Exec("SELECT 1").Error)
	var count int64
	require.NoError(t, db.WithContext(ctx).Table("users").Count(&count).Error)
	assert.Equal(t, int64(3), Count(ctx))

	// カウンターの無いコンテキストのクエリは数えない（別リクエストに混ざらない）
	other := WithCounter(context.Background())
	require.NoError(t, db.WithContext(context.Background()).Exec("SELECT 1").Error)
	assert.Equal(t, int64(0), Count(other))
	assert.Equal(t, int64(3), Count(ctx))
}
