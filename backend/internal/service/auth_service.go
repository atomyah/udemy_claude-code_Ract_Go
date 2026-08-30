package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	firebase "firebase.google.com/go/v4"
	"github.com/atyahara/sns-backend/internal/config"
	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/golang-jwt/jwt/v5"
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
	ErrInvalidIDToken     = errors.New("invalid firebase id token")
	ErrInvalidPassword    = errors.New("invalid current password")
	ErrNoPassword         = errors.New("account has no password set")
)

var handleSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// AuthResult はサービス層からハンドラー層へ返す認証結果（リフレッシュトークン含む）
type AuthResult struct {
	AccessToken  string
	RefreshToken string
	User         dto.UserResponse
}

type AuthService interface {
	Register(ctx context.Context, req *dto.RegisterRequest) (*AuthResult, error)
	Login(ctx context.Context, req *dto.LoginRequest) (*AuthResult, error)
	RefreshAccessToken(ctx context.Context, refreshToken string) (*dto.RefreshResponse, error)
	LoginWithGoogle(ctx context.Context, idToken string) (*AuthResult, error)
}

type authService struct {
	cfg      *config.Config
	userRepo repository.UserRepository
	fbApp    *firebase.App
}

func NewAuthService(cfg *config.Config, userRepo repository.UserRepository, fbApp *firebase.App) AuthService {
	return &authService{cfg: cfg, userRepo: userRepo, fbApp: fbApp}
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

	return s.buildAuthResult(user)
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

	return s.buildAuthResult(user)
}

func (s *authService) RefreshAccessToken(ctx context.Context, refreshToken string) (*dto.RefreshResponse, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(s.cfg.JWTRefreshSecret), nil
	})

	if err != nil || !token.Valid {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	sub, _ := claims["sub"].(string)
	role, _ := claims["role"].(string)

	userID, err := uuid.Parse(sub)
	if err != nil {
		return nil, ErrInvalidToken
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if user.IsSuspended {
		return nil, ErrAccountSuspended
	}

	accessToken, err := s.generateAccessToken(userID, role)
	if err != nil {
		return nil, err
	}

	return &dto.RefreshResponse{AccessToken: accessToken}, nil
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

	return s.buildAuthResult(user)
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

func (s *authService) buildAuthResult(user *model.User) (*AuthResult, error) {
	accessToken, err := s.generateAccessToken(user.ID, user.Role)
	if err != nil {
		return nil, err
	}
	refreshToken, err := s.generateRefreshToken(user.ID, user.Role)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         ToUserResponse(user),
	}, nil
}

func (s *authService) generateAccessToken(userID uuid.UUID, role string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  userID.String(),
		"role": role,
		// アクセストークンの有効期限を15分に設定
		"exp":  time.Now().Add(15 * time.Minute).Unix(),
		"iat":  time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

func (s *authService) generateRefreshToken(userID uuid.UUID, role string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  userID.String(),
		"role": role,
		"exp":  time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTRefreshSecret))
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
