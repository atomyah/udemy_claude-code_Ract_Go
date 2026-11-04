package service

import (
	"context"
	"errors"
	"time"

	"github.com/atyahara/sns-backend/internal/authtoken"
	"github.com/atyahara/sns-backend/internal/config"
	"github.com/atyahara/sns-backend/internal/logger"
	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// ErrInvalidResetToken はリセットリンクのトークンが無効（存在しない・使用済み・再発行で失効・期限切れ）であることを表す
var ErrInvalidResetToken = errors.New("invalid password reset token")

// PasswordResetService は一般ユーザー向けのパスワードリセット（申請・再設定）を担う。
// リセットリンクの発行は管理画面（AdminPanelService）で管理者が行い、ユーザーへの送付は運営が手動で行う
type PasswordResetService interface {
	// RequestReset はパスワードリセットを申請する。メールアドレスが登録されているかどうかは呼び出し元に伝えない
	RequestReset(ctx context.Context, email string) error
	// ResetPassword はリセットリンクのトークンを使って新しいパスワードを設定する（トークンは一回限り）
	ResetPassword(ctx context.Context, token, newPassword string) error
}

type passwordResetService struct {
	cfg       *config.Config
	userRepo  repository.UserRepository
	resetRepo repository.PasswordResetRepository
	now       func() time.Time
}

func NewPasswordResetService(cfg *config.Config, userRepo repository.UserRepository, resetRepo repository.PasswordResetRepository) PasswordResetService {
	return &passwordResetService{cfg: cfg, userRepo: userRepo, resetRepo: resetRepo, now: time.Now}
}

func (s *passwordResetService) RequestReset(ctx context.Context, email string) error {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if errors.Is(err, repository.ErrNotFound) {
		// 登録の有無を推測されないよう、存在しないメールアドレスでも成功として扱う
		return nil
	}
	if err != nil {
		return err
	}
	// 登録を拒否されたユーザーはリセットしても使えないため申請を作らない
	if user.Status == model.UserStatusRejected {
		return nil
	}

	created, err := s.resetRepo.CreateRequestIfNone(ctx, user.ID)
	if err != nil {
		return err
	}
	logger.FromContext(ctx).Info().
		Str("event", "password_reset_requested").
		Str("target_user_id", user.ID.String()).
		Bool("created", created).
		Msg("password reset requested")
	return nil
}

func (s *passwordResetService) ResetPassword(ctx context.Context, token, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if errors.Is(err, bcrypt.ErrPasswordTooLong) {
		return ErrPasswordTooLong
	}
	if err != nil {
		return err
	}

	userID, err := s.resetRepo.ConsumeToken(ctx, authtoken.HashPasswordResetToken(s.cfg.JWTRefreshSecret, token), string(hash), s.now())
	if errors.Is(err, repository.ErrResetTokenNotActive) {
		return ErrInvalidResetToken
	}
	if err != nil {
		return err
	}
	logger.FromContext(ctx).Info().
		Str("event", "password_reset_completed").
		Str("target_user_id", userID.String()).
		Msg("password reset completed")
	return nil
}
