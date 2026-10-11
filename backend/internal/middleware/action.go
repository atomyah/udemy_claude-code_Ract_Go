package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/labstack/echo/v4"
)

// ActionCatalog は「HTTPメソッド + ルート」（例: "POST /api/v1/posts/:id/like"）から
// 利用者の操作の日本語名（例: "いいね"）を引く表。アクセスログの action / description に使う。
type ActionCatalog map[string]string

// 操作名が見つからないときの表記
const (
	ActionUnknownRoute = "存在しないAPIへのアクセス"
	ActionUnknown      = "不明な操作"
)

// swagger に載らないルートの操作名
var builtinActions = ActionCatalog{
	"GET /health":    "ヘルスチェック",
	"GET /swagger/*": "API仕様書（Swagger UI）の表示",
}

var swaggerPathParam = regexp.MustCompile(`\{([^}]+)\}`)

// ActionsFromSwagger は swag が生成した Swagger 定義（JSON）の各エンドポイントの @Summary を操作名として読み込む。
// 全ハンドラーに @Summary を書くルールなので、新しい API を追加しても表を手で更新する必要はない。
// basePath（例: "/api/v1"）を付け、{id} を Echo 形式の :id に変換したルートをキーにする。
func ActionsFromSwagger(doc []byte, basePath string) (ActionCatalog, error) {
	var spec struct {
		Paths map[string]map[string]struct {
			Summary string `json:"summary"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(doc, &spec); err != nil {
		return nil, fmt.Errorf("parse swagger: %w", err)
	}

	actions := make(ActionCatalog, len(spec.Paths)+len(builtinActions))
	for k, v := range builtinActions {
		actions[k] = v
	}
	for path, ops := range spec.Paths {
		route := basePath + swaggerPathParam.ReplaceAllString(path, ":$1")
		for method, op := range ops {
			if op.Summary != "" {
				actions[strings.ToUpper(method)+" "+route] = op.Summary
			}
		}
	}
	return actions, nil
}

// Action はリクエストの操作名を返す
func (a ActionCatalog) Action(method, route string, status int) string {
	if method == http.MethodOptions {
		return "CORS の事前確認（プリフライト）"
	}
	if name, ok := a[method+" "+route]; ok {
		return name
	}
	if status == http.StatusNotFound || status == http.StatusMethodNotAllowed {
		return ActionUnknownRoute
	}
	return ActionUnknown
}

// describeRequest は「誰が・何を・どの対象に・結果どうなったか」を日本語の 1 文にする。
// 例: ログインユーザーが「いいね」（id=8a16...）をリクエストし、成功しました（201）
func describeRequest(c echo.Context, action string, status int, authenticated bool) string {
	actor := "未ログインのユーザー"
	if authenticated {
		actor = "ログインユーザー"
	}

	target := ""
	if names := c.ParamNames(); len(names) > 0 {
		values := c.ParamValues()
		pairs := make([]string, 0, len(names))
		for i, name := range names {
			if i < len(values) && name != "*" {
				pairs = append(pairs, name+"="+values[i])
			}
		}
		if len(pairs) > 0 {
			target = "（" + strings.Join(pairs, ", ") + "）"
		}
	}

	return fmt.Sprintf("%sが「%s」%sをリクエストし、%s（%d）", actor, action, target, describeResult(status), status)
}

// describeResult はステータスコードを利用者の操作結果として説明する
func describeResult(status int) string {
	switch {
	case status < 400:
		return "成功しました"
	case status == http.StatusBadRequest:
		return "入力内容に誤りがあり失敗しました"
	case status == http.StatusUnauthorized:
		return "ログインしていない（認証切れ）ため失敗しました"
	case status == http.StatusForbidden:
		return "権限がないため拒否されました"
	case status == http.StatusNotFound:
		return "対象が見つからず失敗しました"
	case status == http.StatusMethodNotAllowed:
		return "許可されていない操作のため失敗しました"
	case status == http.StatusConflict:
		return "すでに実行済み（重複）のため失敗しました"
	case status == http.StatusRequestEntityTooLarge:
		return "サイズが大きすぎるため失敗しました"
	case status == http.StatusUnprocessableEntity:
		return "処理できない内容のため失敗しました"
	case status == http.StatusTooManyRequests:
		return "回数制限を超えたため拒否されました"
	case status >= 500:
		return "サーバーエラーで失敗しました"
	default:
		return "失敗しました"
	}
}
