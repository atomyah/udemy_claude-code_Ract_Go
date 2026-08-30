package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/service"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

// AuthHandler は認証関連のHTTPハンドラー
type AuthHandler struct {
	svc      service.AuthService
	validate *validator.Validate
}

// NewAuthHandler はAuthHandlerを生成する
func NewAuthHandler(svc service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc, validate: validator.New()}
}

// Register godoc
// @Summary      新規ユーザー登録
// @Description  メールアドレス・パスワード・ハンドル・表示名でユーザーを作成し、JWTを返す
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body     dto.RegisterRequest  true  "登録情報"
// @Success      201   {object} dto.AuthResponse      "登録成功"
// @Failure      400   {object} dto.ErrorResponse     "バリデーションエラー"
// @Failure      409   {object} dto.ErrorResponse     "メールアドレスまたはハンドルが既に使用されています"
// @Failure      500   {object} dto.ErrorResponse     "サーバーエラー"
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c echo.Context) error {
	var req dto.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "リクエストの形式が不正です",
		})
	}
	if err := h.validate.Struct(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "VALIDATION_ERROR",
			Message: validationMessage(err),
		})
	}

	result, err := h.svc.Register(c.Request().Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmailTaken):
			return c.JSON(http.StatusConflict, dto.ErrorResponse{Code: "EMAIL_TAKEN", Message: "このメールアドレスは既に使用されています"})
		case errors.Is(err, service.ErrHandleTaken):
			return c.JSON(http.StatusConflict, dto.ErrorResponse{Code: "HANDLE_TAKEN", Message: "このハンドルは既に使用されています"})
		default:
			return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "サーバーエラーが発生しました"})
		}
	}

	setRefreshTokenCookie(c, result.RefreshToken)

	return c.JSON(http.StatusCreated, dto.AuthResponse{
		AccessToken: result.AccessToken,
		User:        result.User,
	})
}

// Login godoc
// @Summary      ログイン
// @Description  メールアドレスとパスワードで認証し、Access Token（レスポンスボディ）とRefresh Token（HttpOnly Cookie）を返す
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body     dto.LoginRequest  true  "ログイン情報"
// @Success      200   {object} dto.AuthResponse   "ログイン成功"
// @Failure      400   {object} dto.ErrorResponse  "バリデーションエラー"
// @Failure      401   {object} dto.ErrorResponse  "メールアドレスまたはパスワードが不正"
// @Failure      500   {object} dto.ErrorResponse  "サーバーエラー"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c echo.Context) error {
	var req dto.LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "リクエストの形式が不正です",
		})
	}
	if err := h.validate.Struct(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "VALIDATION_ERROR",
			Message: validationMessage(err),
		})
	}

	result, err := h.svc.Login(c.Request().Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "INVALID_CREDENTIALS", Message: "メールアドレスまたはパスワードが正しくありません"})
		case errors.Is(err, service.ErrAccountSuspended):
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "ACCOUNT_SUSPENDED", Message: "このアカウントは停止されています"})
		default:
			return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "サーバーエラーが発生しました"})
		}
	}

	setRefreshTokenCookie(c, result.RefreshToken)

	return c.JSON(http.StatusOK, dto.AuthResponse{
		AccessToken: result.AccessToken,
		User:        result.User,
	})
}

// Logout godoc
// @Summary      ログアウト
// @Description  Refresh Token Cookie を削除する
// @Tags         auth
// @Produce      json
// @Success      204  "ログアウト成功"
// @Failure      500  {object} dto.ErrorResponse  "サーバーエラー"
// @Router       /auth/logout [post]
// @Security     BearerAuth
func (h *AuthHandler) Logout(c echo.Context) error {
	c.SetCookie(&http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
	return c.NoContent(http.StatusNoContent)
}

// Refresh godoc
// @Summary      アクセストークンの更新
// @Description  HttpOnly Cookie に保存されたRefresh Tokenを使いAccess Tokenを再発行する
// @Tags         auth
// @Produce      json
// @Success      200  {object} dto.RefreshResponse  "再発行成功"
// @Failure      401  {object} dto.ErrorResponse    "Refresh Tokenが無効または期限切れ"
// @Failure      500  {object} dto.ErrorResponse    "サーバーエラー"
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c echo.Context) error {
	cookie, err := c.Cookie("refresh_token")
	if err != nil {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Code:    "MISSING_TOKEN",
			Message: "Refresh Tokenが見つかりません",
		})
	}

	resp, err := h.svc.RefreshAccessToken(c.Request().Context(), cookie.Value)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrExpiredToken):
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "EXPIRED_TOKEN", Message: "Refresh Tokenの有効期限が切れています"})
		case errors.Is(err, service.ErrAccountSuspended):
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "ACCOUNT_SUSPENDED", Message: "このアカウントは停止されています"})
		default:
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "INVALID_TOKEN", Message: "無効なRefresh Tokenです"})
		}
	}

	return c.JSON(http.StatusOK, resp)
}

// GoogleLogin godoc
// @Summary      Google OAuthログイン
// @Description  Firebase AuthのIDトークンを検証し、初回は自動登録、JWTを返す
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body     dto.GoogleLoginRequest  true  "Firebase ID Token"
// @Success      200   {object} dto.AuthResponse         "ログイン/登録成功"
// @Failure      400   {object} dto.ErrorResponse        "バリデーションエラー"
// @Failure      401   {object} dto.ErrorResponse        "IDトークンが無効"
// @Failure      500   {object} dto.ErrorResponse        "サーバーエラー"
// @Router       /auth/google [post]
func (h *AuthHandler) GoogleLogin(c echo.Context) error {
	var req dto.GoogleLoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "リクエストの形式が不正です",
		})
	}
	if err := h.validate.Struct(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "VALIDATION_ERROR",
			Message: validationMessage(err),
		})
	}

	result, err := h.svc.LoginWithGoogle(c.Request().Context(), req.IDToken)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidIDToken):
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "INVALID_TOKEN", Message: "Google IDトークンが無効です"})
		case errors.Is(err, service.ErrAccountSuspended):
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "ACCOUNT_SUSPENDED", Message: "このアカウントは停止されています"})
		default:
			return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "サーバーエラーが発生しました"})
		}
	}

	setRefreshTokenCookie(c, result.RefreshToken)

	return c.JSON(http.StatusOK, dto.AuthResponse{
		AccessToken: result.AccessToken,
		User:        result.User,
	})
}

func setRefreshTokenCookie(c echo.Context, token string) {
	c.SetCookie(&http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int((7 * 24 * time.Hour).Seconds()),
	})
}
