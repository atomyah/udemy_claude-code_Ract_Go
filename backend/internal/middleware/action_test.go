package middleware

import (
	"net/http"
	"testing"

	"github.com/atyahara/sns-backend/docs"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testActions はアクセスログのテストで使う操作名の表
var testActions = ActionCatalog{
	"GET /api/v1/posts/:id":       "投稿詳細取得",
	"POST /api/v1/posts/:id/like": "いいね",
	"GET /ok":                     "テスト用の操作",
}

func TestActionsFromSwagger_ConvertsPathsAndSummaries(t *testing.T) {
	doc := []byte(`{"paths":{
		"/posts/{id}/like":{"post":{"summary":"いいね"},"delete":{"summary":"いいね取消"}},
		"/users/{handle}/follow":{"post":{"summary":"ユーザーをフォロー"}},
		"/no-summary":{"get":{}}
	}}`)

	actions, err := ActionsFromSwagger(doc, "/api/v1")

	require.NoError(t, err)
	assert.Equal(t, "いいね", actions["POST /api/v1/posts/:id/like"])
	assert.Equal(t, "いいね取消", actions["DELETE /api/v1/posts/:id/like"])
	assert.Equal(t, "ユーザーをフォロー", actions["POST /api/v1/users/:handle/follow"])
	assert.NotContains(t, actions, "GET /api/v1/no-summary", "summary の無い API は登録しない")
	assert.Equal(t, "ヘルスチェック", actions["GET /health"], "swagger に無い組み込みルートも含む")
}

func TestActionsFromSwagger_InvalidJSON(t *testing.T) {
	_, err := ActionsFromSwagger([]byte("not json"), "/api/v1")
	assert.Error(t, err)
}

// 実際の Swagger 定義（docs/）から、主要な API の操作名が引けること
func TestActionsFromSwagger_RealDocs(t *testing.T) {
	actions, err := ActionsFromSwagger([]byte(docs.SwaggerInfo.ReadDoc()), docs.SwaggerInfo.BasePath)
	require.NoError(t, err)

	cases := map[string]string{
		"POST /api/v1/auth/login":             "ログイン",
		"POST /api/v1/posts":                  "投稿作成",
		"GET /api/v1/posts/:id":               "投稿詳細取得",
		"POST /api/v1/posts/:id/like":         "いいね",
		"DELETE /api/v1/users/:handle/follow": "フォロー解除",
	}
	for route, want := range cases {
		assert.Equal(t, want, actions[route], route)
	}
}

func TestActionCatalog_Action(t *testing.T) {
	assert.Equal(t, "いいね", testActions.Action(http.MethodPost, "/api/v1/posts/:id/like", 201))
	assert.Equal(t, ActionUnknownRoute, testActions.Action(http.MethodGet, "/api/v1/no-such", 404))
	assert.Equal(t, ActionUnknown, testActions.Action(http.MethodGet, "/api/v1/unlisted", 200))
	assert.Equal(t, "CORS の事前確認（プリフライト）", testActions.Action(http.MethodOptions, "/api/v1/posts", 204))
}

func TestDescribeRequest(t *testing.T) {
	newCtx := func() echo.Context {
		c := echo.New().NewContext(nil, nil)
		c.SetParamNames("id")
		c.SetParamValues("abc")
		return c
	}

	cases := []struct {
		name          string
		status        int
		authenticated bool
		want          string
	}{
		{"成功", 201, true, "ログインユーザーが「いいね」（id=abc）をリクエストし、成功しました（201）"},
		{"未ログイン", 401, false, "未ログインのユーザーが「いいね」（id=abc）をリクエストし、ログインしていない（認証切れ）ため失敗しました（401）"},
		{"重複", 409, true, "ログインユーザーが「いいね」（id=abc）をリクエストし、すでに実行済み（重複）のため失敗しました（409）"},
		{"レート制限", 429, true, "ログインユーザーが「いいね」（id=abc）をリクエストし、回数制限を超えたため拒否されました（429）"},
		{"サーバーエラー", 500, true, "ログインユーザーが「いいね」（id=abc）をリクエストし、サーバーエラーで失敗しました（500）"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, describeRequest(newCtx(), "いいね", tc.status, tc.authenticated))
		})
	}
}

func TestAccessLog_OutputsJapaneseActionAndDescription(t *testing.T) {
	buf := captureLogs(t)
	userID := uuid.New()
	e := newLoggingServer()
	e.POST("/api/v1/posts/:id/like", func(c echo.Context) error {
		c.Set("userID", userID)
		return c.NoContent(http.StatusCreated)
	})

	serve(e, http.MethodPost, "/api/v1/posts/8a16/like")

	log := findLog(t, logLines(t, buf), "access")
	assert.Equal(t, "いいね", log["action"])
	assert.Equal(t, "ログインユーザーが「いいね」（id=8a16）をリクエストし、成功しました（201）", log["description"])
}

func TestAccessLog_UnknownRouteDescription(t *testing.T) {
	buf := captureLogs(t)
	e := newLoggingServer()

	serve(e, http.MethodGet, "/api/v1/no-such")

	log := findLog(t, logLines(t, buf), "access")
	assert.Equal(t, ActionUnknownRoute, log["action"])
	assert.Equal(t, "未ログインのユーザーが「存在しないAPIへのアクセス」をリクエストし、対象が見つからず失敗しました（404）", log["description"])
}
