# udemy_claude-code_Ract_Go

## 管理画面（運営者向け）

API サーバーの `/admin`（ローカルは `http://localhost:8080/admin`）で、ユーザーの登録審査・パスワードリセットリンクの発行・管理者ユーザーの管理・ダッシュボードを利用できます。
新規登録したユーザーは管理者が承認するまで利用できません（`pending`）。

利用するには `backend/.env` に次の環境変数を設定して API を再起動します（`docker compose up -d --force-recreate api`）。

| 変数 | 説明 |
|------|------|
| `ADMIN_SESSION_SECRET` | 管理画面セッションの署名鍵（32 文字以上・JWT の鍵と別）。空なら管理画面は無効 |
| `ADMIN_ROOT_USERNAME` / `ADMIN_ROOT_PASSWORD` | ルート管理者のログイン情報（DB に保存しない。パスワード 12 文字以上） |
| `FRONTEND_BASE_URL` | パスワードリセットリンクに使うフロントエンドの URL（既定: `CORS_ORIGINS` の先頭） |
| `PASSWORD_RESET_TOKEN_TTL_MINUTES` | リセットリンクの有効期限（分、既定 1440） |
| `AUTO_APPROVE_USERS` | **E2E テスト専用**。`ENV=test` のときだけ新規登録を自動承認する（それ以外の環境では無視） |

URL 一覧・運用手順・監査ログ・セキュリティ上の注意は [docs/admin-panel.md](docs/admin-panel.md) を参照してください。
