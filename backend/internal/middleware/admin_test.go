package middleware

import (
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminOnly_AdminRole_PassesThrough(t *testing.T) {
	c, rec := newContext("")
	c.Set("userRole", "admin")

	require.NoError(t, AdminOnly()(okHandler)(c))

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAdminOnly_UserRole_Returns403(t *testing.T) {
	c, rec := newContext("")
	c.Set("userRole", "user")

	require.NoError(t, AdminOnly()(okHandler)(c))

	assert.Equal(t, http.StatusForbidden, rec.Code)
	body := decodeError(t, rec)
	assert.Equal(t, "FORBIDDEN", body["code"])
	assert.Equal(t, "管理者権限が必要です", body["message"])
}

func TestAdminOnly_NoRole_Returns403(t *testing.T) {
	c, rec := newContext("")

	require.NoError(t, AdminOnly()(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})(c))

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Equal(t, "FORBIDDEN", decodeError(t, rec)["code"])
}
