package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func AdminOnly() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role, _ := c.Get("userRole").(string)
			if role != "admin" {
				return c.JSON(http.StatusForbidden, map[string]string{
					"code":    "FORBIDDEN",
					"message": "管理者権限が必要です",
				})
			}
			return next(c)
		}
	}
}
