package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/atyahara/sns-backend/internal/adminsession"
	"github.com/atyahara/sns-backend/internal/adminui"
	"github.com/atyahara/sns-backend/internal/httperr"
	"github.com/atyahara/sns-backend/internal/logger"
	appMiddleware "github.com/atyahara/sns-backend/internal/middleware"
	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/atyahara/sns-backend/internal/service"
	"github.com/atyahara/sns-backend/internal/utils"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// AdminPanelActions は管理画面（HTML）のルートのアクセスログ用の操作名。
// 管理画面は Swagger（API 定義）に載らないため、ここで定義して main.go でアクセスログの表に加える
var AdminPanelActions = map[string]string{
	"GET /admin/static/*":                   "管理画面の静的ファイル取得",
	"GET /admin/login":                      "管理画面ログインページの表示",
	"POST /admin/login":                     "管理画面へのログイン",
	"POST /admin/logout":                    "管理画面からのログアウト",
	"GET /admin":                            "管理画面ダッシュボードの表示",
	"GET /admin/users":                      "管理画面のユーザー一覧の表示",
	"GET /admin/users/:id":                  "管理画面のユーザー詳細の表示",
	"POST /admin/users/:id/approve":         "管理画面でのユーザー承認",
	"POST /admin/users/:id/reject":          "管理画面でのユーザー拒否",
	"GET /admin/password-resets":            "管理画面のパスワードリセット申請一覧の表示",
	"POST /admin/password-resets/:id/issue": "管理画面でのパスワードリセットリンク発行",
	"GET /admin/admins":                     "管理画面の管理者ユーザー一覧の表示",
	"POST /admin/admins":                    "管理画面での管理者ユーザー作成",
}

// AdminPanelHandler は管理画面（html/template で描画する運営者向け画面）のハンドラー
type AdminPanelHandler struct {
	svc           service.AdminPanelService
	renderer      *adminui.Renderer
	sessionSecret string
	cookieSecure  bool
	resetTTL      time.Duration
	now           func() time.Time
}

// NewAdminPanelHandler は AdminPanelHandler を生成する
func NewAdminPanelHandler(svc service.AdminPanelService, renderer *adminui.Renderer, sessionSecret string, cookieSecure bool, resetTTL time.Duration) *AdminPanelHandler {
	return &AdminPanelHandler{
		svc:           svc,
		renderer:      renderer,
		sessionSecret: sessionSecret,
		cookieSecure:  cookieSecure,
		resetTTL:      resetTTL,
		now:           time.Now,
	}
}

// adminPage はテンプレートに渡す共通データ
type adminPage struct {
	Title  string
	Active string
	Admin  *adminsession.Principal
	CSRF   string
	Flash  *adminFlash
	Data   interface{}
}

type adminFlash struct {
	Kind    string
	Message string
}

// adminFlashMessages は操作後のリダイレクト先（?msg=xxx）で表示するメッセージ。
// 任意の文字列を画面に出さないよう、決まったキーだけを受け付ける
var adminFlashMessages = map[string]adminFlash{
	"approved":       {"success", "ユーザーを承認しました。運営から本人へ承認の連絡（メール）を送ってください。"},
	"rejected":       {"success", "ユーザーの登録を拒否しました。"},
	"status_changed": {"error", "このユーザーは既に審査済みのため操作できませんでした。最新の状態を確認してください。"},
	"admin_created":  {"success", "管理者ユーザーを作成しました。"},
	"logged_out":     {"info", "ログアウトしました。"},
	"expired":        {"info", "セッションの有効期限が切れました。もう一度ログインしてください。"},
	"reset_closed":   {"error", "この申請は既に再設定が完了しています。"},
	"reset_inactive": {"error", "承認済みでないユーザーにはリセットリンクを発行できません。"},
}

func (h *AdminPanelHandler) render(c echo.Context, status int, page string, p adminPage) error {
	if principal, ok := appMiddleware.AdminPrincipalFromContext(c); ok {
		p.Admin = &principal
		p.CSRF = appMiddleware.AdminCSRFToken(c, h.sessionSecret)
	}
	if p.Flash == nil {
		if f, ok := adminFlashMessages[c.QueryParam("msg")]; ok {
			p.Flash = &f
		}
	}

	var buf bytes.Buffer
	if err := h.renderer.Render(&buf, page, p); err != nil {
		return httperr.Internal(c, "管理画面の表示に失敗しました", err)
	}
	return c.HTMLBlob(status, buf.Bytes())
}

func (h *AdminPanelHandler) renderError(c echo.Context, status int, title, message string) error {
	return h.render(c, status, "error", adminPage{Title: title, Data: map[string]string{"Message": message}})
}

func (h *AdminPanelHandler) principal(c echo.Context) adminsession.Principal {
	p, _ := appMiddleware.AdminPrincipalFromContext(c)
	return p
}

// ============================================================
// ログイン・ログアウト
// ============================================================

type loginPageData struct {
	Username string
}

// LoginPage は管理画面のログインページを表示する。ログイン済みならダッシュボードへ移動する
func (h *AdminPanelHandler) LoginPage(c echo.Context) error {
	if cookie, err := c.Cookie(adminsession.CookieName); err == nil {
		if _, err := adminsession.Parse(h.sessionSecret, cookie.Value); err == nil {
			return c.Redirect(http.StatusSeeOther, "/admin")
		}
	}
	p := adminPage{Title: "ログイン", Data: loginPageData{}}
	if c.QueryParam("expired") != "" {
		f := adminFlashMessages["expired"]
		p.Flash = &f
	}
	return h.render(c, http.StatusOK, "login", p)
}

// Login はユーザー名・パスワードを検証し、セッション Cookie を発行する
func (h *AdminPanelHandler) Login(c echo.Context) error {
	username := sanitizeAdminInput(c.FormValue("username"), 50)
	password := c.FormValue("password")

	fail := func(message string) error {
		return h.render(c, http.StatusUnauthorized, "login", adminPage{
			Title: "ログイン",
			Flash: &adminFlash{Kind: "error", Message: message},
			Data:  loginPageData{Username: username},
		})
	}
	if username == "" || password == "" || len(password) > 200 {
		return fail("ユーザー名とパスワードを入力してください")
	}

	principal, err := h.svc.Login(c.Request().Context(), username, password)
	if errors.Is(err, service.ErrAdminInvalidCredentials) {
		return fail("ユーザー名またはパスワードが正しくありません")
	}
	if err != nil {
		return httperr.Internal(c, "ログインに失敗しました", err)
	}

	token, err := adminsession.Issue(h.sessionSecret, principal, h.now())
	if err != nil {
		return httperr.Internal(c, "ログインに失敗しました", err)
	}
	appMiddleware.SetAdminSessionCookie(c, token, h.cookieSecure)
	return c.Redirect(http.StatusSeeOther, "/admin")
}

// Logout はセッション Cookie を削除してログインページへ戻す
func (h *AdminPanelHandler) Logout(c echo.Context) error {
	p := h.principal(c)
	appMiddleware.ClearAdminSessionCookie(c, h.cookieSecure)
	logAdminLogout(c, p)
	return c.Redirect(http.StatusSeeOther, appMiddleware.AdminLoginPath+"?msg=logged_out")
}

// ============================================================
// ダッシュボード
// ============================================================

type dashboardPageData struct {
	*service.Dashboard
	Chart dailyChart
}

// Dashboard は運営向けの集計（ユーザー数・承認待ち・投稿数・リセット申請・日別推移）を表示する
func (h *AdminPanelHandler) Dashboard(c echo.Context) error {
	d, err := h.svc.Dashboard(c.Request().Context())
	if err != nil {
		return httperr.Internal(c, "ダッシュボードの取得に失敗しました", err)
	}
	return h.render(c, http.StatusOK, "dashboard", adminPage{
		Title:  "ダッシュボード",
		Active: "dashboard",
		Data:   dashboardPageData{Dashboard: d, Chart: buildDailyChart(d.Daily)},
	})
}

// dailyChart は日別推移の SVG 棒グラフの描画位置（CSP で inline style を禁止しているため、座標を属性で渡す）
type dailyChart struct {
	Width, Height, BaseY int
	MaxPosts, MaxUsers   int64
	Bars                 []dailyBar
}

type dailyBar struct {
	repository.DailyCount
	BarWidth               int
	PostsX, PostsY, PostsH int
	UsersX, UsersY, UsersH int
	LabelX, LabelY         int
}

func buildDailyChart(days []repository.DailyCount) dailyChart {
	const (
		slot     = 48
		barWidth = 16
		plotH    = 160
		top      = 8
		labelH   = 18
	)
	chart := dailyChart{Width: slot * len(days), Height: top + plotH + labelH, BaseY: top + plotH}
	if chart.Width == 0 {
		chart.Width = slot
	}
	for _, d := range days {
		if d.Posts > chart.MaxPosts {
			chart.MaxPosts = d.Posts
		}
		if d.Users > chart.MaxUsers {
			chart.MaxUsers = d.Users
		}
	}
	scale := func(v, max int64) int {
		if max == 0 {
			return 0
		}
		return int(v * plotH / max)
	}
	for i, d := range days {
		x := i*slot + (slot-barWidth*2-2)/2
		ph, uh := scale(d.Posts, chart.MaxPosts), scale(d.Users, chart.MaxUsers)
		chart.Bars = append(chart.Bars, dailyBar{
			DailyCount: d,
			BarWidth:   barWidth,
			PostsX:     x, PostsY: chart.BaseY - ph, PostsH: ph,
			UsersX: x + barWidth + 2, UsersY: chart.BaseY - uh, UsersH: uh,
			LabelX: i*slot + slot/2, LabelY: chart.Height - 4,
		})
	}
	return chart
}

// ============================================================
// ユーザー
// ============================================================

type usersPageData struct {
	Status     string
	Users      []model.User
	NextCursor string
}

var userStatusFilters = map[string]bool{
	"": true, model.UserStatusPending: true, model.UserStatusApproved: true, model.UserStatusRejected: true,
}

// ListUsers はユーザー一覧を表示する（?status=pending|approved|rejected で絞り込み）
func (h *AdminPanelHandler) ListUsers(c echo.Context) error {
	status := c.QueryParam("status")
	if !userStatusFilters[status] {
		return h.renderError(c, http.StatusBadRequest, "入力内容に誤りがあります", "ステータスの指定が正しくありません。")
	}
	cursor, ok := validCursor(c.QueryParam("cursor"))
	if !ok {
		return h.renderError(c, http.StatusBadRequest, "入力内容に誤りがあります", "ページの指定が正しくありません。")
	}

	users, next, err := h.svc.ListUsers(c.Request().Context(), status, cursor)
	if err != nil {
		return httperr.Internal(c, "ユーザー一覧の取得に失敗しました", err)
	}
	return h.render(c, http.StatusOK, "users", adminPage{
		Title:  "ユーザー一覧",
		Active: "users",
		Data:   usersPageData{Status: status, Users: users, NextCursor: next},
	})
}

// GetUser はユーザー詳細を表示する
func (h *AdminPanelHandler) GetUser(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return h.renderError(c, http.StatusNotFound, "ユーザーが見つかりません", "指定されたユーザーは存在しません。")
	}
	detail, err := h.svc.GetUserDetail(c.Request().Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		return h.renderError(c, http.StatusNotFound, "ユーザーが見つかりません", "指定されたユーザーは存在しません。")
	}
	if err != nil {
		return httperr.Internal(c, "ユーザー情報の取得に失敗しました", err)
	}
	return h.render(c, http.StatusOK, "user_detail", adminPage{
		Title:  "@" + detail.User.Handle,
		Active: "users",
		Data:   detail,
	})
}

// ApproveUser はユーザーを承認する（承認後のメール連絡は運営が手動で行う）
func (h *AdminPanelHandler) ApproveUser(c echo.Context) error {
	return h.changeUserStatus(c, h.svc.ApproveUser, "approved")
}

// RejectUser はユーザーの登録を拒否する
func (h *AdminPanelHandler) RejectUser(c echo.Context) error {
	return h.changeUserStatus(c, h.svc.RejectUser, "rejected")
}

func (h *AdminPanelHandler) changeUserStatus(
	c echo.Context,
	op func(ctx context.Context, actor adminsession.Principal, id uuid.UUID) (*model.User, error),
	successMsg string,
) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return h.renderError(c, http.StatusNotFound, "ユーザーが見つかりません", "指定されたユーザーは存在しません。")
	}
	_, err = op(c.Request().Context(), h.principal(c), id)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return h.renderError(c, http.StatusNotFound, "ユーザーが見つかりません", "指定されたユーザーは存在しません。")
	case errors.Is(err, service.ErrUserStatusConflict):
		return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/admin/users/%s?msg=status_changed", id))
	case err != nil:
		return httperr.Internal(c, "ユーザーの審査に失敗しました", err)
	}
	return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/admin/users/%s?msg=%s", id, successMsg))
}

// ============================================================
// パスワードリセット
// ============================================================

type passwordResetsPageData struct {
	Status     string
	Requests   []model.PasswordResetRequest
	NextCursor string
	TTL        string
}

var resetStatusFilters = map[string]bool{
	"": true, model.ResetRequestPending: true, model.ResetRequestIssued: true, model.ResetRequestCompleted: true,
}

// ListPasswordResets はパスワードリセット申請一覧を表示する
func (h *AdminPanelHandler) ListPasswordResets(c echo.Context) error {
	status := c.QueryParam("status")
	if !resetStatusFilters[status] {
		return h.renderError(c, http.StatusBadRequest, "入力内容に誤りがあります", "ステータスの指定が正しくありません。")
	}
	cursor, ok := validCursor(c.QueryParam("cursor"))
	if !ok {
		return h.renderError(c, http.StatusBadRequest, "入力内容に誤りがあります", "ページの指定が正しくありません。")
	}

	requests, next, err := h.svc.ListResetRequests(c.Request().Context(), status, cursor)
	if err != nil {
		return httperr.Internal(c, "パスワードリセット申請の取得に失敗しました", err)
	}
	return h.render(c, http.StatusOK, "password_resets", adminPage{
		Title:  "パスワードリセット申請",
		Active: "password_resets",
		Data: passwordResetsPageData{
			Status: status, Requests: requests, NextCursor: next, TTL: formatTTL(h.resetTTL),
		},
	})
}

// IssuePasswordResetLink はユーザー専用のリセットリンク（有効期限付き・一回限り）を発行し、その場で 1 度だけ表示する
func (h *AdminPanelHandler) IssuePasswordResetLink(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return h.renderError(c, http.StatusNotFound, "申請が見つかりません", "指定された申請は存在しません。")
	}
	link, err := h.svc.IssueResetLink(c.Request().Context(), h.principal(c), id)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return h.renderError(c, http.StatusNotFound, "申請が見つかりません", "指定された申請は存在しません。")
	case errors.Is(err, service.ErrResetRequestClosed):
		return c.Redirect(http.StatusSeeOther, "/admin/password-resets?msg=reset_closed")
	case errors.Is(err, service.ErrResetUserNotApproved):
		return c.Redirect(http.StatusSeeOther, "/admin/password-resets?msg=reset_inactive")
	case err != nil:
		return httperr.Internal(c, "リセットリンクの発行に失敗しました", err)
	}
	return h.render(c, http.StatusOK, "reset_link", adminPage{
		Title:  "リセットリンクを発行しました",
		Active: "password_resets",
		Data:   link,
	})
}

// ============================================================
// 管理者ユーザー
// ============================================================

type adminsPageData struct {
	Admins     []model.Admin
	NextCursor string
	Username   string
}

// ListAdmins は管理者ユーザー一覧（ルート管理者には作成フォームも）を表示する
func (h *AdminPanelHandler) ListAdmins(c echo.Context) error {
	return h.renderAdmins(c, http.StatusOK, "", nil)
}

func (h *AdminPanelHandler) renderAdmins(c echo.Context, status int, username string, flash *adminFlash) error {
	cursor, ok := validCursor(c.QueryParam("cursor"))
	if !ok {
		return h.renderError(c, http.StatusBadRequest, "入力内容に誤りがあります", "ページの指定が正しくありません。")
	}
	admins, next, err := h.svc.ListAdmins(c.Request().Context(), cursor)
	if err != nil {
		return httperr.Internal(c, "管理者ユーザーの取得に失敗しました", err)
	}
	return h.render(c, status, "admins", adminPage{
		Title:  "管理者ユーザー",
		Active: "admins",
		Flash:  flash,
		Data:   adminsPageData{Admins: admins, NextCursor: next, Username: username},
	})
}

// CreateAdmin は管理者ユーザーを作成する（ルート管理者のみ）
func (h *AdminPanelHandler) CreateAdmin(c echo.Context) error {
	username := sanitizeAdminInput(c.FormValue("username"), 50)
	password := c.FormValue("password")

	_, err := h.svc.CreateAdmin(c.Request().Context(), h.principal(c), username, password)
	invalid := func(message string) error {
		return h.renderAdmins(c, http.StatusBadRequest, username, &adminFlash{Kind: "error", Message: message})
	}
	switch {
	case errors.Is(err, service.ErrAdminForbidden):
		return h.renderError(c, http.StatusForbidden, "権限がありません", "管理者ユーザーの作成はルート管理者だけが行えます。")
	case errors.Is(err, service.ErrAdminInvalidUsername):
		return invalid("ユーザー名は 3〜50 文字の英数字・アンダースコア・ドット・ハイフンで入力してください（ルート管理者と同じ名前は使えません）")
	case errors.Is(err, service.ErrAdminInvalidPassword):
		return invalid(fmt.Sprintf("パスワードは %d 文字以上・%d バイト以内で入力してください", service.AdminPasswordMinLength, service.AdminPasswordMaxBytes))
	case errors.Is(err, service.ErrAdminUsernameTaken):
		return invalid("このユーザー名は既に使われています")
	case err != nil:
		return httperr.Internal(c, "管理者ユーザーの作成に失敗しました", err)
	}
	return c.Redirect(http.StatusSeeOther, "/admin/admins?msg=admin_created")
}

// ============================================================
// ヘルパー
// ============================================================

// validCursor は一覧の cursor（RFC3339 の日時）を検証する。空は最初のページ
func validCursor(raw string) (string, bool) {
	if raw == "" {
		return "", true
	}
	if _, err := utils.ParseCursor(raw); err != nil {
		return "", false
	}
	return raw, true
}

// sanitizeText は制御文字・ゼロ幅文字を除き、前後の空白を取り除く（HTML エスケープは表示側で行う）
func sanitizeText(s string) string {
	cleaned := strings.Map(func(r rune) rune {
		switch {
		case unicode.Is(unicode.Cf, r): // ゼロ幅スペース・BOM などの不可視の書式文字
			return -1
		case unicode.IsControl(r):
			return -1
		default:
			return r
		}
	}, s)
	return strings.TrimSpace(cleaned)
}

// logAdminLogout は管理画面からのログアウトを監査ログに残す
func logAdminLogout(c echo.Context, p adminsession.Principal) {
	logger.FromContext(c.Request().Context()).Info().
		Str("event", "admin_audit").
		Time("performed_at", time.Now()).
		Str("admin_kind", p.Kind).
		Str("admin_id", p.ID()).
		Str("admin_username", p.Username).
		Str("operation", "admin.logout").
		Str("target_type", "admin").
		Str("target_id", p.ID()).
		Str("result", "success").
		Msg("admin audit")
}

// sanitizeAdminInput は前後の空白・制御文字・ゼロ幅文字を除き、最大文字数で切り詰める
func sanitizeAdminInput(s string, maxRunes int) string {
	cleaned := sanitizeText(s)
	if r := []rune(cleaned); len(r) > maxRunes {
		cleaned = string(r[:maxRunes])
	}
	return cleaned
}

func formatTTL(d time.Duration) string {
	if d%time.Hour == 0 {
		return fmt.Sprintf("%d 時間", int(d.Hours()))
	}
	return fmt.Sprintf("%d 分", int(d.Minutes()))
}
