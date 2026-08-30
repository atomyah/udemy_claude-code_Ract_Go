// @title           SNS API
// @version         1.0
// @description     Twitter ライクな SNS アプリの REST API。テキスト・画像・動画を投稿し、いいね・コメント・リポスト・ブックマークでインタラクションできる。
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  atyahara@gmail.com

// @license.name  MIT

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
// @description                JWT Access Token を入力してください。形式: "Bearer {token}"

// @tag.name         auth
// @tag.description  ユーザー認証（登録・ログイン・トークン管理）

// @tag.name         users
// @tag.description  ユーザープロフィール・フォロー機能

// @tag.name         posts
// @tag.description  投稿のCRUDおよびコメント機能

// @tag.name         interactions
// @tag.description  いいね・リポスト・ブックマーク機能

// @tag.name         search
// @tag.description  ユーザー・投稿・ハッシュタグ検索

// @tag.name         notifications
// @tag.description  アプリ内通知

// @tag.name         admin
// @tag.description  管理者専用機能（adminロールが必要）
package main

import (
	"fmt"
	"log"
	"net/http"

	_ "github.com/atyahara/sns-backend/docs"
	"github.com/atyahara/sns-backend/internal/config"
	"github.com/atyahara/sns-backend/internal/handler"
	appMiddleware "github.com/atyahara/sns-backend/internal/middleware"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/atyahara/sns-backend/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
)

func main() {
	cfg := config.Load()
	db := config.NewDB(cfg)

	// ============================================================
	// 依存性注入
	// ============================================================
	userRepo := repository.NewUserRepository(db)
	followRepo := repository.NewFollowRepository(db)
	postRepo := repository.NewPostRepository(db)
	mediaRepo := repository.NewMediaRepository(db)
	hashtagRepo := repository.NewHashtagRepository(db)
	likeRepo := repository.NewLikeRepository(db)
	bookmarkRepo := repository.NewBookmarkRepository(db)
	notificationRepo := repository.NewNotificationRepository(db)
	searchRepo := repository.NewSearchRepository(db)

	// Firebaseは開発・本番では必須。テスト環境（ENV=test）では認証情報がなくても
	// サーバーを起動できるようにする（E2Eテストはメール＋パスワード認証のみ使うため）
	fbApp, err := config.NewFirebaseApp(cfg)
	if err != nil {
		if !cfg.IsTest() {
			log.Fatalf("firebase app init failed: %v", err)
		}
		log.Printf("firebase app init skipped (ENV=test): %v", err)
		fbApp = nil
	}
	authSvc := service.NewAuthService(cfg, userRepo, fbApp)
	storageSvc := service.NewStorageService(cfg, fbApp)
	userSvc := service.NewUserService(userRepo, followRepo)
	postSvc := service.NewPostService(postRepo, mediaRepo, hashtagRepo, userRepo, likeRepo, bookmarkRepo, storageSvc)
	notificationSvc := service.NewNotificationService(notificationRepo)
	likeSvc := service.NewLikeService(postRepo, likeRepo, notificationSvc)
	repostSvc := service.NewRepostService(postRepo, postSvc, notificationSvc)
	bookmarkSvc := service.NewBookmarkService(postRepo, bookmarkRepo, postSvc)
	commentSvc := service.NewCommentService(postRepo, postSvc, notificationSvc)
	searchSvc := service.NewSearchService(searchRepo, hashtagRepo, postRepo, followRepo, postSvc)
	adminSvc := service.NewAdminService(postRepo, userRepo)

	authHandler := handler.NewAuthHandler(authSvc)
	userHandler := handler.NewUserHandler(userRepo, storageSvc, userSvc, postSvc)
	postHandler := handler.NewPostHandler(postSvc, commentSvc)
	interactionHandler := handler.NewInteractionHandler(likeSvc, repostSvc, bookmarkSvc)
	searchHandler := handler.NewSearchHandler(searchSvc)
	notificationHandler := handler.NewNotificationHandler(notificationSvc)
	adminHandler := handler.NewAdminHandler(adminSvc)

	// ============================================================
	// Echo セットアップ
	// ============================================================
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     cfg.AllowedOrigins(),
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
		AllowCredentials: true,
	}))

	if cfg.IsDevelopment() {
		e.GET("/swagger/*", echoSwagger.WrapHandler)
	}

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// ============================================================
	// ルーティング
	// ============================================================
	v1 := e.Group("/api/v1")

	jwtMiddleware := appMiddleware.JWTAuth(cfg.JWTSecret)
	optionalAuthMiddleware := appMiddleware.OptionalJWTAuth(cfg.JWTSecret)
	adminMiddleware := appMiddleware.AdminOnly()

	// --- 認証（認証不要） ---
	auth := v1.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/logout", authHandler.Logout)
	auth.POST("/refresh", authHandler.Refresh)
	auth.POST("/google", authHandler.GoogleLogin)

	// --- ユーザー（一部認証必要） ---
	v1.GET("/users/me", userHandler.GetMe, jwtMiddleware)
	v1.PUT("/users/me", userHandler.UpdateProfile, jwtMiddleware)
	v1.PUT("/users/me/avatar", userHandler.UpdateAvatar, jwtMiddleware)
	v1.PUT("/users/me/banner", userHandler.UpdateBanner, jwtMiddleware)
	v1.PUT("/users/me/theme", userHandler.UpdateTheme, jwtMiddleware)
	v1.PUT("/users/me/email", userHandler.ChangeEmail, jwtMiddleware)
	v1.PUT("/users/me/password", userHandler.ChangePassword, jwtMiddleware)
	v1.GET("/users/:handle", userHandler.GetProfile, optionalAuthMiddleware)
	v1.GET("/users/:handle/posts", userHandler.GetUserPosts, optionalAuthMiddleware)
	v1.GET("/users/:handle/replies", userHandler.GetUserReplies, optionalAuthMiddleware)
	v1.GET("/users/:handle/followers", userHandler.GetFollowers, optionalAuthMiddleware)
	v1.GET("/users/:handle/following", userHandler.GetFollowing, optionalAuthMiddleware)
	v1.POST("/users/:handle/follow", userHandler.Follow, jwtMiddleware)
	v1.DELETE("/users/:handle/follow", userHandler.Unfollow, jwtMiddleware)

	// --- 投稿 ---
	v1.GET("/posts", postHandler.GetExplore, optionalAuthMiddleware)
	v1.GET("/posts/home", postHandler.GetHome, jwtMiddleware)
	v1.POST("/posts", postHandler.CreatePost, jwtMiddleware)
	v1.GET("/posts/:id", postHandler.GetPost, optionalAuthMiddleware)
	v1.PUT("/posts/:id", postHandler.UpdatePost, jwtMiddleware)
	v1.DELETE("/posts/:id", postHandler.DeletePost, jwtMiddleware)
	v1.GET("/posts/:id/comments", postHandler.GetComments, optionalAuthMiddleware)
	v1.POST("/posts/:id/comments", postHandler.CreateComment, jwtMiddleware)

	// --- インタラクション ---
	v1.POST("/posts/:id/like", interactionHandler.Like, jwtMiddleware)
	v1.DELETE("/posts/:id/like", interactionHandler.Unlike, jwtMiddleware)
	v1.POST("/posts/:id/repost", interactionHandler.Repost, jwtMiddleware)
	v1.DELETE("/posts/:id/repost", interactionHandler.Unrepost, jwtMiddleware)
	v1.POST("/posts/:id/bookmark", interactionHandler.Bookmark, jwtMiddleware)
	v1.DELETE("/posts/:id/bookmark", interactionHandler.Unbookmark, jwtMiddleware)
	v1.GET("/bookmarks", interactionHandler.GetBookmarks, jwtMiddleware)

	// --- 検索 ---
	v1.GET("/search/users", searchHandler.SearchUsers, optionalAuthMiddleware)
	v1.GET("/search/posts", searchHandler.SearchPosts, optionalAuthMiddleware)
	v1.GET("/search/hashtags/:tag", searchHandler.GetHashtagPosts, optionalAuthMiddleware)

	// --- 通知 ---
	v1.GET("/notifications", notificationHandler.GetNotifications, jwtMiddleware)
	v1.PUT("/notifications/read", notificationHandler.MarkAllRead, jwtMiddleware)

	// --- 管理者 ---
	admin := v1.Group("/admin", jwtMiddleware, adminMiddleware)
	admin.DELETE("/posts/:id", adminHandler.AdminDeletePost)
	admin.PUT("/users/:id/suspend", adminHandler.SuspendUser)
	admin.DELETE("/users/:id/suspend", adminHandler.UnsuspendUser)
	admin.GET("/users", adminHandler.ListUsers)

	addr := fmt.Sprintf(":%s", cfg.Port)
	e.Logger.Fatal(e.Start(addr))
}
