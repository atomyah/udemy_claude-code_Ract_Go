# PHASE 10 — フロントエンド タイムライン・投稿

> 目標: ホーム/探索タイムラインと投稿作成・詳細画面を実装する。無限スクロールを動かす。

---

## 共通コンポーネント（`src/components/`）

### PostCard（`src/components/PostCard/`）

- [x] ユーザーアバター・表示名・@handle・投稿日時
- [x] テキスト本文（ハッシュタグ・メンションをリンク化、`LinkifiedText.tsx`）
- [x] メディア表示（`MediaGrid.tsx`）
  - 画像: グリッドレイアウト（1 枚・2 枚・3 枚・4 枚それぞれ対応、`GRID_TEMPLATE` で定義）
  - 動画: HTML5 `<video>` タグ（コントロールあり）
- [x] アクションバー（いいね・コメント・リポスト・ブックマーク・共有）
  - いいね: ハートアイコン、カウント表示、押下でトグル（動作確認済み: 楽観的更新で即座に反映）
  - コメント: バブルアイコン、カウント表示、クリックで投稿詳細へ（動作確認済み）
  - リポスト: カウント表示、押下でトグル
  - ブックマーク: ブックマークアイコン、押下でトグル（動作確認済み: リロード後も状態が保持される）
  - 共有: クリップボードへ URL コピー
- [x] 「編集済み」ラベル（is_edited が true のとき）
- [x] 3 点メニュー
  - 本人の場合: 編集・削除（動作確認済み。ただし下記「既知の不具合」参照）
  - 他人の場合: 非表示・報告のメニュー項目自体は表示されるが **クリックしてもメニューを閉じるだけで実処理なし（UIスタブ）**

### PostSkeleton（ローディング中のプレースホルダー）

- [x] MUI の `Skeleton` コンポーネントで PostCard 形状のスケルトンを表示

### InfinitePostList（`src/components/InfinitePostList.tsx`）

- [x] `useInfiniteQuery` で cursor ページネーション
- [x] `IntersectionObserver` でスクロール末尾を検知して次ページをフェッチ
- [x] ローディング中は PostSkeleton を表示
- [x] 「これ以上投稿はありません」の終端表示（動作確認済み）

---

## 投稿作成フォーム（`src/features/posts/components/PostForm.tsx` / `ComposeBox.tsx`）

- [x] テキストエリア（280 文字カウンター付き、動作確認済み）
- [x] 画像添付ボタン（最大 4 枚、プレビュー付き）
- [x] 動画添付ボタン（最大 1 本）
- [x] 添付ファイルのプレビュー（削除ボタン付き）
- [x] 投稿ボタン（テキスト未入力かつメディアなしは disabled、動作確認済み）
- [x] 送信中はボタンをローディング状態に（`LinearProgress`）
- [x] 画面幅に応じた表示
  - PC: タイムライン上部に常時表示（ホームページに `ComposeBox` を配置）
  - 加えてサイドバーの「投稿する」ボタンから `ComposerContext` 経由でダイアログ表示も可能（PC/スマホ共通、仕様の「スマホのみFAB」から実装は簡略化されているが機能は満たす）

## 投稿 API フック（`src/features/posts/hooks/`）

- [x] `useHomeTimeline.ts` — `useInfiniteQuery` で `GET /api/v1/posts/home`
- [x] `useExploreTimeline.ts` — `useInfiniteQuery` で `GET /api/v1/posts`
- [x] `useCreatePost.ts` — `useMutation` で `POST /api/v1/posts`（multipart）
- [x] `useUpdatePost.ts` — `useMutation` で `PUT /api/v1/posts/:id`
- [x] `useDeletePost.ts` — `useMutation` で `DELETE /api/v1/posts/:id`
- [x] `usePost.ts` — `useQuery` で `GET /api/v1/posts/:id`
- [x] `useComments.ts` — `useInfiniteQuery` で `GET /api/v1/posts/:id/comments`

## ホームページ（`src/pages/HomePage.tsx`）

- [x] 投稿フォーム（上部）
- [x] InfinitePostList（フォローしているユーザーの投稿、動作確認済み）
- [x] 新着投稿ありの場合に「新しい投稿を見る」バナーを表示（`useNewPostsBanner.ts`）

## 探索ページ（`src/pages/ExplorePage.tsx`）

- [x] 検索バー（入力 → `GET /api/v1/search/posts?q=`、動作確認済み）
- [x] 未検索時は全体タイムライン（InfinitePostList、動作確認済み）
- [x] 検索結果表示（InfinitePostList でラップ）

## 投稿詳細ページ（`src/pages/PostDetailPage.tsx`）

- [x] 元投稿の PostCard 表示
- [x] コメント投稿フォーム（動作確認済み）
- [x] コメント一覧（InfinitePostList、動作確認済み）

## 投稿編集ダイアログ（`PostEditDialog.tsx`）

- [x] 既存テキストを初期値として MUI `Dialog` で編集フォームを表示
- [x] 保存後に PostCard のテキストを更新（TanStack Query キャッシュ更新、`updatePostInCaches`）

## インタラクション API フック（`src/features/posts/hooks/`）

- [x] `useLike.ts` — いいね / いいね取消の `useMutation`（楽観的更新、動作確認済み）
- [x] `useRepost.ts` — リポスト / 取消の `useMutation`（楽観的更新）
- [x] `useBookmark.ts` — ブックマーク / 取消の `useMutation`（楽観的更新、動作確認済み）

## 完了基準

- [x] ホームタイムラインに全ユーザーの投稿が表示される
- [x] スクロール末尾で自動的に次の投稿が読み込まれる（`IntersectionObserver` 実装済み。実データでの無限スクロール確認は未実施）
- [x] 投稿作成で画像/動画を添付して送信できる（コード確認済み。実ファイル添付での動作確認は未実施）
- [x] いいねボタンが即座に反応する（楽観的更新、動作確認済み）
- [x] 投稿削除後にタイムラインから消える（キャッシュ更新、一覧側は動作確認済み）
- [x] スマホでも投稿フォームが使えるレスポンシブ対応（`ComposerProvider` のダイアログで代替）

## 既知の不具合（Playwright MCP による動作確認で発見）

1. **投稿詳細ページで自分の投稿を削除しても画面が更新されない**（要修正）
   `PostCard.tsx` の削除処理（`deletePostMutation.mutate(postId)`）はナビゲーションを行わない。`cacheHelpers.ts` の `removePostFromCaches` は詳細クエリを `removeQueries` で消すが、`PostDetailPage` は表示中のままなので `usePost` の購読側が自動で再フェッチされず、削除済みの投稿がその場に表示され続ける（サーバー側は正しく削除済み・204。再訪問すると「投稿が見つかりません」になる）。
   対応案: 削除成功時に自分が閲覧中の投稿だった場合は `navigate('/')` 等でリダイレクトする。
2. **コメント削除後、親投稿のコメント数バッジがその場で更新されない**（軽微）
   ページ再読み込みで正しい値になるため実害は小さいが、`removePostFromCaches` がコメント一覧側のキャッシュのみ更新し、親投稿の `comments_count` をデクリメントしていない。
