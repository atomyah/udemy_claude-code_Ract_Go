# Git 規約

## ブランチ戦略

```
main          本番リリース済みのコード。直接コミット禁止。
develop       開発統合ブランチ。PR 経由でのみ更新。
feature/xxx   機能追加
fix/xxx       バグ修正
chore/xxx     設定・依存関係・ドキュメントなどコード外の変更
refactor/xxx  リファクタリング
```

### ブランチ命名規則

```
feature/add-post-like
fix/jwt-refresh-token-expiry
chore/update-go-dependencies
refactor/post-service-layer
```

- 小文字ハイフン区切り（ケバブケース）
- 英語で内容が分かる名前を付ける

---

## コミットメッセージ規約

**Conventional Commits** に準拠する。

```
<type>(<scope>): <summary>

[optional body]
```

### type 一覧

| type | 用途 |
|------|------|
| `feat` | 新機能の追加 |
| `fix` | バグ修正 |
| `chore` | ビルド・設定・依存関係の変更 |
| `refactor` | 機能変更を伴わないコード改善 |
| `test` | テストの追加・修正 |
| `docs` | ドキュメントのみの変更 |
| `style` | フォーマット・セミコロン等（ロジック変更なし） |
| `perf` | パフォーマンス改善 |

### scope の例

`auth`, `posts`, `users`, `follows`, `notifications`, `search`, `docker`, `db`

### 例

```
feat(posts): add cursor-based pagination to timeline

fix(auth): refresh token not cleared on logout

chore(deps): bump echo from v4.11 to v4.12

test(posts): add integration test for create post endpoint
```

### ルール

- summary は現在形・命令形の英語（`Add`, `Fix`, `Remove`）
- 50 文字以内
- 末尾にピリオドを付けない
- 日本語コミットメッセージは禁止

---

## .gitignore

必ず無視するもの:

```gitignore
# 環境変数
.env
.env.local
.env.*.local
backend/.env

# Go ビルド成果物
backend/tmp/
backend/server

# Swagger 生成ファイル（CI で生成する場合）
# backend/docs/

# フロントエンド
frontend/node_modules/
frontend/dist/

# OpenAPI 生成型
frontend/src/api/generated/

# Firebase 認証情報
*.serviceAccountKey.json
firebase-credentials.json
service-account.json

# IDE
.idea/
.vscode/
*.swp

# OS
.DS_Store
Thumbs.db

# ログ
*.log
```

---

## PR（プルリクエスト）規約

- ベースブランチは `develop`（`main` に直接 PR は出さない）
- 1 PR = 1 機能 or 1 修正
- PR タイトルはコミットメッセージと同じ規約（`feat(posts): ...`）
- PR 本文に変更概要・動作確認手順を書く
- セルフレビューしてから PR を出す

---

## 禁止事項

- `main` / `develop` に直接コミットする。
- 機能ブランチ上で `git push --force`（`--force-with-lease` は可）。
- `.env` や認証情報ファイルをコミットする。
- コミットメッセージを `fix`, `update`, `wip` だけで終わらせる（何を fix/update したか必ず書く）。
