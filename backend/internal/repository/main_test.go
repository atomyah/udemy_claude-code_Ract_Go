package repository

import (
	"context"
	"log"
	"os"
	"sync"
	"testing"

	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/querycount"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testDB はテスト用PostgreSQL（db_test）への接続。DATABASE_URLが無い場合はnilのまま
var testDB *gorm.DB

// TestMain はテスト用DBに接続し、スキーマを適用する。
// api_testコンテナのエントリーポイントでもマイグレーションが走るが、
// ホストから直接実行された場合にも動くようここでもAutoMigrateしておく。
func TestMain(m *testing.M) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Println("DATABASE_URL が未設定のためリポジトリ統合テストをスキップします")
		os.Exit(m.Run())
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		log.Fatalf("テストDB接続失敗: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.Post{}, &model.Media{}, &model.Like{},
		&model.Bookmark{}, &model.Follow{}, &model.Hashtag{}, &model.PostHashtag{}, &model.Notification{}, &model.RefreshToken{},
	); err != nil {
		log.Fatalf("テストDBマイグレーション失敗: %v", err)
	}
	// N+1検出テストでクエリ本数を数えられるようにする（カウンター付きコンテキストのクエリのみ数える）
	if err := querycount.Register(db); err != nil {
		log.Fatalf("クエリ数カウンターの登録失敗: %v", err)
	}
	testDB = db

	os.Exit(m.Run())
}

// setupDB はテストごとに全テーブルを空にしたDBハンドルを返す。
// DATABASE_URLが無い環境ではテストをスキップする。
//
// 注意: 全テーブルをTRUNCATEするため、このパッケージのテストは直列実行が前提。
// t.Parallel() を呼ぶと他のテストのデータを消してしまうので、
// 誤って並列化された場合はここで気づけるよう明示的に失敗させる。
func setupDB(t *testing.T) *gorm.DB {
	t.Helper()

	if testDB == nil {
		t.Skip("DATABASE_URL が未設定のためスキップ（make test-backend で実行してください）")
	}
	assertNotParallel(t)

	err := testDB.Exec(`TRUNCATE refresh_tokens, notifications, post_hashtags, hashtags, bookmarks, likes, follows, media, posts, users RESTART IDENTITY CASCADE`).Error
	if err != nil {
		t.Fatalf("テストデータのクリア失敗: %v", err)
	}
	return testDB
}

var (
	activeMu   sync.Mutex
	activeTest string
)

// assertNotParallel は同時に2つ以上のテストがテストDBを使っていないことを確認する。
// TRUNCATEで全テーブルを消すため、並列実行されると他方のテストデータが消えて
// 原因の分かりにくい失敗になる。それを即座に検出する。
func assertNotParallel(t *testing.T) {
	t.Helper()

	activeMu.Lock()
	if activeTest != "" && activeTest != t.Name() {
		running := activeTest
		activeMu.Unlock()
		t.Fatalf("リポジトリテストは直列実行が前提です（%s と並列実行されました）。t.Parallel() を外してください", running)
	}
	activeTest = t.Name()
	activeMu.Unlock()

	t.Cleanup(func() {
		activeMu.Lock()
		activeTest = ""
		activeMu.Unlock()
	})
}

// createUser はテスト用ユーザーを作成する
func createUser(t *testing.T, db *gorm.DB, handle string) *model.User {
	t.Helper()

	user := &model.User{
		Email:       handle + "@example.com",
		Handle:      handle,
		DisplayName: handle + "さん",
		Theme:       "light",
		Role:        "user",
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("テストユーザー作成失敗: %v", err)
	}
	return user
}

// createPost はテスト用投稿を作成する
func createPost(t *testing.T, db *gorm.DB, userID uuid.UUID, content string) *model.Post {
	t.Helper()

	post := &model.Post{UserID: userID, Content: content}
	if err := db.Create(post).Error; err != nil {
		t.Fatalf("テスト投稿作成失敗: %v", err)
	}
	return post
}

// createReply は指定投稿への返信を作成する
func createReply(t *testing.T, db *gorm.DB, userID, parentID uuid.UUID, content string) *model.Post {
	t.Helper()

	post := &model.Post{UserID: userID, Content: content, ReplyTo: &parentID}
	if err := db.Create(post).Error; err != nil {
		t.Fatalf("テスト返信作成失敗: %v", err)
	}
	return post
}

func testCtx() context.Context {
	return context.Background()
}
