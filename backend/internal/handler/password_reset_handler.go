package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/atyahara/sns-backend/internal/dto"
	"github.com/atyahara/sns-backend/internal/httperr"
	"github.com/atyahara/sns-backend/internal/service"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

// PasswordResetHandler は一般ユーザー向けのパスワードリセット（申請・再設定）のハンドラー
type PasswordResetHandler struct {
	svc      service.PasswordResetService
	validate *validator.Validate
}

// NewPasswordResetHandler は PasswordResetHandler を生成する
func NewPasswordResetHandler(svc service.PasswordResetService) *PasswordResetHandler {
	return &PasswordResetHandler{svc: svc, validate: validator.New()}
}

// RequestReset godoc
// @Summary      パスワードリセットの申請
// @Description  パスワードリセットを申請する。運営が申請内容を確認し、リセットリンクをメールで送付する（自動送信はしない）。登録の有無を推測されないよう、メールアドレスが未登録でも同じ 202 を返す
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body     dto.PasswordResetRequestRequest  true  "申請するアカウントのメールアドレス"
// @Success      202   {object} dto.SuccessResponse              "申請を受け付けた"
// @Failure      400   {object} dto.ErrorResponse                "バリデーションエラー"
// @Failure      429   {object} dto.ErrorResponse                "リクエストが多すぎる"
// @Failure      500   {object} dto.ErrorResponse                "サーバーエラー"
// @Router       /auth/password-reset-requests [post]
func (h *PasswordResetHandler) RequestReset(c echo.Context) error {
	var req dto.PasswordResetRequestRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "INVALID_REQUEST", Message: "リクエストの形式が不正です"})
	}
	req.Email = strings.TrimSpace(sanitizeText(req.Email))
	if err := h.validate.Struct(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: validationMessage(err)})
	}

	if err := h.svc.RequestReset(c.Request().Context(), req.Email); err != nil {
		return httperr.Internal(c, "パスワードリセットの申請に失敗しました", err)
	}
	return c.JSON(http.StatusAccepted, dto.SuccessResponse{
		Message: "申請を受け付けました。登録されているメールアドレスの場合、運営から再設定用のリンクをお送りします",
	})
}

// ResetPassword godoc
// @Summary      パスワードの再設定
// @Description  運営から送付されたリセットリンクのトークンで新しいパスワードを設定する。トークンは有効期限付き・一回限りで、再設定するとほかの端末のログインも無効になる
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body     dto.PasswordResetRequest  true  "リセット用トークンと新しいパスワード"
// @Success      200   {object} dto.SuccessResponse       "再設定成功"
// @Failure      400   {object} dto.ErrorResponse         "バリデーションエラー・トークンが無効または期限切れ（INVALID_RESET_TOKEN）"
// @Failure      429   {object} dto.ErrorResponse         "リクエストが多すぎる"
// @Failure      500   {object} dto.ErrorResponse         "サーバーエラー"
// @Router       /auth/password-reset [post]
func (h *PasswordResetHandler) ResetPassword(c echo.Context) error {
	var req dto.PasswordResetRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "INVALID_REQUEST", Message: "リクエストの形式が不正です"})
	}
	req.Token = strings.TrimSpace(req.Token)
	if err := h.validate.Struct(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: validationMessage(err)})
	}

	err := h.svc.ResetPassword(c.Request().Context(), req.Token, req.Password)
	switch {
	case errors.Is(err, service.ErrInvalidResetToken):
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "INVALID_RESET_TOKEN",
			Message: "リセット用のリンクが無効か、有効期限が切れています。もう一度申請してください",
		})
	case errors.Is(err, service.ErrPasswordTooLong):
		return c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: passwordTooLongMessage})
	case err != nil:
		return httperr.Internal(c, "パスワードの再設定に失敗しました", err)
	}
	return c.JSON(http.StatusOK, dto.SuccessResponse{Message: "パスワードを再設定しました。新しいパスワードでログインしてください"})
}
