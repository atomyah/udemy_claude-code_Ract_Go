package service

import (
	"context"
	"testing"
	"time"

	"github.com/atyahara/sns-backend/internal/config"
	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

const (
	testJWTSecret        = "unit-test-jwt-secret"
	testJWTRefreshSecret = "unit-test-jwt-refresh-secret"
)

func testConfig() *config.Config {
	return &config.Config{
		JWTSecret:        testJWTSecret,
		JWTRefreshSecret: testJWTRefreshSecret,
		Env:              "test",
	}
}

// hashedUser はパスワードハッシュ済みのユーザーを作る
func hashedUser(t *testing.T, password string) *model.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	h := string(hash)
	return &model.User{
		ID:           uuid.New(),
		Email:        "taro@example.com",
		PasswordHash: &h,
		Handle:       "taro",
		DisplayName:  "タロウ",
		Theme:        "light",
		Role:         "user",
	}
}

// parseToken は署名検証したうえでクレームを取り出す
func parseToken(t *testing.T, tokenStr, secret string) jwt.MapClaims {
	t.Helper()
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(*jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	require.NoError(t, err)
	require.True(t, token.Valid)
	return claims
}

func TestAuthService_Register_Success(t *testing.T) {
	userRepo := new(mockUserRepo)
	userRepo.On("ExistsByEmail", mock.Anything, "taro@example.com").Return(false, nil)
	userRepo.On("ExistsByHandle", mock.Anything, "taro").Return(false, nil)
	userRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.User")).
		Run(func(args mock.Arguments) {
			// DBのBeforeCreateフックの代わりにIDを採番する
			args.Get(1).(*model.User).ID = uuid.New()
		}).Return(nil)

	svc := NewAuthService(testConfig(), userRepo, nil)
	result, err := svc.Register(context.Background(), &dto.RegisterRequest{
		Email:       "taro@example.com",
		Password:    "password123",
		Handle:      "taro",
		DisplayName: "タロウ",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.AccessToken)
	assert.NotEmpty(t, result.RefreshToken)
	assert.Equal(t, "taro", result.User.Handle)
	assert.Equal(t, "タロウ", result.User.DisplayName)
	userRepo.AssertExpectations(t)

	// パスワードはbcryptでハッシュ化されて保存される（平文で保存しない）
	created := userRepo.Calls[2].Arguments.Get(1).(*model.User)
	require.NotNil(t, created.PasswordHash)
	assert.NotEqual(t, "password123", *created.PasswordHash)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(*created.PasswordHash), []byte("password123")))
	assert.Equal(t, "user", created.Role)
	assert.Equal(t, "light", created.Theme)
}

func TestAuthService_Register_EmailTaken(t *testing.T) {
	userRepo := new(mockUserRepo)
	userRepo.On("ExistsByEmail", mock.Anything, "taro@example.com").Return(true, nil)

	svc := NewAuthService(testConfig(), userRepo, nil)
	_, err := svc.Register(context.Background(), &dto.RegisterRequest{
		Email: "taro@example.com", Password: "password123", Handle: "taro", DisplayName: "タロウ",
	})

	assert.ErrorIs(t, err, ErrEmailTaken)
	userRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestAuthService_Register_HandleTaken(t *testing.T) {
	userRepo := new(mockUserRepo)
	userRepo.On("ExistsByEmail", mock.Anything, "taro@example.com").Return(false, nil)
	userRepo.On("ExistsByHandle", mock.Anything, "taro").Return(true, nil)

	svc := NewAuthService(testConfig(), userRepo, nil)
	_, err := svc.Register(context.Background(), &dto.RegisterRequest{
		Email: "taro@example.com", Password: "password123", Handle: "taro", DisplayName: "タロウ",
	})

	assert.ErrorIs(t, err, ErrHandleTaken)
	userRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestAuthService_Login_Success(t *testing.T) {
	user := hashedUser(t, "password123")
	userRepo := new(mockUserRepo)
	userRepo.On("FindByEmail", mock.Anything, "taro@example.com").Return(user, nil)

	svc := NewAuthService(testConfig(), userRepo, nil)
	result, err := svc.Login(context.Background(), &dto.LoginRequest{
		Email: "taro@example.com", Password: "password123",
	})

	require.NoError(t, err)
	assert.Equal(t, user.ID.String(), result.User.ID)

	// アクセストークンはJWT_SECRETで検証でき、sub/roleを含み有効期限は15分
	claims := parseToken(t, result.AccessToken, testJWTSecret)
	assert.Equal(t, user.ID.String(), claims["sub"])
	assert.Equal(t, "user", claims["role"])
	exp := time.Unix(int64(claims["exp"].(float64)), 0)
	assert.WithinDuration(t, time.Now().Add(15*time.Minute), exp, time.Minute)

	// リフレッシュトークンは別の鍵で署名され、有効期限は7日
	refreshClaims := parseToken(t, result.RefreshToken, testJWTRefreshSecret)
	refreshExp := time.Unix(int64(refreshClaims["exp"].(float64)), 0)
	assert.WithinDuration(t, time.Now().Add(7*24*time.Hour), refreshExp, time.Minute)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	user := hashedUser(t, "password123")
	userRepo := new(mockUserRepo)
	userRepo.On("FindByEmail", mock.Anything, "taro@example.com").Return(user, nil)

	svc := NewAuthService(testConfig(), userRepo, nil)
	_, err := svc.Login(context.Background(), &dto.LoginRequest{
		Email: "taro@example.com", Password: "wrong-password",
	})

	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	userRepo := new(mockUserRepo)
	userRepo.On("FindByEmail", mock.Anything, "nobody@example.com").Return(nil, repository.ErrNotFound)

	svc := NewAuthService(testConfig(), userRepo, nil)
	_, err := svc.Login(context.Background(), &dto.LoginRequest{
		Email: "nobody@example.com", Password: "password123",
	})

	// ユーザーの存在を推測されないよう、存在しない場合も認証エラーを返す
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthService_Login_SuspendedAccount(t *testing.T) {
	user := hashedUser(t, "password123")
	user.IsSuspended = true
	userRepo := new(mockUserRepo)
	userRepo.On("FindByEmail", mock.Anything, "taro@example.com").Return(user, nil)

	svc := NewAuthService(testConfig(), userRepo, nil)
	_, err := svc.Login(context.Background(), &dto.LoginRequest{
		Email: "taro@example.com", Password: "password123",
	})

	assert.ErrorIs(t, err, ErrAccountSuspended)
}

func TestAuthService_Login_GoogleOnlyAccountHasNoPassword(t *testing.T) {
	user := &model.User{ID: uuid.New(), Email: "taro@example.com", Handle: "taro", Role: "user"}
	userRepo := new(mockUserRepo)
	userRepo.On("FindByEmail", mock.Anything, "taro@example.com").Return(user, nil)

	svc := NewAuthService(testConfig(), userRepo, nil)
	_, err := svc.Login(context.Background(), &dto.LoginRequest{
		Email: "taro@example.com", Password: "password123",
	})

	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthService_RefreshAccessToken_Success(t *testing.T) {
	user := hashedUser(t, "password123")
	userRepo := new(mockUserRepo)
	userRepo.On("FindByEmail", mock.Anything, user.Email).Return(user, nil)
	userRepo.On("FindByID", mock.Anything, user.ID).Return(user, nil)

	svc := NewAuthService(testConfig(), userRepo, nil)
	login, err := svc.Login(context.Background(), &dto.LoginRequest{Email: user.Email, Password: "password123"})
	require.NoError(t, err)

	resp, err := svc.RefreshAccessToken(context.Background(), login.RefreshToken)

	require.NoError(t, err)
	claims := parseToken(t, resp.AccessToken, testJWTSecret)
	assert.Equal(t, user.ID.String(), claims["sub"])
}

func TestAuthService_RefreshAccessToken_ExpiredToken(t *testing.T) {
	expired := signRefreshToken(t, uuid.New().String(), "user", time.Now().Add(-time.Hour))

	svc := NewAuthService(testConfig(), new(mockUserRepo), nil)
	_, err := svc.RefreshAccessToken(context.Background(), expired)

	assert.ErrorIs(t, err, ErrExpiredToken)
}

func TestAuthService_RefreshAccessToken_InvalidSignature(t *testing.T) {
	claims := jwt.MapClaims{"sub": uuid.New().String(), "role": "user", "exp": time.Now().Add(time.Hour).Unix()}
	tampered, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("wrong-secret"))
	require.NoError(t, err)

	svc := NewAuthService(testConfig(), new(mockUserRepo), nil)
	_, err = svc.RefreshAccessToken(context.Background(), tampered)

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestAuthService_RefreshAccessToken_SuspendedAccount(t *testing.T) {
	user := hashedUser(t, "password123")
	user.IsSuspended = true
	userRepo := new(mockUserRepo)
	userRepo.On("FindByID", mock.Anything, user.ID).Return(user, nil)

	token := signRefreshToken(t, user.ID.String(), "user", time.Now().Add(time.Hour))
	svc := NewAuthService(testConfig(), userRepo, nil)
	_, err := svc.RefreshAccessToken(context.Background(), token)

	assert.ErrorIs(t, err, ErrAccountSuspended)
}

func TestAuthService_RefreshAccessToken_UnknownUser(t *testing.T) {
	userID := uuid.New()
	userRepo := new(mockUserRepo)
	userRepo.On("FindByID", mock.Anything, userID).Return(nil, repository.ErrNotFound)

	token := signRefreshToken(t, userID.String(), "user", time.Now().Add(time.Hour))
	svc := NewAuthService(testConfig(), userRepo, nil)
	_, err := svc.RefreshAccessToken(context.Background(), token)

	assert.ErrorIs(t, err, ErrInvalidToken)
}

// signRefreshToken はテスト用のリフレッシュトークンを生成する
func signRefreshToken(t *testing.T, sub, role string, exp time.Time) string {
	t.Helper()
	claims := jwt.MapClaims{"sub": sub, "role": role, "exp": exp.Unix(), "iat": time.Now().Unix()}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testJWTRefreshSecret))
	require.NoError(t, err)
	return signed
}
