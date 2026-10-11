package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/atyahara/sns-backend/internal/model"
	"github.com/atyahara/sns-backend/internal/repository"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// UserFinder は AdminOnly がロールを確認するために使うユーザー取得インターフェース（repository.UserRepository が満たす）
type UserFinder interface {
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

// AdminOnly は JWTAuth の後に置き、ログインユーザーが admin ロールかを DB で確認する。
// アクセストークンにはロールを含めないため、ロール変更・停止は次のリクエストから反映される
func AdminOnly(users UserFinder) echo.MiddlewareFunc {
	forbidden := func(c echo.Context) error {
		return c.JSON(http.StatusForbidden, map[string]string{
			"code":    "FORBIDDEN",
			"message": "管理者権限が必要です",
		})
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID, ok := c.Get("userID").(uuid.UUID)
			if !ok {
				return forbidden(c)
			}

			user, err := users.FindByID(c.Request().Context(), userID)
			if errors.Is(err, repository.ErrNotFound) {
				return forbidden(c)
			}
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"code":    "INTERNAL_ERROR",
					"message": "サーバーエラーが発生しました",
				})
			}
			if user.Role != "admin" || user.IsSuspended {
				return forbidden(c)
			}
			return next(c)
		}
	}
}
