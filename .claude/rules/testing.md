# テスト規約

## 方針

- バックエンド: サービス層のユニットテスト + ハンドラー層の統合テストを書く。
- フロントエンド: コンポーネントの表示テスト + カスタムフックのテストを書く。
- E2E テストはオプション（PHASE14 で対応）。
- テストなしでの `main` / `develop` へのマージは禁止。

---

## バックエンド テスト

### ツール

| 用途 | ライブラリ |
|------|-----------|
| テストフレームワーク | 標準 `testing` パッケージ |
| アサーション | `github.com/stretchr/testify/assert` |
| モック | `github.com/stretchr/testify/mock` |
| HTTP テスト | `net/http/httptest` + Echo の `echo.New()` |
| DB 統合テスト | テスト用 PostgreSQL（Docker）を使う |

### ディレクトリ構成

```
backend/internal/
├── service/
│   ├── post_service.go
│   └── post_service_test.go    # サービスのユニットテスト
├── handler/
│   ├── post_handler.go
│   └── post_handler_test.go    # ハンドラーの統合テスト
└── repository/
    ├── post_repository.go
    └── post_repository_test.go # リポジトリの統合テスト（DB 使用）
```

テストファイルはテスト対象ファイルと同じディレクトリに置く。

### ユニットテスト（サービス層）

- リポジトリをモックして、サービスのビジネスロジックだけをテストする。
- `testify/mock` でリポジトリインターフェースをモック実装する。

```go
func TestPostService_CreatePost_Success(t *testing.T) {
    mockRepo := new(mocks.PostRepository)
    mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Post")).
        Return(&model.Post{ID: uuid.New(), Content: "test"}, nil)

    svc := NewPostService(mockRepo)
    result, err := svc.CreatePost(context.Background(), userID, req, nil)

    assert.NoError(t, err)
    assert.Equal(t, "test", result.Content)
    mockRepo.AssertExpectations(t)
}
```

### 統合テスト（ハンドラー層）

- `httptest.NewRecorder` と Echo を使って HTTP リクエストをシミュレートする。
- `testify/assert` でステータスコードとレスポンスボディを検証する。

```go
func TestPostHandler_CreatePost_Unauthorized(t *testing.T) {
    e := echo.New()
    req := httptest.NewRequest(http.MethodPost, "/api/v1/posts", nil)
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)
    // JWT ミドルウェアなしで直接ハンドラーを呼ぶ

    handler := NewPostHandler(mockService)
    err := handler.CreatePost(c)

    assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
```

### テスト命名規則

```
Test{型名}_{メソッド名}_{状況}
```

例:
- `TestPostService_CreatePost_Success`
- `TestPostService_CreatePost_ContentTooLong`
- `TestAuthHandler_Login_InvalidPassword`

### テスト実行

```bash
# 全テスト実行
cd backend && go test ./...

# カバレッジ確認
go test ./... -cover

# 特定パッケージのみ
go test ./internal/service/...
```

---

## フロントエンド テスト

### ツール

| 用途 | ライブラリ |
|------|-----------|
| テストランナー | Vitest |
| コンポーネントテスト | `@testing-library/react` |
| DOM マッチャー | `@testing-library/jest-dom` |
| API モック | `msw`（Mock Service Worker） |
| フックテスト | `@testing-library/react` の `renderHook` |

### ディレクトリ構成

```
frontend/src/
├── components/
│   └── PostCard/
│       ├── PostCard.tsx
│       └── PostCard.test.tsx   # コンポーネントテスト
├── features/
│   └── posts/
│       └── hooks/
│           ├── useLike.ts
│           └── useLike.test.ts # フックテスト
└── mocks/
    ├── handlers.ts             # MSW ハンドラー定義
    └── server.ts               # MSW サーバー設定
```

### コンポーネントテスト

- ユーザーから見える振る舞いをテストする（実装の詳細はテストしない）。
- `render`, `screen`, `userEvent` を使う。

```tsx
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { PostCard } from './PostCard';

test('いいねボタンを押すといいね数が増える', async () => {
  const user = userEvent.setup();
  render(<PostCard post={mockPost} />);

  const likeButton = screen.getByRole('button', { name: /いいね/i });
  await user.click(likeButton);

  expect(screen.getByText('1')).toBeInTheDocument();
});
```

### API モック（MSW）

- テスト中の API リクエストは MSW でインターセプトする。実際のバックエンドは使わない。
- モックハンドラーは `src/mocks/handlers.ts` に集約する。

```ts
// src/mocks/handlers.ts
import { http, HttpResponse } from 'msw';

export const handlers = [
  http.get('/api/v1/posts/home', () => {
    return HttpResponse.json({ data: [mockPost], nextCursor: null, hasMore: false });
  }),
];
```

### テスト命名規則

```
test('状況 + 期待する振る舞い', ...)
```

例:
- `test('未ログインのとき投稿フォームが表示されない')`
- `test('いいねボタンを押すとカウントが増える')`
- `test('280文字を超えると投稿ボタンが disabled になる')`

### テスト実行

```bash
cd frontend

# 全テスト実行
npm run test

# ウォッチモード
npm run test -- --watch

# カバレッジ
npm run test -- --coverage
```

---

## テストすべき優先項目

### バックエンド（優先度 高）

- [ ] `AuthService`: Register・Login のバリデーション・重複チェック
- [ ] `PostService`: 作成・編集・削除の権限チェック・文字数制限
- [ ] `LikeService`: 重複いいね防止・自分の投稿への通知スキップ
- [ ] `JWTMiddleware`: 有効トークン / 無効トークン / 期限切れ

### フロントエンド（優先度 高）

- [ ] `PostCard`: いいね・ブックマーク・削除ボタンの表示制御
- [ ] `PostForm`: 280 文字制限・送信ボタン disabled 制御
- [ ] `PrivateRoute`: 未ログイン時のリダイレクト
- [ ] `AuthContext`: ログイン・ログアウト後の状態変化

---

## 禁止事項

- テストで本番 DB・本番 Firebase に接続する。
- `console.log` を使ったデバッグをテストにコミットする。
- `any` 型でモックオブジェクトを作る。
- テストをスキップ（`t.Skip()` / `test.skip()`）したままコミットする（原則禁止）。
