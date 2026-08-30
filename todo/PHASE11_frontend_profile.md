# PHASE 11 — フロントエンド プロフィール・フォロー・設定

> 目標: プロフィールページ、プロフィール編集、フォロー/フォロワー一覧、設定画面を実装する。

---

## プロフィール API フック（`src/features/profile/hooks/`）

- [ ] `useProfile.ts` — `useQuery` で `GET /api/v1/users/:handle`
- [ ] `useUserPosts.ts` — `useInfiniteQuery` で `GET /api/v1/users/:handle/posts`
- [ ] `useFollowers.ts` — `useInfiniteQuery` で `GET /api/v1/users/:handle/followers`
- [ ] `useFollowing.ts` — `useInfiniteQuery` で `GET /api/v1/users/:handle/following`
- [ ] `useFollow.ts` — フォロー / アンフォローの `useMutation`（楽観的更新）
- [ ] `useUpdateProfile.ts` — `useMutation` で `PUT /api/v1/users/me`
- [ ] `useUploadAvatar.ts` — `useMutation` で `PUT /api/v1/users/me/avatar`（multipart）
- [ ] `useUploadBanner.ts` — `useMutation` で `PUT /api/v1/users/me/banner`（multipart）

> `features/profile/` ディレクトリ自体が未作成。バックエンド API（PHASE04）は完成済みのため、フックとページの実装のみが残っている。

## プロフィールページ（`src/pages/ProfilePage.tsx`）

- [ ] バナー画像（表示、自分のプロフィールでは編集ボタンを重ねる）
- [ ] アバター画像（バナー下に重なって表示）
- [ ] 表示名 / @handle
- [ ] bio・場所・ウェブサイト URL・誕生日（設定されている場合のみ表示）
- [ ] フォロワー数 / フォロー数（クリックで一覧モーダルまたは別ページへ）
- [ ] フォローボタン（自分以外、トグル式）
- [ ] 「プロフィールを編集」ボタン（自分のページのみ）
- [ ] タブ切り替え: 「投稿」「返信」「メディア」「いいね」
  - 投稿タブ: InfinitePostList

> 現状 `ProfilePage.tsx` は9行のプレースホルダーのみ。

## フォロー一覧コンポーネント（`src/features/profile/components/FollowList.tsx`）

- [ ] ユーザーカード（アバター・表示名・@handle・フォローボタン）
- [ ] 無限スクロール対応

## プロフィール編集ダイアログ（`src/features/profile/components/EditProfileDialog.tsx`）

- [ ] MUI `Dialog` で表示
- [ ] 表示名・bio（160 文字カウンター）・場所・ウェブサイット URL・誕生日
- [ ] アバター画像クリックで選択・プレビュー
- [ ] バナー画像クリックで選択・プレビュー
- [ ] 保存ボタン → プロフィール更新 → 画像があれば追加でアップロード

## 設定ページ（`src/pages/SettingsPage.tsx`）

### アカウント設定

- [ ] メールアドレス変更（確認メール送信）
- [ ] パスワード変更

### 表示設定

- [x] テーマ切り替えトグル（ライト / ダーク）— **ただし `SettingsPage.tsx` ではなく `Header.tsx` にグローバルなトグルボタンとして実装**（`theme/ThemeContext.tsx`）
  - 切り替え時に `PUT /api/v1/users/me/theme` を呼ぶ（動作確認済み: ダーク/ライト双方とも表示崩れなし）

### アカウント削除（オプション）

- [ ] 確認ダイアログ付きの危険な操作セクション

> `SettingsPage.tsx` 自体は5行のプレースホルダー。テーマ切り替え以外の設定項目はまだページとして存在しない。

## ブックマークページ（`src/pages/BookmarksPage.tsx`）

- [ ] `useInfiniteQuery` で `GET /api/v1/bookmarks`
- [ ] InfinitePostList で表示

> 現状5行のプレースホルダー（「ブックマーク（PHASE10で実装予定）」の文言のみ）。バックエンド API・ブックマークのトグル操作自体は PostCard から可能（`PHASE06`/`PHASE10` で完成済み）だが、一覧ページがない。

## 完了基準

- [ ] `/@handle` でユーザーのプロフィールと投稿が表示される
- [ ] フォローボタンが即座に反応する（楽観的更新）
- [ ] プロフィール編集ダイアログで保存するとページが更新される
- [ ] アバター・バナー画像を変更できる
- [x] テーマ切り替えが設定画面（の代わりにヘッダー）から行える
- [ ] ブックマーク一覧が表示される

## 備考

このフェーズはテーマ切り替え以外ほぼ未着手。対応するバックエンド API（プロフィール取得・更新・フォロー・アバター/バナーアップロード・ブックマーク一覧）は `PHASE04_user_follow.md` / `PHASE06_interactions.md` で完成済みのため、フロントエンドの実装だけで着手できる状態。
