// Package adminui は管理画面の HTML テンプレート（html/template）と静的ファイルを提供する。
// テンプレートはバイナリに埋め込む（embed）ため、本番イメージに別途ファイルを置く必要はない。
package adminui

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"time"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

// StaticFS は管理画面の静的ファイル（CSS）。/admin/static/ で配信する
func StaticFS() fs.FS {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	return sub
}

// jst は管理画面の日時表示に使うタイムゾーン（コンテナに tzdata がなくても使えるよう固定オフセットにする）
var jst = time.FixedZone("JST", 9*60*60)

var funcs = template.FuncMap{
	// formatTime は日時を日本時間の「2006-01-02 15:04」で表示する
	"formatTime": func(v interface{}) string {
		switch t := v.(type) {
		case time.Time:
			if t.IsZero() {
				return "-"
			}
			return t.In(jst).Format("2006-01-02 15:04")
		case *time.Time:
			if t == nil || t.IsZero() {
				return "-"
			}
			return t.In(jst).Format("2006-01-02 15:04")
		default:
			return "-"
		}
	},
	"formatDay": func(t time.Time) string { return t.Format("1/2") },
	"deref": func(s *string) string {
		if s == nil {
			return "-"
		}
		return *s
	},
	"userStatusLabel": func(status string) string {
		switch status {
		case "pending":
			return "承認待ち"
		case "approved":
			return "承認済み"
		case "rejected":
			return "拒否"
		default:
			return status
		}
	},
	"resetStatusLabel": func(status string) string {
		switch status {
		case "pending":
			return "未対応"
		case "issued":
			return "リンク発行済み"
		case "completed":
			return "再設定完了"
		default:
			return status
		}
	},
	"number": func(n int64) string { return formatNumber(n) },
}

// pages は描画できるページ名（templates/<name>.html）。各ページは layout.html と組み合わせて描画する
var pages = []string{
	"login", "dashboard", "users", "user_detail", "password_resets", "reset_link", "admins", "error",
}

// Renderer は管理画面のページを描画する
type Renderer struct {
	templates map[string]*template.Template
}

// NewRenderer は全ページのテンプレートを読み込む（起動時に 1 回だけ。構文エラーは起動時に検出される）
func NewRenderer() (*Renderer, error) {
	r := &Renderer{templates: make(map[string]*template.Template, len(pages))}
	for _, name := range pages {
		t, err := template.New("layout.html").Funcs(funcs).ParseFS(templateFS, "templates/layout.html", "templates/"+name+".html")
		if err != nil {
			return nil, fmt.Errorf("parse admin template %s: %w", name, err)
		}
		r.templates[name] = t
	}
	return r, nil
}

// Render はページを描画する
func (r *Renderer) Render(w io.Writer, page string, data interface{}) error {
	t, ok := r.templates[page]
	if !ok {
		return fmt.Errorf("admin template %s not found", page)
	}
	return t.ExecuteTemplate(w, "layout.html", data)
}

func formatNumber(n int64) string {
	s := fmt.Sprintf("%d", n)
	if n < 0 {
		return s
	}
	out := make([]byte, 0, len(s)+len(s)/3)
	for i := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	return string(out)
}
