# バックエンド コーディング規約（Go / Echo / GORM）

## ディレクトリ構成

```
backend/
├── cmd/server/
│   └── main.go            # エントリーポイント
├── internal/
│   ├── handler/           # Echo ハンドラー（HTTP 層のみ。ビジネスロジックなし）
│   ├── service/           # ビジネスロジック層
│   ├── repository/        # GORM を使った DB アクセス層
│   ├── model/             # GORM モデル（DB テーブル対応構造体）
│   ├── dto/               # リクエスト / レスポンス用構造体
│   ├── middleware/        # Echo ミドルウェア（JWT 検証など）
│   └── config/            # 設定読み込み（環境変数）
├── docs/                  # swaggo が生成する OpenAPI ファイル（手動編集禁止）
├── .air.toml              # air ホットリロード設定
├── Dockerfile
└── go.mod
```

---

## 開発環境

### ホットリロード（air）

**開発時は必ず air を使う。`go run` による毎回フルコンパイルは禁止。**

```toml
# .air.toml の最低限設定例
[build]
  cmd = "go build -o ./tmp/main ./cmd/server"
  bin = "./tmp/main"
  include_ext = ["go"]
  exclude_dir = ["docs", "tmp", "vendor"]
```

起動コマンド:
```bash
air
```

---

## レイヤー責務

| レイヤー | 責務 | 禁止事項 |
|---------|------|---------|
| handler | HTTP リクエスト受信・バリデーション・レスポンス返却 | DB 直接アクセス、ビジネスロジック |
| service | ビジネスロジック・複数リポジトリの組み合わせ | HTTP 知識（echo.Context など） |
| repository | GORM を使った DB CRUD | ビジネスロジック |

---

## コーディング規約

### 命名

- パッケージ名: 小文字単語（`handler`, `service`, `repository`）
- 構造体・インターフェース: パスカルケース（`PostHandler`, `PostService`）
- インターフェースは `PostService`、実装は `postService`（非公開）
- 関数: パスカルケース（公開）/ キャメルケース（非公開）
- 定数: 大文字スネークケース（`MAX_POST_LENGTH`）

### エラーハンドリング

- エラーは必ず呼び出し元に返す。`_` で無視しない。
- ハンドラー層でエラーを HTTP レスポンスに変換する。
- カスタムエラー型で業務エラー（NotFound, Forbidden など）を表現する。

```go
// カスタムエラー例
type AppError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Status  int    `json:"-"`
}
```

- 全エラーレスポンスは `{ "code": "...", "message": "..." }` 形式で返す。

### 依存性注入

- コンストラクタ関数（`NewXxx`）で依存を注入する。グローバル変数は使わない。
- インターフェースを通じて依存する（テスタビリティのため）。

```go
type PostRepository interface {
    FindByID(ctx context.Context, id uuid.UUID) (*model.Post, error)
    // ...
}

type postService struct {
    repo PostRepository
}

func NewPostService(repo PostRepository) *postService {
    return &postService{repo: repo}
}
```

---

## JWT 認証

- Access Token: 有効期限 15 分、`Authorization: Bearer <token>` ヘッダーで受け取る。
- Refresh Token: 有効期限 7 日、HttpOnly Cookie で送受信する。
- JWT ミドルウェアを `middleware/auth.go` に実装し、保護ルートに適用する。
- `echo.Context` のカスタムキーでユーザー ID を取り出せるようにする。

```go
// ミドルウェア使用例
restricted := e.Group("/api/v1")
restricted.Use(middleware.JWTAuth(cfg.JWTSecret))
```

---

## GORM 規約

- モデルは `internal/model/` に置く。`gorm.Model`（ID, CreatedAt, UpdatedAt, DeletedAt）は使わず、UUID と明示的な型を定義する。
- UUID は `github.com/google/uuid` を使い、DB は `uuid` 型で保存する。
- 論理削除は `is_deleted bool` フィールドで行う。GORM の `SoftDelete` は使わない。
- マイグレーションは `AutoMigrate` を本番で使わない。マイグレーションファイルを管理する（`golang-migrate` 推奨）。

```go
// モデル例
type Post struct {
    ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
    UserID    uuid.UUID `gorm:"type:uuid;not null"`
    Content   string    `gorm:"size:280;not null"`
    IsEdited  bool      `gorm:"default:false"`
    IsDeleted bool      `gorm:"default:false"`
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

---

## Swagger（swaggo）規約

- 全ハンドラーに swaggo コメントを書く。コメントなしのエンドポイントは禁止。
- `// @Summary`, `// @Tags`, `// @Accept`, `// @Produce`, `// @Param`, `// @Success`, `// @Failure`, `// @Router` を必ず記載する。

```go
// CreatePost godoc
// @Summary      投稿を作成する
// @Tags         posts
// @Accept       multipart/form-data
// @Produce      json
// @Param        content  formData  string  true  "投稿内容（最大280文字）"
// @Success      201  {object}  dto.PostResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Router       /posts [post]
// @Security     BearerAuth
func (h *PostHandler) CreatePost(c echo.Context) error {
```

- `docs/` ディレクトリは `swag init` で自動生成する。手動編集禁止。

---

## ファイルアップロード

- `multipart/form-data` でファイルを受け取り、バックエンドが Firebase Storage SDK でアップロードする。
- フロントから Firebase Storage に直接アップロードさせない（アクセス制御の一元化のため）。
- アップロード後に Storage の URL を DB に保存する。

---

## 環境変数

設定は全て環境変数で管理する。`config/config.go` で `os.Getenv` または `github.com/caarlos0/env` で読み込む。

```
DATABASE_URL=postgres://...
JWT_SECRET=...
JWT_REFRESH_SECRET=...
FIREBASE_CREDENTIALS_JSON=...  # Firebase サービスアカウント JSON
PORT=8080
ENV=development  # development | production
```

`.env` ファイルはローカル開発用のみ。コミットしない（`.gitignore` に追加）。

---

## 禁止事項

- `go run` で毎回起動する（air を使う）。
- グローバル変数でのセッション / DB 接続管理。
- `panic` を業務エラーとして使う。
- DB 直接アクセスをハンドラー層で行う。
- `fmt.Println` / `log.Println` をコミットする（`slog` または `zerolog` を使う）。
