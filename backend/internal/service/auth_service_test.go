package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/atyahara/sns-backend/internal/authtoken"
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

	svc := NewAuthService(testConfig(), userRepo, acceptingRefreshTokenRepo(), nil)
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

	svc := NewAuthService(testConfig(), userRepo, acceptingRefreshTokenRepo(), nil)
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

	svc := NewAuthService(testConfig(), userRepo, acceptingRefreshTokenRepo(), nil)
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

	svc := NewAuthService(testConfig(), userRepo, acceptingRefreshTokenRepo(), nil)
	result, err := svc.Login(context.Background(), &dto.LoginRequest{
		Email: "taro@example.com", Password: "password123",
	})

	require.NoError(t, err)
	assert.Equal(t, user.ID.String(), result.User.ID)

	// アクセストークンはJWT_SECRETで検証でき、ペイロードは user_id と exp のみ・有効期限は1時間
	claims := parseToken(t, result.AccessToken, testJWTSecret)
	assert.Len(t, claims, 2)
	assert.Equal(t, user.ID.String(), claims["user_id"])
	exp := time.Unix(int64(claims["exp"].(float64)), 0)
	assert.WithinDuration(t, time.Now().Add(time.Hour), exp, time.Minute)

	// 本人向けレスポンスには role が入る（フロントエンドの管理者画面の表示判定に使う）
	require.NotNil(t, result.User.Role)
	assert.Equal(t, "user", *result.User.Role)
	assert.NotEmpty(t, result.RefreshToken)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	user := hashedUser(t, "password123")
	userRepo := new(mockUserRepo)
	userRepo.On("FindByEmail", mock.Anything, "taro@example.com").Return(user, nil)

	svc := NewAuthService(testConfig(), userRepo, acceptingRefreshTokenRepo(), nil)
	_, err := svc.Login(context.Background(), &dto.LoginRequest{
		Email: "taro@example.com", Password: "wrong-password",
	})

	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	userRepo := new(mockUserRepo)
	userRepo.On("FindByEmail", mock.Anything, "nobody@example.com").Return(nil, repository.ErrNotFound)

	svc := NewAuthService(testConfig(), userRepo, acceptingRefreshTokenRepo(), nil)
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

	svc := NewAuthService(testConfig(), userRepo, acceptingRefreshTokenRepo(), nil)
	_, err := svc.Login(context.Background(), &dto.LoginRequest{
		Email: "taro@example.com", Password: "password123",
	})

	assert.ErrorIs(t, err, ErrAccountSuspended)
}

func TestAuthService_Login_GoogleOnlyAccountHasNoPassword(t *testing.T) {
	user := &model.User{ID: uuid.New(), Email: "taro@example.com", Handle: "taro", Role: "user"}
	userRepo := new(mockUserRepo)
	userRepo.On("FindByEmail", mock.Anything, "taro@example.com").Return(user, nil)

	svc := NewAuthService(testConfig(), userRepo, acceptingRefreshTokenRepo(), nil)
	_, err := svc.Login(context.Background(), &dto.LoginRequest{
		Email: "taro@example.com", Password: "password123",
	})

	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

// acceptingRefreshTokenRepo はリフレッシュトークンの保存を常に成功させるモック（トークン発行の検証が主目的でないテスト用）
func acceptingRefreshTokenRepo() *mockRefreshTokenRepo {
	repo := new(mockRefreshTokenRepo)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*model.RefreshToken")).Return(nil).Maybe()
	return repo
}

// refreshTestNow はリフレッシュ系テストの固定時刻
var refreshTestNow = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

// newRefreshTestService は時刻を固定した authService を返す
func newRefreshTestService(userRepo *mockUserRepo, refreshRepo *mockRefreshTokenRepo) *authService {
	svc := NewAuthService(testConfig(), userRepo, refreshRepo, nil).(*authService)
	svc.now = func() time.Time { return refreshTestNow }
	return svc
}

// storedRefreshToken は DB に保存されている（raw のハッシュを持つ）リフレッシュトークンのレコードを作る
func storedRefreshToken(userID uuid.UUID, raw string) *model.RefreshToken {
	return &model.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		FamilyID:  uuid.New(),
		TokenHash: authtoken.HashRefreshToken(testJWTRefreshSecret, raw),
		ExpiresAt: refreshTestNow.Add(24 * time.Hour),
	}
}

func TestAuthService_Login_StoresHashedRefreshTokenWithSevenDayExpiry(t *testing.T) {
	user := hashedUser(t, "password123")
	userRepo := new(mockUserRepo)
	userRepo.On("FindByEmail", mock.Anything, user.Email).Return(user, nil)
	refreshRepo := new(mockRefreshTokenRepo)
	var saved *model.RefreshToken
	refreshRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.RefreshToken")).
		Run(func(args mock.Arguments) { saved = args.Get(1).(*model.RefreshToken) }).Return(nil)

	svc := newRefreshTestService(userRepo, refreshRepo)
	result, err := svc.Login(context.Background(), &dto.LoginRequest{Email: user.Email, Password: "password123"})

	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, user.ID, saved.UserID)
	assert.NotEqual(t, uuid.Nil, saved.FamilyID, "ログインごとに新しい系列を作る")
	assert.Equal(t, refreshTestNow.Add(7*24*time.Hour), saved.ExpiresAt, "有効期限は7日")
	assert.Equal(t, authtoken.HashRefreshToken(testJWTRefreshSecret, result.RefreshToken), saved.TokenHash)
	assert.NotEqual(t, result.RefreshToken, saved.TokenHash, "DBにはトークン本体を保存しない")
	assert.Nil(t, saved.UsedAt)
	assert.Nil(t, saved.RevokedAt)
}

func TestAuthService_Login_RefreshTokenSaveFailure_ReturnsError(t *testing.T) {
	user := hashedUser(t, "password123")
	userRepo := new(mockUserRepo)
	userRepo.On("FindByEmail", mock.Anything, user.Email).Return(user, nil)
	refreshRepo := new(mockRefreshTokenRepo)
	refreshRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db down"))

	_, err := newRefreshTestService(userRepo, refreshRepo).Login(context.Background(), &dto.LoginRequest{Email: user.Email, Password: "password123"})

	assert.Error(t, err, "リフレッシュトークンを保存できなければログインさせない")
}

func TestAuthService_Refresh_RotatesTokens(t *testing.T) {
	user := hashedUser(t, "password123")
	stored := storedRefreshToken(user.ID, "old-refresh-token")
	userRepo := new(mockUserRepo)
	userRepo.On("FindByID", mock.Anything, user.ID).Return(user, nil)
	refreshRepo := new(mockRefreshTokenRepo)
	refreshRepo.On("FindByHash", mock.Anything, stored.TokenHash).Return(stored, nil)
	var next *model.RefreshToken
	refreshRepo.On("Rotate", mock.Anything, stored.ID, mock.AnythingOfType("*model.RefreshToken"), refreshTestNow).
		Run(func(args mock.Arguments) { next = args.Get(2).(*model.RefreshToken) }).Return(nil)

	result, err := newRefreshTestService(userRepo, refreshRepo).Refresh(context.Background(), "old-refresh-token")

	require.NoError(t, err)
	refreshRepo.AssertExpectations(t)

	// 新しいリフレッシュトークンが発行され、同じ系列・7日後の期限で保存される
	assert.NotEqual(t, "old-refresh-token", result.RefreshToken)
	require.NotNil(t, next)
	assert.Equal(t, stored.FamilyID, next.FamilyID)
	assert.Equal(t, user.ID, next.UserID)
	assert.Equal(t, authtoken.HashRefreshToken(testJWTRefreshSecret, result.RefreshToken), next.TokenHash)
	assert.Equal(t, refreshTestNow.Add(7*24*time.Hour), next.ExpiresAt)

	// 新しいアクセストークンも発行される
	claims := jwt.MapClaims{}
	_, _, err = jwt.NewParser().ParseUnverified(result.AccessToken, claims)
	require.NoError(t, err)
	assert.Equal(t, user.ID.String(), claims["user_id"])
	assert.Equal(t, float64(refreshTestNow.Add(time.Hour).Unix()), claims["exp"])
	assert.Equal(t, "taro", result.User.Handle)
}

func TestAuthService_Refresh_UnknownToken_ReturnsInvalid(t *testing.T) {
	refreshRepo := new(mockRefreshTokenRepo)
	refreshRepo.On("FindByHash", mock.Anything, mock.Anything).Return(nil, repository.ErrNotFound)

	_, err := newRefreshTestService(new(mockUserRepo), refreshRepo).Refresh(context.Background(), "unknown")

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestAuthService_Refresh_EmptyToken_ReturnsInvalid(t *testing.T) {
	refreshRepo := new(mockRefreshTokenRepo)

	_, err := newRefreshTestService(new(mockUserRepo), refreshRepo).Refresh(context.Background(), "")

	assert.ErrorIs(t, err, ErrInvalidToken)
	refreshRepo.AssertNotCalled(t, "FindByHash", mock.Anything, mock.Anything)
}

func TestAuthService_Refresh_ExpiredToken(t *testing.T) {
	stored := storedRefreshToken(uuid.New(), "expired")
	stored.ExpiresAt = refreshTestNow // 期限ちょうどは失効扱い
	refreshRepo := new(mockRefreshTokenRepo)
	refreshRepo.On("FindByHash", mock.Anything, stored.TokenHash).Return(stored, nil)

	_, err := newRefreshTestService(new(mockUserRepo), refreshRepo).Refresh(context.Background(), "expired")

	assert.ErrorIs(t, err, ErrExpiredToken)
	refreshRepo.AssertNotCalled(t, "Rotate", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestAuthService_Refresh_RevokedToken(t *testing.T) {
	stored := storedRefreshToken(uuid.New(), "revoked")
	revokedAt := refreshTestNow.Add(-time.Minute)
	stored.RevokedAt = &revokedAt
	refreshRepo := new(mockRefreshTokenRepo)
	refreshRepo.On("FindByHash", mock.Anything, stored.TokenHash).Return(stored, nil)

	_, err := newRefreshTestService(new(mockUserRepo), refreshRepo).Refresh(context.Background(), "revoked")

	assert.ErrorIs(t, err, ErrRevokedToken)
	refreshRepo.AssertNotCalled(t, "Rotate", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestAuthService_Refresh_ReusedToken_RevokesFamily(t *testing.T) {
	stored := storedRefreshToken(uuid.New(), "used")
	usedAt := refreshTestNow.Add(-time.Minute)
	stored.UsedAt = &usedAt
	refreshRepo := new(mockRefreshTokenRepo)
	refreshRepo.On("FindByHash", mock.Anything, stored.TokenHash).Return(stored, nil)
	refreshRepo.On("RevokeFamily", mock.Anything, stored.FamilyID, refreshTestNow).Return(nil)

	_, err := newRefreshTestService(new(mockUserRepo), refreshRepo).Refresh(context.Background(), "used")

	// 使用済みトークンの再利用は漏洩の兆候なので、同じ系列（そのログインの全トークン）を失効させる
	assert.ErrorIs(t, err, ErrTokenReused)
	refreshRepo.AssertExpectations(t)
	refreshRepo.AssertNotCalled(t, "Rotate", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestAuthService_Refresh_ConcurrentRotation_RevokesFamily(t *testing.T) {
	user := hashedUser(t, "password123")
	stored := storedRefreshToken(user.ID, "raced")
	userRepo := new(mockUserRepo)
	userRepo.On("FindByID", mock.Anything, user.ID).Return(user, nil)
	refreshRepo := new(mockRefreshTokenRepo)
	refreshRepo.On("FindByHash", mock.Anything, stored.TokenHash).Return(stored, nil)
	refreshRepo.On("Rotate", mock.Anything, stored.ID, mock.Anything, refreshTestNow).Return(repository.ErrRefreshTokenNotActive)
	refreshRepo.On("RevokeFamily", mock.Anything, stored.FamilyID, refreshTestNow).Return(nil)

	_, err := newRefreshTestService(userRepo, refreshRepo).Refresh(context.Background(), "raced")

	assert.ErrorIs(t, err, ErrTokenReused)
	refreshRepo.AssertExpectations(t)
}

func TestAuthService_Refresh_SuspendedAccount_RevokesAllTokens(t *testing.T) {
	user := hashedUser(t, "password123")
	user.IsSuspended = true
	stored := storedRefreshToken(user.ID, "suspended")
	userRepo := new(mockUserRepo)
	userRepo.On("FindByID", mock.Anything, user.ID).Return(user, nil)
	refreshRepo := new(mockRefreshTokenRepo)
	refreshRepo.On("FindByHash", mock.Anything, stored.TokenHash).Return(stored, nil)
	refreshRepo.On("RevokeAllByUser", mock.Anything, user.ID, refreshTestNow).Return(nil)

	_, err := newRefreshTestService(userRepo, refreshRepo).Refresh(context.Background(), "suspended")

	assert.ErrorIs(t, err, ErrAccountSuspended)
	refreshRepo.AssertExpectations(t)
}

func TestAuthService_Refresh_UnknownUser(t *testing.T) {
	stored := storedRefreshToken(uuid.New(), "orphan")
	userRepo := new(mockUserRepo)
	userRepo.On("FindByID", mock.Anything, stored.UserID).Return(nil, repository.ErrNotFound)
	refreshRepo := new(mockRefreshTokenRepo)
	refreshRepo.On("FindByHash", mock.Anything, stored.TokenHash).Return(stored, nil)

	_, err := newRefreshTestService(userRepo, refreshRepo).Refresh(context.Background(), "orphan")

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestAuthService_Logout_RevokesFamily(t *testing.T) {
	stored := storedRefreshToken(uuid.New(), "current")
	refreshRepo := new(mockRefreshTokenRepo)
	refreshRepo.On("FindByHash", mock.Anything, stored.TokenHash).Return(stored, nil)
	refreshRepo.On("RevokeFamily", mock.Anything, stored.FamilyID, refreshTestNow).Return(nil)

	require.NoError(t, newRefreshTestService(new(mockUserRepo), refreshRepo).Logout(context.Background(), "current"))

	refreshRepo.AssertExpectations(t)
}

func TestAuthService_Logout_UnknownOrMissingToken_Succeeds(t *testing.T) {
	refreshRepo := new(mockRefreshTokenRepo)
	refreshRepo.On("FindByHash", mock.Anything, mock.Anything).Return(nil, repository.ErrNotFound)
	svc := newRefreshTestService(new(mockUserRepo), refreshRepo)

	assert.NoError(t, svc.Logout(context.Background(), "unknown"))
	assert.NoError(t, svc.Logout(context.Background(), ""))
	refreshRepo.AssertNumberOfCalls(t, "FindByHash", 1)
	refreshRepo.AssertNotCalled(t, "RevokeFamily", mock.Anything, mock.Anything, mock.Anything)
}

func TestAuthService_RevokeAll_RevokesAllUserTokens(t *testing.T) {
	userID := uuid.New()
	refreshRepo := new(mockRefreshTokenRepo)
	refreshRepo.On("RevokeAllByUser", mock.Anything, userID, refreshTestNow).Return(nil)

	require.NoError(t, newRefreshTestService(new(mockUserRepo), refreshRepo).RevokeAll(context.Background(), userID))

	refreshRepo.AssertExpectations(t)
}
