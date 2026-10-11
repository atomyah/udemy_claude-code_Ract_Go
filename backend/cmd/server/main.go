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
	notificationSvc := service.NewNotificationService(notificationRepo)
	storageSvc := service.NewStorageService(cfg, fbApp)
	userSvc := service.NewUserService(userRepo, followRepo, notificationSvc)
	postSvc := service.NewPostService(postRepo, mediaRepo, hashtagRepo, userRepo, likeRepo, bookmarkRepo, storageSvc, notificationSvc)
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
	// c.RealIP()（レートリミット・アクセスログ）で使うクライアントIPの取得方法。
	// Echo 既定の X-Forwarded-For 先頭は偽装できるため、信頼するプロキシ段数で右端から取る
	e.IPExtractor = appMiddleware.ClientIPExtractor(cfg.TrustedProxyHops)

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	if cfg.ShouldCountQueries() {
		e.Use(appMiddleware.QueryCount())
	}
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     cfg.AllowedOrigins(),
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
		AllowCredentials: true,
		// フロントエンドがレート制限の状態を読めるようにする
		ExposeHeaders: []string{
			appMiddleware.HeaderRateLimitLimit,
			appMiddleware.HeaderRateLimitRemaining,
			appMiddleware.HeaderRateLimitReset,
			echo.HeaderRetryAfter,
		},
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

	// レートリミット（インメモリ・fixed window。Cloud Run の max instances = 1 構成が前提）
	// 認証系: IP 単位で API ごとに 5 回/分。一般: 認証済みはユーザーID・未認証は IP 単位で API ごとに 60 回/分
	authRateLimit := appMiddleware.AuthRateLimit(cfg.AuthRateLimit)
	generalRateLimit := appMiddleware.GeneralRateLimit(cfg.GeneralRateLimit)

	// --- 認証（認証不要） ---
	// ログイン・登録（Google ログインを含む）は認証系のレート制限。パスワードリセットAPIを追加する場合も authRateLimit を付ける
	auth := v1.Group("/auth")
	auth.POST("/register", authHandler.Register, authRateLimit)
	auth.POST("/login", authHandler.Login, authRateLimit)
	auth.POST("/google", authHandler.GoogleLogin, authRateLimit)
	auth.POST("/logout", authHandler.Logout, generalRateLimit)
	auth.POST("/refresh", authHandler.Refresh, generalRateLimit)

	// 認証系以外のAPI。トークンがあれば先に userID を確定させ、ユーザーID単位でレート制限する。
	// 無効なトークンのリクエストも IP 単位で制限されるよう、必須認証（jwtMiddleware）はレート制限の後に置く
	api := v1.Group("", optionalAuthMiddleware, generalRateLimit)

	// --- ユーザー（一部認証必要） ---
	api.GET("/users/me", userHandler.GetMe, jwtMiddleware)
	api.PUT("/users/me", userHandler.UpdateProfile, jwtMiddleware)
	api.PUT("/users/me/avatar", userHandler.UpdateAvatar, jwtMiddleware)
	api.PUT("/users/me/banner", userHandler.UpdateBanner, jwtMiddleware)
	api.PUT("/users/me/theme", userHandler.UpdateTheme, jwtMiddleware)
	api.PUT("/users/me/email", userHandler.ChangeEmail, jwtMiddleware)
	api.PUT("/users/me/password", userHandler.ChangePassword, jwtMiddleware)
	api.GET("/users/:handle", userHandler.GetProfile)
	api.GET("/users/:handle/posts", userHandler.GetUserPosts)
	api.GET("/users/:handle/replies", userHandler.GetUserReplies)
	api.GET("/users/:handle/followers", userHandler.GetFollowers)
	api.GET("/users/:handle/following", userHandler.GetFollowing)
	api.POST("/users/:handle/follow", userHandler.Follow, jwtMiddleware)
	api.DELETE("/users/:handle/follow", userHandler.Unfollow, jwtMiddleware)

	// --- 投稿 ---
	api.GET("/posts", postHandler.GetExplore)
	api.GET("/posts/home", postHandler.GetHome, jwtMiddleware)
	api.POST("/posts", postHandler.CreatePost, jwtMiddleware)
	api.GET("/posts/:id", postHandler.GetPost)
	api.PUT("/posts/:id", postHandler.UpdatePost, jwtMiddleware)
	api.DELETE("/posts/:id", postHandler.DeletePost, jwtMiddleware)
	api.GET("/posts/:id/comments", postHandler.GetComments)
	api.POST("/posts/:id/comments", postHandler.CreateComment, jwtMiddleware)

	// --- インタラクション ---
	api.POST("/posts/:id/like", interactionHandler.Like, jwtMiddleware)
	api.DELETE("/posts/:id/like", interactionHandler.Unlike, jwtMiddleware)
	api.POST("/posts/:id/repost", interactionHandler.Repost, jwtMiddleware)
	api.DELETE("/posts/:id/repost", interactionHandler.Unrepost, jwtMiddleware)
	api.POST("/posts/:id/bookmark", interactionHandler.Bookmark, jwtMiddleware)
	api.DELETE("/posts/:id/bookmark", interactionHandler.Unbookmark, jwtMiddleware)
	api.GET("/bookmarks", interactionHandler.GetBookmarks, jwtMiddleware)

	// --- 検索 ---
	api.GET("/search/users", searchHandler.SearchUsers)
	api.GET("/search/posts", searchHandler.SearchPosts)
	api.GET("/search/hashtags/:tag", searchHandler.GetHashtagPosts)

	// --- 通知 ---
	api.GET("/notifications", notificationHandler.GetNotifications, jwtMiddleware)
	api.PUT("/notifications/read", notificationHandler.MarkAllRead, jwtMiddleware)

	// --- 管理者 ---
	admin := api.Group("/admin", jwtMiddleware, adminMiddleware)
	admin.DELETE("/posts/:id", adminHandler.AdminDeletePost)
	admin.PUT("/users/:id/suspend", adminHandler.SuspendUser)
	admin.DELETE("/users/:id/suspend", adminHandler.UnsuspendUser)
	admin.GET("/users", adminHandler.ListUsers)

	addr := fmt.Sprintf(":%s", cfg.Port)
	e.Logger.Fatal(e.Start(addr))
}
