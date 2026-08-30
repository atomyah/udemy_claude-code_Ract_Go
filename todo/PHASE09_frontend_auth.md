# PHASE 09 — フロントエンド 認証画面

> 目標: ログイン・サインアップ画面と Google OAuth 連携を実装する。

---

## API フック（`src/features/auth/hooks/`）

- [x] `useLogin.ts` — `useMutation` でログイン API を呼ぶ
- [x] `useRegister.ts` — `useMutation` で新規登録 API を呼ぶ
- [ ] `useGoogleLogin.ts` — Firebase Auth で Google ポップアップ → idToken 取得 → バックエンドに送信
  - **未対応**: バックエンドの `POST /auth/google` が未実装（501スタブ）で、フロント用の Firebase
    クライアント設定（`VITE_FIREBASE_API_KEY` 等）もリポジトリに無いため、今回はスコープ外とした
    （ユーザー確認済み）。ログイン/サインアップ画面には「Google でログイン/登録」ボタンを設置し、
    押下時は「実装予定です」という Snackbar を表示するのみにしている。
- [x] `useLogout.ts` — ログアウト API → キャッシュクリア → `/login` へリダイレクト
- [x] `useCurrentUser.ts` — `useQuery` で `GET /api/v1/users/me` を取得

## サインアップ画面（`src/pages/SignupPage.tsx`）

- [x] メールアドレス入力
- [x] パスワード入力（最小 8 文字、確認用フィールドあり）
- [x] 表示名入力
- [x] ユーザーハンドル（@handle）入力
- [x] バリデーション（クライアントサイド）
  - メール形式チェック
  - パスワード強度チェック（8〜72文字）
  - handle: 英数字・アンダースコアのみ（正規表現）
- [x] エラーメッセージ表示（API エラーを MUI の `Alert` で表示）
- [x] 「Google で登録」ボタン（未実装プレースホルダー、上記参照）
- [x] 「すでにアカウントをお持ちの方はログイン」リンク

## ログイン画面（`src/pages/LoginPage.tsx`）

- [x] メールアドレス入力
- [x] パスワード入力（表示/非表示切り替えアイコン）
- [x] 「ログイン」ボタン（送信中はローディングスピナー）
- [x] 「Google でログイン」ボタン（未実装プレースホルダー、上記参照）
- [x] 「アカウントをお持ちでない方は新規登録」リンク
- [x] ログイン成功 → `/`（`PrivateRoute` からのリダイレクト時は元のパス）にリダイレクト

## Google OAuth 連携（Firebase Auth）

- [ ] `firebase.ts` で Firebase App を初期化（`VITE_FIREBASE_API_KEY` 等の環境変数）
- [ ] `signInWithPopup(auth, new GoogleAuthProvider())` で ID Token を取得
- [ ] ID Token をバックエンド `POST /api/v1/auth/google` に送信
- [ ] バックエンドから返ってきた JWT を保存

（上記4項目はスコープ外。バックエンドの `POST /auth/google` 実装と Firebase クライアント設定が
揃った時点で改めて対応する）

## JWT 保存・管理

- [x] Access Token: `localStorage`（`access_token` キー）
- [x] Refresh Token: HttpOnly Cookie（バックエンドがセット）
- [x] ログアウト時に `localStorage` をクリア

## フォームライブラリ選択

- [x] `react-hook-form` + MUI の TextField で実装
  - バリデーションは `react-hook-form` の標準ルール（`required`/`pattern`/`minLength`/`validate`）で実装

## 完了基準

- [x] サインアップで新規ユーザーが作成されてログイン状態になる
- [x] ログイン → JWT が `localStorage` に保存される
- [ ] Google ログインが動作する（スコープ外、上記参照）
- [x] 不正な入力でクライアントサイドエラーが表示される
- [x] メール重複・パスワード不一致などの API エラーが画面に表示される
- [x] ログイン後に `/` にリダイレクトされる
- [x] ログアウトで `/login` に戻る

## 付随して対応したフロントエンドの不具合

- [x] `src/api/client.ts` のレスポンスインターセプター: `/auth/login` や `/auth/register` が
  401（メール重複以外の認証失敗等）を返した際に、誤って Refresh Token 再取得フローへ入り、
  本来の「メールアドレスまたはパスワードが正しくありません」ではなく
  「Refresh Token が見つかりません」という無関係なエラーメッセージを表示してしまう不具合を発見・修正。
  `/auth/refresh` に加えて `/auth/login`・`/auth/register` も再試行対象から除外した。
