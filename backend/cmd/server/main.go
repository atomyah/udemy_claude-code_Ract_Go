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
// @description                ブラウザは HttpOnly Cookie（access_token）で自動送信される。ブラウザ以外のクライアントは "Bearer {token}" 形式でも送れる

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
	"errors"
	"fmt"
	"net/http"

	"github.com/atyahara/sns-backend/docs"
	"github.com/atyahara/sns-backend/internal/adminui"
	"github.com/atyahara/sns-backend/internal/config"
	"github.com/atyahara/sns-backend/internal/handler"
	"github.com/atyahara/sns-backend/internal/httperr"
	"github.com/atyahara/sns-backend/internal/logger"
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
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	adminRepo := repository.NewAdminRepository(db)
	adminUserRepo := repository.NewAdminUserRepository(db)
	passwordResetRepo := repository.NewPasswordResetRepository(db)
	dashboardRepo := repository.NewDashboardRepository(db)

	// Firebaseは開発・本番では必須。テスト環境（ENV=test）では認証情報がなくても
	// サーバーを起動できるようにする（E2Eテストはメール＋パスワード認証のみ使うため）
	fbApp, err := config.NewFirebaseApp(cfg)
	if err != nil {
		if !cfg.IsTest() {
			logger.L().Fatal().Err(err).Msg("firebase app init failed")
		}
		logger.L().Warn().Err(err).Msg("firebase app init skipped (ENV=test)")
		fbApp = nil
	}
	authSvc := service.NewAuthService(cfg, userRepo, refreshTokenRepo, fbApp)
	notificationSvc := service.NewNotificationService(notificationRepo)
	storageSvc := service.NewStorageService(cfg, fbApp)
	userSvc := service.NewUserService(userRepo, followRepo, notificationSvc)
	postSvc := service.NewPostService(postRepo, mediaRepo, hashtagRepo, userRepo, likeRepo, bookmarkRepo, storageSvc, notificationSvc)
	likeSvc := service.NewLikeService(postRepo, likeRepo, notificationSvc)
	repostSvc := service.NewRepostService(postRepo, postSvc, notificationSvc)
	bookmarkSvc := service.NewBookmarkService(postRepo, bookmarkRepo, postSvc)
	commentSvc := service.NewCommentService(postRepo, postSvc, notificationSvc)
	searchSvc := service.NewSearchService(searchRepo, hashtagRepo, postRepo, followRepo, postSvc)
	adminSvc := service.NewAdminService(postRepo, userRepo, refreshTokenRepo)
	passwordResetSvc := service.NewPasswordResetService(cfg, userRepo, passwordResetRepo)

	authHandler := handler.NewAuthHandler(authSvc, handler.AuthCookieConfig{
		Secure:   cfg.CookieSecure,
		SameSite: cfg.CookieSameSite(),
	})
	userHandler := handler.NewUserHandler(userRepo, storageSvc, userSvc, postSvc)
	postHandler := handler.NewPostHandler(postSvc, commentSvc)
	interactionHandler := handler.NewInteractionHandler(likeSvc, repostSvc, bookmarkSvc)
	searchHandler := handler.NewSearchHandler(searchSvc)
	notificationHandler := handler.NewNotificationHandler(notificationSvc)
	adminHandler := handler.NewAdminHandler(adminSvc)
	passwordResetHandler := handler.NewPasswordResetHandler(passwordResetSvc)

	// ============================================================
	// Echo セットアップ
	// ============================================================
	e := echo.New()
	e.HideBanner = true
	// 起動メッセージはテキストで出るため隠し、構造化ログ（"server started"）で出す
	e.HidePort = true
	// c.RealIP()（レートリミット・アクセスログ）で使うクライアントIPの取得方法。
	// Echo 既定の X-Forwarded-For 先頭は偽装できるため、信頼するプロキシ段数で右端から取る
	e.IPExtractor = appMiddleware.ClientIPExtractor(cfg.TrustedProxyHops)
	// ハンドラーのエラー・panic・Echo 標準エラー（404 / 405 など）を { code, message } の日本語にそろえ、
	// 5xx は原因をスタックトレース付きで ERROR ログに残してリクエストIDだけを返す
	e.HTTPErrorHandler = httperr.Handler

	// 順序: リクエストID → アクセスログ → panic 回復（panic も 500 としてアクセスログに残す）
	e.Use(appMiddleware.RequestID())
	e.Use(appMiddleware.AccessLog(accessLogActions()))
	e.Use(appMiddleware.Recover())
	if cfg.ShouldCountQueries() {
		e.Use(appMiddleware.QueryCount())
	}
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     cfg.AllowedOrigins(),
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
		AllowCredentials: true,
		// フロントエンドがレート制限の状態とリクエストIDを読めるようにする
		ExposeHeaders: []string{
			echo.HeaderXRequestID,
			appMiddleware.HeaderRateLimitLimit,
			appMiddleware.HeaderRateLimitRemaining,
			appMiddleware.HeaderRateLimitReset,
			echo.HeaderRetryAfter,
		},
	}))
	// 認証 Cookie は SameSite=None のため、状態を変更するリクエストは Origin を検証して CSRF を防ぐ
	e.Use(appMiddleware.OriginCheck(cfg.AllowedOrigins()))

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

	jwtMiddleware := appMiddleware.JWTAuth(cfg.JWTSecret, userRepo)
	optionalAuthMiddleware := appMiddleware.OptionalJWTAuth(cfg.JWTSecret, userRepo)
	adminMiddleware := appMiddleware.AdminOnly(userRepo)

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
	auth.POST("/password-reset-requests", passwordResetHandler.RequestReset, authRateLimit)
	auth.POST("/password-reset", passwordResetHandler.ResetPassword, authRateLimit)
	auth.POST("/logout", authHandler.Logout, generalRateLimit)
	auth.POST("/refresh", authHandler.Refresh, generalRateLimit)
	// 全デバイスからログアウト（要認証）。ユーザーID単位で数えるため、トークン確認 → レート制限 → 必須認証の順に置く
	auth.POST("/revoke-all", authHandler.RevokeAll, optionalAuthMiddleware, generalRateLimit, jwtMiddleware)

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

	// ============================================================
	// 管理画面（運営者向け・html/template）
	// SNS アプリの一般ユーザー（users.role=admin を含む）とは別の、管理画面専用アカウントでログインする
	// ============================================================
	if cfg.AdminPanelEnabled() {
		renderer, err := adminui.NewRenderer()
		if err != nil {
			logger.L().Fatal().Err(err).Msg("admin panel templates load failed")
		}
		adminPanelSvc := service.NewAdminPanelService(cfg, userRepo, adminUserRepo, adminRepo, passwordResetRepo, dashboardRepo, refreshTokenRepo)
		adminPanelHandler := handler.NewAdminPanelHandler(adminPanelSvc, renderer, cfg.AdminSessionSecret, cfg.CookieSecure, cfg.PasswordResetTokenTTL)

		// 全ルートにセキュリティヘッダー・リクエストサイズ上限（フォームのみのため 64KB）・一般 API と同じレート制限を掛ける。
		// ログインは認証系のレート制限（IP あたり 5 回/分）も掛けてブルートフォースを防ぐ
		panel := e.Group("/admin", appMiddleware.AdminSecurityHeaders(), middleware.BodyLimit("64K"), generalRateLimit)
		panel.GET("/static/*", echo.WrapHandler(http.StripPrefix("/admin/static/", http.FileServer(http.FS(adminui.StaticFS())))))
		panel.GET("/login", adminPanelHandler.LoginPage)
		panel.POST("/login", adminPanelHandler.Login, authRateLimit)

		// ログインページ以外はすべて管理者セッションの確認（＋ POST は CSRF トークンの検証）を通す
		adminSession := appMiddleware.AdminSession(cfg.AdminSessionSecret, adminRepo, cfg.CookieSecure)
		panelAuthed := panel.Group("", adminSession)
		panelAuthed.POST("/logout", adminPanelHandler.Logout)
		panelAuthed.GET("", adminPanelHandler.Dashboard)
		panelAuthed.GET("/", adminPanelHandler.Dashboard)
		panelAuthed.GET("/users", adminPanelHandler.ListUsers)
		panelAuthed.GET("/users/:id", adminPanelHandler.GetUser)
		panelAuthed.POST("/users/:id/approve", adminPanelHandler.ApproveUser)
		panelAuthed.POST("/users/:id/reject", adminPanelHandler.RejectUser)
		panelAuthed.GET("/password-resets", adminPanelHandler.ListPasswordResets)
		panelAuthed.POST("/password-resets/:id/issue", adminPanelHandler.IssuePasswordResetLink)
		panelAuthed.GET("/admins", adminPanelHandler.ListAdmins)
		panelAuthed.POST("/admins", adminPanelHandler.CreateAdmin)

		if !cfg.RootAdminEnabled() {
			logger.L().Warn().Msg("ADMIN_ROOT_USERNAME / ADMIN_ROOT_PASSWORD are not set: root admin login is disabled")
		}
		logger.L().Info().Str("path", "/admin").Msg("admin panel enabled")
	} else {
		logger.L().Warn().Msg("ADMIN_SESSION_SECRET is not set: admin panel (/admin) is disabled")
	}

	addr := fmt.Sprintf(":%s", cfg.Port)
	logger.L().Info().Str("addr", addr).Str("env", cfg.Env).Msg("server started")
	if err := e.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.L().Fatal().Err(err).Msg("server stopped")
	}
}

// accessLogActions はアクセスログの action（操作の日本語名）に使う表を、Swagger 定義の @Summary から作る。
// 読み込めなくてもサーバーは起動し、action は「不明な操作」になる
func accessLogActions() appMiddleware.ActionCatalog {
	actions, err := appMiddleware.ActionsFromSwagger([]byte(docs.SwaggerInfo.ReadDoc()), docs.SwaggerInfo.BasePath)
	if err != nil {
		logger.L().Warn().Err(err).Msg("failed to load access log actions from swagger")
		actions = appMiddleware.ActionCatalog{}
	}
	for route, name := range handler.AdminPanelActions {
		actions[route] = name
	}
	actions["GET /admin/"] = handler.AdminPanelActions["GET /admin"]
	return actions
}
