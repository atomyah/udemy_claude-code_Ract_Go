package service

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/atyahara/sns-backend/internal/adminsession"
	"github.com/atyahara/sns-backend/internal/authtoken"
	"github.com/atyahara/sns-backend/internal/config"
	"github.com/atyahara/sns-backend/internal/logger"
	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/atyahara/sns-backend/internal/utils"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

// 管理画面の業務エラー
var (
	ErrAdminInvalidCredentials = errors.New("invalid admin credentials")
	// ErrAdminForbidden はルート管理者だけが行える操作を通常管理者が行おうとしたことを表す
	ErrAdminForbidden = errors.New("admin operation forbidden")
	// ErrUserStatusConflict は承認・拒否できない状態のユーザー（拒否済みを再度拒否など）を表す
	ErrUserStatusConflict = errors.New("user status conflict")
	// ErrResetRequestClosed は完了済みのパスワードリセット申請にリンクを発行しようとしたことを表す
	ErrResetRequestClosed = errors.New("password reset request already completed")
	// ErrResetUserNotApproved は承認済みでないユーザーにリセットリンクを発行しようとしたことを表す
	ErrResetUserNotApproved = errors.New("password reset user is not approved")
	ErrAdminUsernameTaken   = errors.New("admin username already taken")
	ErrAdminInvalidUsername = errors.New("invalid admin username")
	ErrAdminInvalidPassword = errors.New("invalid admin password")
)

// 管理者アカウントの入力制約
const (
	AdminPasswordMinLength = 12
	AdminPasswordMaxBytes  = 72
	// AdminPageSize は管理画面の一覧の 1 ページの件数
	AdminPageSize = 20
	// DashboardDays はダッシュボードの日別グラフの日数
	DashboardDays = 14
)

var adminUsernamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{3,50}$`)

// UserDetail は管理画面のユーザー詳細
type UserDetail struct {
	User          *model.User
	PostCount     int64
	ResetRequests []model.PasswordResetRequest
}

// IssuedResetLink は発行したパスワードリセットリンク（画面に 1 度だけ表示する）
type IssuedResetLink struct {
	URL       string
	ExpiresAt time.Time
	User      model.User
}

// Dashboard は管理画面トップの表示内容
type Dashboard struct {
	Summary      *repository.DashboardSummary
	Daily        []repository.DailyCount
	PendingUsers []model.User
}

// AdminPanelService は管理画面（ユーザー審査・パスワードリセットリンク発行・管理者管理・ダッシュボード）を担う。
// 状態を変える操作はすべて監査ログ（event=admin_audit）に「誰が・いつ・何を・誰に対して」を記録する
type AdminPanelService interface {
	Login(ctx context.Context, username, password string) (adminsession.Principal, error)
	Dashboard(ctx context.Context) (*Dashboard, error)
	ListUsers(ctx context.Context, status, cursor string) ([]model.User, string, error)
	GetUserDetail(ctx context.Context, id uuid.UUID) (*UserDetail, error)
	ApproveUser(ctx context.Context, actor adminsession.Principal, id uuid.UUID) (*model.User, error)
	RejectUser(ctx context.Context, actor adminsession.Principal, id uuid.UUID) (*model.User, error)
	ListResetRequests(ctx context.Context, status, cursor string) ([]model.PasswordResetRequest, string, error)
	IssueResetLink(ctx context.Context, actor adminsession.Principal, requestID uuid.UUID) (*IssuedResetLink, error)
	ListAdmins(ctx context.Context, cursor string) ([]model.Admin, string, error)
	CreateAdmin(ctx context.Context, actor adminsession.Principal, username, password string) (*model.Admin, error)
}

type adminPanelService struct {
	cfg              *config.Config
	userRepo         repository.UserRepository
	adminUserRepo    repository.AdminUserRepository
	adminRepo        repository.AdminRepository
	resetRepo        repository.PasswordResetRepository
	dashboardRepo    repository.DashboardRepository
	refreshTokenRepo repository.RefreshTokenRepository
	now              func() time.Time
	// dummyHash は存在しない管理者名でのログインでも bcrypt の比較を行い、応答時間から存在を推測されないようにするためのハッシュ
	dummyHash []byte
}

func NewAdminPanelService(
	cfg *config.Config,
	userRepo repository.UserRepository,
	adminUserRepo repository.AdminUserRepository,
	adminRepo repository.AdminRepository,
	resetRepo repository.PasswordResetRepository,
	dashboardRepo repository.DashboardRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
) AdminPanelService {
	dummy, _ := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), 12)
	return &adminPanelService{
		cfg:              cfg,
		userRepo:         userRepo,
		adminUserRepo:    adminUserRepo,
		adminRepo:        adminRepo,
		resetRepo:        resetRepo,
		dashboardRepo:    dashboardRepo,
		refreshTokenRepo: refreshTokenRepo,
		now:              time.Now,
		dummyHash:        dummy,
	}
}

// ============================================================
// ログイン
// ============================================================

func (s *adminPanelService) Login(ctx context.Context, username, password string) (adminsession.Principal, error) {
	username = strings.TrimSpace(username)

	// ルート管理者（環境変数）。admins テーブルに同名がいてもルートを優先する
	if s.cfg.RootAdminEnabled() && strings.EqualFold(username, s.cfg.AdminRootUsername) {
		if !constantTimeEqual(password, s.cfg.AdminRootPassword) {
			s.logLoginFailure(ctx, username, "invalid_password")
			return adminsession.Principal{}, ErrAdminInvalidCredentials
		}
		p := adminsession.Principal{Kind: adminsession.KindRoot, Username: s.cfg.AdminRootUsername}
		s.audit(ctx, p, "admin.login", "admin", p.ID()).Msg("admin audit")
		return p, nil
	}

	admin, err := s.adminRepo.FindByUsername(ctx, username)
	if errors.Is(err, repository.ErrNotFound) {
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
		s.logLoginFailure(ctx, username, "unknown_username")
		return adminsession.Principal{}, ErrAdminInvalidCredentials
	}
	if err != nil {
		return adminsession.Principal{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		s.logLoginFailure(ctx, username, "invalid_password")
		return adminsession.Principal{}, ErrAdminInvalidCredentials
	}
	if !admin.IsActive {
		s.logLoginFailure(ctx, username, "inactive")
		return adminsession.Principal{}, ErrAdminInvalidCredentials
	}

	if err := s.adminRepo.UpdateLastLogin(ctx, admin.ID, s.now()); err != nil {
		return adminsession.Principal{}, err
	}
	p := adminsession.Principal{Kind: adminsession.KindAdmin, AdminID: admin.ID, Username: admin.Username}
	s.audit(ctx, p, "admin.login", "admin", p.ID()).Msg("admin audit")
	return p, nil
}

// constantTimeEqual は長さに関係なく定数時間で文字列を比較する（ハッシュを取って長さをそろえる）
func constantTimeEqual(a, b string) bool {
	ha := sha256.Sum256([]byte(a))
	hb := sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(ha[:], hb[:]) == 1
}

// ============================================================
// ダッシュボード
// ============================================================

func (s *adminPanelService) Dashboard(ctx context.Context) (*Dashboard, error) {
	summary, err := s.dashboardRepo.Summary(ctx, s.now())
	if err != nil {
		return nil, err
	}
	daily, err := s.dashboardRepo.Daily(ctx, DashboardDays)
	if err != nil {
		return nil, err
	}
	pending, _, err := s.adminUserRepo.List(ctx, model.UserStatusPending, nil, 5)
	if err != nil {
		return nil, err
	}
	return &Dashboard{Summary: summary, Daily: daily, PendingUsers: pending}, nil
}

// ============================================================
// ユーザー審査
// ============================================================

func (s *adminPanelService) ListUsers(ctx context.Context, status, cursor string) ([]model.User, string, error) {
	cursorTime, err := utils.ParseCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	users, next, err := s.adminUserRepo.List(ctx, status, cursorTime, AdminPageSize)
	if err != nil {
		return nil, "", err
	}
	return users, formatNextCursor(next), nil
}

func (s *adminPanelService) GetUserDetail(ctx context.Context, id uuid.UUID) (*UserDetail, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	postCount, err := s.adminUserRepo.CountPosts(ctx, id)
	if err != nil {
		return nil, err
	}
	resets, err := s.resetRepo.ListRequestsByUser(ctx, id, 10)
	if err != nil {
		return nil, err
	}
	return &UserDetail{User: user, PostCount: postCount, ResetRequests: resets}, nil
}

func (s *adminPanelService) ApproveUser(ctx context.Context, actor adminsession.Principal, id uuid.UUID) (*model.User, error) {
	// 承認待ちに加え、誤って拒否したユーザーも承認し直せる
	return s.changeStatus(ctx, actor, id, "user.approve",
		[]string{model.UserStatusPending, model.UserStatusRejected}, model.UserStatusApproved)
}

func (s *adminPanelService) RejectUser(ctx context.Context, actor adminsession.Principal, id uuid.UUID) (*model.User, error) {
	// 承認済みユーザーの利用停止は「停止（suspend）」で行うため、拒否は承認待ちのユーザーだけにする
	user, err := s.changeStatus(ctx, actor, id, "user.reject", []string{model.UserStatusPending}, model.UserStatusRejected)
	if err != nil {
		return nil, err
	}
	// 念のためログイン状態も無効にする（承認待ちのユーザーには通常リフレッシュトークンはない）
	if err := s.refreshTokenRepo.RevokeAllByUser(ctx, id, s.now()); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *adminPanelService) changeStatus(ctx context.Context, actor adminsession.Principal, id uuid.UUID, operation string, from []string, to string) (*model.User, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	before := user.Status

	err = s.adminUserRepo.UpdateStatus(ctx, id, from, to, actor.Label(), s.now())
	if errors.Is(err, repository.ErrStatusConflict) {
		s.auditFailure(ctx, actor, operation, "user", id.String(), "status_conflict").
			Str("target_handle", user.Handle).Str("status_before", before).Msg("admin audit")
		return nil, ErrUserStatusConflict
	}
	if err != nil {
		return nil, err
	}

	s.audit(ctx, actor, operation, "user", id.String()).
		Str("target_handle", user.Handle).
		Str("status_before", before).
		Str("status_after", to).
		Msg("admin audit")
	user.Status = to
	return user, nil
}

// ============================================================
// パスワードリセット
// ============================================================

func (s *adminPanelService) ListResetRequests(ctx context.Context, status, cursor string) ([]model.PasswordResetRequest, string, error) {
	cursorTime, err := utils.ParseCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	requests, next, err := s.resetRepo.ListRequests(ctx, status, cursorTime, AdminPageSize)
	if err != nil {
		return nil, "", err
	}
	return requests, formatNextCursor(next), nil
}

func (s *adminPanelService) IssueResetLink(ctx context.Context, actor adminsession.Principal, requestID uuid.UUID) (*IssuedResetLink, error) {
	req, err := s.resetRepo.FindRequestByID(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if req.Status == model.ResetRequestCompleted {
		s.auditFailure(ctx, actor, "password_reset.issue_link", "user", req.UserID.String(), "request_completed").
			Str("reset_request_id", requestID.String()).Msg("admin audit")
		return nil, ErrResetRequestClosed
	}
	if !req.User.IsApproved() {
		s.auditFailure(ctx, actor, "password_reset.issue_link", "user", req.UserID.String(), "user_not_approved").
			Str("reset_request_id", requestID.String()).Msg("admin audit")
		return nil, ErrResetUserNotApproved
	}

	raw, err := authtoken.NewPasswordResetToken()
	if err != nil {
		return nil, err
	}
	now := s.now()
	token := &model.PasswordResetToken{
		UserID:    req.UserID,
		RequestID: &req.ID,
		TokenHash: authtoken.HashPasswordResetToken(s.cfg.JWTRefreshSecret, raw),
		ExpiresAt: now.Add(s.cfg.PasswordResetTokenTTL),
		IssuedBy:  actor.Label(),
	}
	if err := s.resetRepo.IssueToken(ctx, token, actor.Label(), now); err != nil {
		return nil, err
	}

	// トークン・リンクはログに出さない（ログを見られるとパスワードを変更できてしまうため）
	s.audit(ctx, actor, "password_reset.issue_link", "user", req.UserID.String()).
		Str("target_handle", req.User.Handle).
		Str("reset_request_id", req.ID.String()).
		Time("expires_at", token.ExpiresAt).
		Msg("admin audit")

	return &IssuedResetLink{
		URL:       s.cfg.FrontendBaseURL + "/reset-password?token=" + url.QueryEscape(raw),
		ExpiresAt: token.ExpiresAt,
		User:      req.User,
	}, nil
}

// ============================================================
// 管理者ユーザー
// ============================================================

func (s *adminPanelService) ListAdmins(ctx context.Context, cursor string) ([]model.Admin, string, error) {
	cursorTime, err := utils.ParseCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	admins, next, err := s.adminRepo.List(ctx, cursorTime, AdminPageSize)
	if err != nil {
		return nil, "", err
	}
	return admins, formatNextCursor(next), nil
}

func (s *adminPanelService) CreateAdmin(ctx context.Context, actor adminsession.Principal, username, password string) (*model.Admin, error) {
	username = strings.TrimSpace(username)
	if !actor.IsRoot() {
		s.auditFailure(ctx, actor, "admin.create", "admin", "", "forbidden").Str("target_username", username).Msg("admin audit")
		return nil, ErrAdminForbidden
	}
	if !adminUsernamePattern.MatchString(username) || strings.EqualFold(username, s.cfg.AdminRootUsername) {
		return nil, ErrAdminInvalidUsername
	}
	if len([]rune(password)) < AdminPasswordMinLength || len(password) > AdminPasswordMaxBytes {
		return nil, ErrAdminInvalidPassword
	}

	exists, err := s.adminRepo.ExistsByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrAdminUsernameTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, err
	}
	admin := &model.Admin{
		Username:     username,
		PasswordHash: string(hash),
		Role:         model.AdminRoleAdmin,
		IsActive:     true,
		CreatedBy:    actor.Label(),
	}
	if err := s.adminRepo.Create(ctx, admin); err != nil {
		return nil, err
	}

	s.audit(ctx, actor, "admin.create", "admin", admin.ID.String()).
		Str("target_username", admin.Username).
		Str("target_role", admin.Role).
		Msg("admin audit")
	return admin, nil
}

// ============================================================
// 監査ログ
// ============================================================

// audit は成功した管理操作の監査ログ（INFO）を組み立てる。呼び出し側でフィールドを追加して Msg("admin audit") で出力する。
// 「いつ」は performed_at（と zerolog の time）、「誰が」は admin_*、「何を」は operation、「誰に対して」は target_* で表す。
// メールアドレスなどの個人情報・トークン・パスワードは出さない
func (s *adminPanelService) audit(ctx context.Context, actor adminsession.Principal, operation, targetType, targetID string) *zerolog.Event {
	return auditEvent(logger.FromContext(ctx).Info(), actor, operation, targetType, targetID, s.now()).Str("result", "success")
}

// auditFailure は拒否・失敗した管理操作の監査ログ（WARN）を組み立てる
func (s *adminPanelService) auditFailure(ctx context.Context, actor adminsession.Principal, operation, targetType, targetID, reason string) *zerolog.Event {
	return auditEvent(logger.FromContext(ctx).Warn(), actor, operation, targetType, targetID, s.now()).
		Str("result", "failure").
		Str("reason", reason)
}

func auditEvent(e *zerolog.Event, actor adminsession.Principal, operation, targetType, targetID string, now time.Time) *zerolog.Event {
	return e.Str("event", "admin_audit").
		Time("performed_at", now).
		Str("admin_kind", actor.Kind).
		Str("admin_id", actor.ID()).
		Str("admin_username", actor.Username).
		Str("operation", operation).
		Str("target_type", targetType).
		Str("target_id", targetID)
}

// logLoginFailure は管理画面のログイン失敗を記録する（ブルートフォースの検知用。パスワードは出さない）
func (s *adminPanelService) logLoginFailure(ctx context.Context, username, reason string) {
	logger.FromContext(ctx).Warn().
		Str("event", "admin_audit").
		Time("performed_at", s.now()).
		Str("operation", "admin.login").
		Str("attempted_username", username).
		Str("result", "failure").
		Str("reason", reason).
		Msg("admin audit")
}

func formatNextCursor(next *time.Time) string {
	if next == nil {
		return ""
	}
	return utils.FormatCursor(*next)
}
