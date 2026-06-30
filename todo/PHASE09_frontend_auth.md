# PHASE 09 — フロントエンド 認証画面

> 目標: ログイン・サインアップ画面と Google OAuth 連携を実装する。

---

## API フック（`src/features/auth/hooks/`）

- [ ] `useLogin.ts` — `useMutation` でログイン API を呼ぶ
- [ ] `useRegister.ts` — `useMutation` で新規登録 API を呼ぶ
- [ ] `useGoogleLogin.ts` — Firebase Auth で Google ポップアップ → idToken 取得 → バックエンドに送信
- [ ] `useLogout.ts` — ログアウト API → キャッシュクリア → `/login` へリダイレクト
- [ ] `useCurrentUser.ts` — `useQuery` で `GET /api/v1/users/me` を取得

## サインアップ画面（`src/pages/SignupPage.tsx`）

- [ ] メールアドレス入力
- [ ] パスワード入力（最小 8 文字、確認用フィールドあり）
- [ ] 表示名入力
- [ ] ユーザーハンドル（@handle）入力
- [ ] バリデーション（クライアントサイド）
  - メール形式チェック
  - パスワード強度チェック
  - handle: 英数字・アンダースコアのみ（正規表現）
- [ ] エラーメッセージ表示（API エラーを MUI の `Alert` で表示）
- [ ] 「Google で登録」ボタン
- [ ] 「すでにアカウントをお持ちの方はログイン」リンク

## ログイン画面（`src/pages/LoginPage.tsx`）

- [ ] メールアドレス入力
- [ ] パスワード入力（表示/非表示切り替えアイコン）
- [ ] 「ログイン」ボタン（送信中はローディングスピナー）
- [ ] 「Google でログイン」ボタン（Firebase Auth ポップアップ）
- [ ] 「アカウントをお持ちでない方は新規登録」リンク
- [ ] ログイン成功 → `/` にリダイレクト

## Google OAuth 連携（Firebase Auth）

- [ ] `firebase.ts` で Firebase App を初期化（`VITE_FIREBASE_API_KEY` 等の環境変数）
- [ ] `signInWithPopup(auth, new GoogleAuthProvider())` で ID Token を取得
- [ ] ID Token をバックエンド `POST /api/v1/auth/google` に送信
- [ ] バックエンドから返ってきた JWT を保存

## JWT 保存・管理

- [ ] Access Token: `localStorage`（`access_token` キー）
- [ ] Refresh Token: HttpOnly Cookie（バックエンドがセット）
- [ ] ログアウト時に `localStorage` をクリア

## フォームライブラリ選択

- [ ] `react-hook-form` + MUI の TextField で実装（推奨）
  - バリデーションは `react-hook-form` の `validate` または `zod` で行う

## 完了基準

- [ ] サインアップで新規ユーザーが作成されてログイン状態になる
- [ ] ログイン → JWT が `localStorage` に保存される
- [ ] Google ログインが動作する
- [ ] 不正な入力でクライアントサイドエラーが表示される
- [ ] メール重複・パスワード不一致などの API エラーが画面に表示される
- [ ] ログイン後に `/` にリダイレクトされる
- [ ] ログアウトで `/login` に戻る
