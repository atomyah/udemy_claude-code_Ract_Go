package handler

import (
	"errors"
	"net/http"

	"github.com/atyahara/sns-backend/internal/authtoken"
	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/service"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// AuthCookieConfig は認証 Cookie の属性（環境ごとに config から決める）
type AuthCookieConfig struct {
	Secure   bool
	SameSite http.SameSite
}

// AuthHandler は認証関連のHTTPハンドラー
type AuthHandler struct {
	svc      service.AuthService
	cookies  AuthCookieConfig
	validate *validator.Validate
}

// NewAuthHandler はAuthHandlerを生成する
func NewAuthHandler(svc service.AuthService, cookies AuthCookieConfig) *AuthHandler {
	return &AuthHandler{svc: svc, cookies: cookies, validate: validator.New()}
}

// Register godoc
// @Summary      新規ユーザー登録
// @Description  メールアドレス・パスワード・ハンドル・表示名でユーザーを作成し、Access Token・Refresh Token を HttpOnly Cookie で返す
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
		case errors.Is(err, service.ErrPasswordTooLong):
			return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: passwordTooLongMessage})
		default:
			return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "サーバーエラーが発生しました"})
		}
	}

	h.setAuthCookies(c, result)
	return c.JSON(http.StatusCreated, dto.AuthResponse{User: result.User})
}

// Login godoc
// @Summary      ログイン
// @Description  メールアドレスとパスワードで認証し、Access Token・Refresh Token を HttpOnly Cookie（Set-Cookie）で返す
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

	h.setAuthCookies(c, result)
	return c.JSON(http.StatusOK, dto.AuthResponse{User: result.User})
}

// Logout godoc
// @Summary      ログアウト
// @Description  Refresh Token（Cookie）を DB 上で失効させ、access_token / refresh_token Cookie を削除する
// @Tags         auth
// @Produce      json
// @Success      204  "ログアウト成功"
// @Failure      500  {object} dto.ErrorResponse  "サーバーエラー"
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c echo.Context) error {
	var refreshToken string
	if cookie, err := c.Cookie(authtoken.RefreshTokenCookie); err == nil {
		refreshToken = cookie.Value
	}

	if err := h.svc.Logout(c.Request().Context(), refreshToken); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "サーバーエラーが発生しました"})
	}

	h.clearAuthCookies(c)
	return c.NoContent(http.StatusNoContent)
}

// RevokeAll godoc
// @Summary      全デバイスからログアウト
// @Description  ログイン中ユーザーの全 Refresh Token を DB 上で失効させ、このブラウザの Cookie も削除する。他の端末は Access Token の期限（最長1時間）が切れた時点でログアウトされる
// @Tags         auth
// @Produce      json
// @Success      204  "失効成功"
// @Failure      401  {object} dto.ErrorResponse  "未認証"
// @Failure      500  {object} dto.ErrorResponse  "サーバーエラー"
// @Router       /auth/revoke-all [post]
// @Security     BearerAuth
func (h *AuthHandler) RevokeAll(c echo.Context) error {
	userID, ok := c.Get("userID").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "MISSING_TOKEN", Message: "認証が必要です"})
	}

	if err := h.svc.RevokeAll(c.Request().Context(), userID); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "サーバーエラーが発生しました"})
	}

	h.clearAuthCookies(c)
	return c.NoContent(http.StatusNoContent)
}

// Refresh godoc
// @Summary      トークンの再発行
// @Description  refresh_token Cookie を検証し、新しい Access Token と Refresh Token を Cookie で返す（古い Refresh Token は使用済みになる）
// @Tags         auth
// @Produce      json
// @Success      200  {object} dto.AuthResponse   "再発行成功"
// @Failure      401  {object} dto.ErrorResponse  "Refresh Tokenが無効・失効済み・期限切れ"
// @Failure      500  {object} dto.ErrorResponse  "サーバーエラー"
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c echo.Context) error {
	cookie, err := c.Cookie(authtoken.RefreshTokenCookie)
	if err != nil || cookie.Value == "" {
		return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Code:    "MISSING_TOKEN",
			Message: "Refresh Tokenが見つかりません",
		})
	}

	result, err := h.svc.Refresh(c.Request().Context(), cookie.Value)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrExpiredToken):
			h.clearAuthCookies(c)
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "EXPIRED_TOKEN", Message: "ログインの有効期限が切れました。再度ログインしてください"})
		case errors.Is(err, service.ErrAccountSuspended):
			h.clearAuthCookies(c)
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "ACCOUNT_SUSPENDED", Message: "このアカウントは停止されています"})
		case errors.Is(err, service.ErrInvalidToken), errors.Is(err, service.ErrRevokedToken), errors.Is(err, service.ErrTokenReused):
			h.clearAuthCookies(c)
			return c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: "INVALID_TOKEN", Message: "ログイン情報が無効です。再度ログインしてください"})
		default:
			return c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: "INTERNAL_ERROR", Message: "サーバーエラーが発生しました"})
		}
	}

	h.setAuthCookies(c, result)
	return c.JSON(http.StatusOK, dto.AuthResponse{User: result.User})
}

// GoogleLogin godoc
// @Summary      Google OAuthログイン
// @Description  Firebase AuthのIDトークンを検証し、初回は自動登録、Access Token・Refresh Token を HttpOnly Cookie で返す
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

	h.setAuthCookies(c, result)
	return c.JSON(http.StatusOK, dto.AuthResponse{User: result.User})
}

// setAuthCookies はアクセストークン・リフレッシュトークンを HttpOnly Cookie にセットする
func (h *AuthHandler) setAuthCookies(c echo.Context, result *service.AuthResult) {
	c.SetCookie(h.newCookie(authtoken.AccessTokenCookie, result.AccessToken, authtoken.AccessTokenCookiePath, int(authtoken.AccessTokenTTL.Seconds())))
	c.SetCookie(h.newCookie(authtoken.RefreshTokenCookie, result.RefreshToken, authtoken.RefreshTokenCookiePath, int(authtoken.RefreshTokenTTL.Seconds())))
}

// clearAuthCookies は認証 Cookie を削除する（同じ名前・Path・属性で Max-Age=0 を返す）
func (h *AuthHandler) clearAuthCookies(c echo.Context) {
	c.SetCookie(h.newCookie(authtoken.AccessTokenCookie, "", authtoken.AccessTokenCookiePath, -1))
	c.SetCookie(h.newCookie(authtoken.RefreshTokenCookie, "", authtoken.RefreshTokenCookiePath, -1))
}

func (h *AuthHandler) newCookie(name, value, path string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.cookies.Secure,
		SameSite: h.cookies.SameSite,
	}
}
