package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	firebase "firebase.google.com/go/v4"
	"github.com/atyahara/sns-backend/internal/authtoken"
	"github.com/atyahara/sns-backend/internal/config"
	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken         = errors.New("email already taken")
	ErrHandleTaken        = errors.New("handle already taken")
	ErrAccountSuspended   = errors.New("account suspended")
	ErrExpiredToken       = errors.New("token expired")
	ErrInvalidToken       = errors.New("invalid token")
	ErrRevokedToken       = errors.New("token revoked")
	// ErrTokenReused は使用済みリフレッシュトークンの再利用（漏洩の兆候）を表す
	ErrTokenReused     = errors.New("refresh token reused")
	ErrInvalidIDToken  = errors.New("invalid firebase id token")
	ErrInvalidPassword = errors.New("invalid current password")
	ErrNoPassword      = errors.New("account has no password set")
	// ErrPasswordTooLong は bcrypt の上限（72バイト）を超えたパスワードを表す。
	// validator の max=72 は「文字数」で数えるため、日本語など1文字が複数バイトになる
	// パスワードはバリデーションを通過したあとでbcryptが弾く。
	ErrPasswordTooLong = errors.New("password exceeds bcrypt 72 bytes limit")
)

var handleSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// AuthResult はサービス層からハンドラー層へ返す認証結果。
// トークンはハンドラー層で HttpOnly Cookie に詰めて返す
type AuthResult struct {
	AccessToken  string
	RefreshToken string
	User         dto.UserResponse
}

type AuthService interface {
	Register(ctx context.Context, req *dto.RegisterRequest) (*AuthResult, error)
	Login(ctx context.Context, req *dto.LoginRequest) (*AuthResult, error)
	LoginWithGoogle(ctx context.Context, idToken string) (*AuthResult, error)
	// Refresh はリフレッシュトークンを検証し、使用済みにしたうえで新しいアクセストークン・リフレッシュトークンを発行する（ローテーション）
	Refresh(ctx context.Context, refreshToken string) (*AuthResult, error)
	// Logout はリフレッシュトークン（とそのローテーション系列）を失効させる
	Logout(ctx context.Context, refreshToken string) error
	// RevokeAll はユーザーの全リフレッシュトークンを失効させる（全デバイスからログアウト）
	RevokeAll(ctx context.Context, userID uuid.UUID) error
}

type authService struct {
	cfg              *config.Config
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	fbApp            *firebase.App
	now              func() time.Time
}

func NewAuthService(cfg *config.Config, userRepo repository.UserRepository, refreshTokenRepo repository.RefreshTokenRepository, fbApp *firebase.App) AuthService {
	return &authService{cfg: cfg, userRepo: userRepo, refreshTokenRepo: refreshTokenRepo, fbApp: fbApp, now: time.Now}
}

func (s *authService) Register(ctx context.Context, req *dto.RegisterRequest) (*AuthResult, error) {
	emailExists, err := s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if emailExists {
		return nil, ErrEmailTaken
	}

	handleExists, err := s.userRepo.ExistsByHandle(ctx, req.Handle)
	if err != nil {
		return nil, err
	}
	if handleExists {
		return nil, ErrHandleTaken
	}

	// パスワードをハッシュ化する
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if errors.Is(err, bcrypt.ErrPasswordTooLong) {
		return nil, ErrPasswordTooLong
	}
	if err != nil {
		return nil, err
	}
	passwordHash := string(hash)

	user := &model.User{
		Email:        req.Email,
		PasswordHash: &passwordHash,
		Handle:       req.Handle,
		DisplayName:  req.DisplayName,
		Theme:        "light",
		Role:         "user",
	}

	if err = s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return s.buildAuthResult(ctx, user)
}

func (s *authService) Login(ctx context.Context, req *dto.LoginRequest) (*AuthResult, error) {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	if user.IsSuspended {
		return nil, ErrAccountSuspended
	}

	if user.PasswordHash == nil {
		return nil, ErrInvalidCredentials
	}

	if err = bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.buildAuthResult(ctx, user)
}

func (s *authService) Refresh(ctx context.Context, refreshToken string) (*AuthResult, error) {
	if refreshToken == "" {
		return nil, ErrInvalidToken
	}
	now := s.now()

	stored, err := s.refreshTokenRepo.FindByHash(ctx, authtoken.HashRefreshToken(s.cfg.JWTRefreshSecret, refreshToken))
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, err
	}

	if stored.RevokedAt != nil {
		return nil, ErrRevokedToken
	}
	if stored.UsedAt != nil {
		// 交換済みのトークンが再び使われた = トークンが漏洩した可能性がある。系列ごと失効させる
		if err := s.refreshTokenRepo.RevokeFamily(ctx, stored.FamilyID, now); err != nil {
			return nil, err
		}
		return nil, ErrTokenReused
	}
	if !now.Before(stored.ExpiresAt) {
		return nil, ErrExpiredToken
	}

	user, err := s.userRepo.FindByID(ctx, stored.UserID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, err
	}
	if user.IsSuspended {
		if err := s.refreshTokenRepo.RevokeAllByUser(ctx, user.ID, now); err != nil {
			return nil, err
		}
		return nil, ErrAccountSuspended
	}

	next, raw, err := s.newRefreshTokenRecord(user.ID, stored.FamilyID, now)
	if err != nil {
		return nil, err
	}
	if err := s.refreshTokenRepo.Rotate(ctx, stored.ID, next, now); err != nil {
		if errors.Is(err, repository.ErrRefreshTokenNotActive) {
			// 同じトークンで同時にリフレッシュされ、先に交換された（再利用と同じ扱い）
			if rerr := s.refreshTokenRepo.RevokeFamily(ctx, stored.FamilyID, now); rerr != nil {
				return nil, rerr
			}
			return nil, ErrTokenReused
		}
		return nil, err
	}

	accessToken, err := authtoken.IssueAccessToken(s.cfg.JWTSecret, user.ID, now)
	if err != nil {
		return nil, err
	}
	return &AuthResult{AccessToken: accessToken, RefreshToken: raw, User: toSelfUserResponse(user)}, nil
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	stored, err := s.refreshTokenRepo.FindByHash(ctx, authtoken.HashRefreshToken(s.cfg.JWTRefreshSecret, refreshToken))
	if errors.Is(err, repository.ErrNotFound) {
		// 既に存在しないトークンでもログアウト自体は成功扱いにする（Cookie は削除する）
		return nil
	}
	if err != nil {
		return err
	}
	return s.refreshTokenRepo.RevokeFamily(ctx, stored.FamilyID, s.now())
}

func (s *authService) RevokeAll(ctx context.Context, userID uuid.UUID) error {
	return s.refreshTokenRepo.RevokeAllByUser(ctx, userID, s.now())
}

func (s *authService) LoginWithGoogle(ctx context.Context, idToken string) (*AuthResult, error) {
	authClient, err := s.fbApp.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("firebase auth client: %w", err)
	}

	token, err := authClient.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, ErrInvalidIDToken
	}

	email, _ := token.Claims["email"].(string)
	if email == "" {
		return nil, ErrInvalidIDToken
	}
	name, _ := token.Claims["name"].(string)
	picture, _ := token.Claims["picture"].(string)

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			return nil, err
		}

		handle, herr := s.generateUniqueHandle(ctx, email)
		if herr != nil {
			return nil, herr
		}
		if name == "" {
			name = handle
		}

		newUser := &model.User{
			Email:       email,
			Handle:      handle,
			DisplayName: name,
			Theme:       "light",
			Role:        "user",
		}
		if picture != "" {
			newUser.AvatarURL = &picture
		}

		if err := s.userRepo.Create(ctx, newUser); err != nil {
			return nil, err
		}
		user = newUser
	}

	if user.IsSuspended {
		return nil, ErrAccountSuspended
	}

	return s.buildAuthResult(ctx, user)
}

// generateUniqueHandle はメールアドレスのローカル部から未使用のハンドルを生成する
func (s *authService) generateUniqueHandle(ctx context.Context, email string) (string, error) {
	local := email
	if i := strings.Index(email, "@"); i > 0 {
		local = email[:i]
	}
	base := strings.ToLower(handleSanitizer.ReplaceAllString(local, "_"))
	if base == "" {
		base = "user"
	}
	if len(base) > 40 {
		base = base[:40]
	}

	candidate := base
	for i := 0; i < 1000; i++ {
		if i > 0 {
			candidate = fmt.Sprintf("%s%d", base, i)
		}
		exists, err := s.userRepo.ExistsByHandle(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique handle for %s", email)
}

// buildAuthResult はログイン成功時のトークンを発行する。リフレッシュトークンは新しい系列（family）として DB に保存する
func (s *authService) buildAuthResult(ctx context.Context, user *model.User) (*AuthResult, error) {
	now := s.now()
	accessToken, err := authtoken.IssueAccessToken(s.cfg.JWTSecret, user.ID, now)
	if err != nil {
		return nil, err
	}

	record, raw, err := s.newRefreshTokenRecord(user.ID, uuid.New(), now)
	if err != nil {
		return nil, err
	}
	if err := s.refreshTokenRepo.Create(ctx, record); err != nil {
		return nil, err
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: raw,
		User:         toSelfUserResponse(user),
	}, nil
}

// newRefreshTokenRecord は新しいリフレッシュトークンを生成し、DB 保存用のレコード（ハッシュのみ保持）と生の値を返す
func (s *authService) newRefreshTokenRecord(userID, familyID uuid.UUID, now time.Time) (*model.RefreshToken, string, error) {
	raw, err := authtoken.NewRefreshToken()
	if err != nil {
		return nil, "", err
	}
	return &model.RefreshToken{
		UserID:    userID,
		FamilyID:  familyID,
		TokenHash: authtoken.HashRefreshToken(s.cfg.JWTRefreshSecret, raw),
		ExpiresAt: now.Add(authtoken.RefreshTokenTTL),
	}, raw, nil
}

// toSelfUserResponse は本人向けのユーザー情報（role を含む）を返す
func toSelfUserResponse(u *model.User) dto.UserResponse {
	resp := ToUserResponse(u)
	role := u.Role
	resp.Role = &role
	return resp
}

// ToUserResponse はmodel.Userをdto.UserResponseに変換する
func ToUserResponse(u *model.User) dto.UserResponse {
	var birthday *string
	if u.Birthday != nil {
		s := u.Birthday.Format("2006-01-02")
		birthday = &s
	}
	return dto.UserResponse{
		ID:             u.ID.String(),
		Handle:         u.Handle,
		DisplayName:    u.DisplayName,
		AvatarURL:      u.AvatarURL,
		BannerURL:      u.BannerURL,
		Bio:            u.Bio,
		Location:       u.Location,
		WebsiteURL:     u.WebsiteURL,
		Birthday:       birthday,
		Theme:          u.Theme,
		FollowersCount: 0,
		FollowingCount: 0,
		IsFollowing:    false,
		CreatedAt:      u.CreatedAt.Format(time.RFC3339),
	}
}
