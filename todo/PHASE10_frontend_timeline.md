# PHASE 10 — フロントエンド タイムライン・投稿

> 目標: ホーム/探索タイムラインと投稿作成・詳細画面を実装する。無限スクロールを動かす。

---

## 共通コンポーネント（`src/components/`）

### PostCard（`src/components/PostCard/`）

- [ ] ユーザーアバター・表示名・@handle・投稿日時
- [ ] テキスト本文（ハッシュタグ・メンションをリンク化）
- [ ] メディア表示
  - 画像: グリッドレイアウト（1 枚・2 枚・3 枚・4 枚それぞれ対応）
  - 動画: HTML5 `<video>` タグ（コントロールあり）
- [ ] アクションバー（いいね・コメント・リポスト・ブックマーク・共有）
  - いいね: ハートアイコン、カウント表示、押下でトグル
  - コメント: バブルアイコン、カウント表示、クリックで投稿詳細へ
  - リポスト: カウント表示、押下でトグル
  - ブックマーク: ブックマークアイコン、押下でトグル
- [ ] 「編集済み」ラベル（is_edited が true のとき）
- [ ] 3 点メニュー（本人の場合: 編集・削除、他人の場合: 非表示・報告）

### PostSkeleton（ローディング中のプレースホルダー）

- [ ] MUI の `Skeleton` コンポーネントで PostCard 形状のスケルトンを表示

### InfinitePostList（`src/components/InfinitePostList.tsx`）

- [ ] `useInfiniteQuery` で cursor ページネーション
- [ ] `IntersectionObserver` でスクロール末尾を検知して次ページをフェッチ
- [ ] ローディング中は PostSkeleton を表示
- [ ] 「これ以上投稿はありません」の終端表示

---

## 投稿作成フォーム（`src/features/posts/components/PostForm.tsx`）

- [ ] テキストエリア（280 文字カウンター付き）
- [ ] 画像添付ボタン（最大 4 枚、プレビュー付き）
- [ ] 動画添付ボタン（最大 1 本）
- [ ] 添付ファイルのプレビュー（削除ボタン付き）
- [ ] 投稿ボタン（テキスト未入力かつメディアなしは disabled）
- [ ] 送信中はボタンをローディング状態に
- [ ] 画面幅に応じた表示
  - PC: タイムライン上部に常時表示
  - スマホ: FAB ボタン → ダイアログで表示

## 投稿 API フック（`src/features/posts/hooks/`）

- [ ] `useHomeTimeline.ts` — `useInfiniteQuery` で `GET /api/v1/posts/home`
- [ ] `useExploreTimeline.ts` — `useInfiniteQuery` で `GET /api/v1/posts`
- [ ] `useCreatePost.ts` — `useMutation` で `POST /api/v1/posts`（multipart）
- [ ] `useUpdatePost.ts` — `useMutation` で `PUT /api/v1/posts/:id`
- [ ] `useDeletePost.ts` — `useMutation` で `DELETE /api/v1/posts/:id`
- [ ] `usePost.ts` — `useQuery` で `GET /api/v1/posts/:id`
- [ ] `useComments.ts` — `useInfiniteQuery` で `GET /api/v1/posts/:id/comments`

## ホームページ（`src/pages/HomePage.tsx`）

- [ ] 投稿フォーム（上部）
- [ ] InfinitePostList（フォローしているユーザーの投稿）
- [ ] 新着投稿ありの場合に「新しい投稿を見る」バナーを表示

## 探索ページ（`src/pages/ExplorePage.tsx`）

- [ ] 検索バー（入力 → `GET /api/v1/search/posts?q=`）
- [ ] 未検索時は全体タイムライン（InfinitePostList）
- [ ] 検索結果表示（InfinitePostList でラップ）

## 投稿詳細ページ（`src/pages/PostDetailPage.tsx`）

- [ ] 元投稿の PostCard 表示
- [ ] コメント投稿フォーム
- [ ] コメント一覧（InfinitePostList）

## 投稿編集ダイアログ

- [ ] 既存テキストを初期値として MUI `Dialog` で編集フォームを表示
- [ ] 保存後に PostCard のテキストを更新（TanStack Query キャッシュ更新）

## インタラクション API フック（`src/features/posts/hooks/`）

- [ ] `useLike.ts` — いいね / いいね取消の `useMutation`（楽観的更新）
- [ ] `useRepost.ts` — リポスト / 取消の `useMutation`（楽観的更新）
- [ ] `useBookmark.ts` — ブックマーク / 取消の `useMutation`（楽観的更新）

## 完了基準

- [ ] ホームタイムラインに自分とフォロー中ユーザーの投稿が表示される
- [ ] スクロール末尾で自動的に次の投稿が読み込まれる
- [ ] 投稿作成で画像/動画を添付して送信できる
- [ ] いいねボタンが即座に反応する（楽観的更新）
- [ ] 投稿削除後にタイムラインから消える（キャッシュ更新）
- [ ] スマホでも投稿フォームが使えるレスポンシブ対応
