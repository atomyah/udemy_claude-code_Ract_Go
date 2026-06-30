# フロントエンド コーディング規約

## ディレクトリ構成

```
frontend/src/
├── api/              # openapi-typescript 生成型 + Axios インスタンス
│   ├── generated/    # 自動生成ファイル（手動編集禁止）
│   └── client.ts     # Axios インスタンス設定
├── components/       # 再利用可能な UI コンポーネント
├── features/         # 機能モジュール（下記参照）
│   ├── auth/
│   ├── posts/
│   ├── profile/
│   ├── notifications/
│   └── search/
├── hooks/            # カスタムフック（useXxx 命名）
├── pages/            # React Router のページコンポーネント
├── theme/            # MUI テーマ定義
│   ├── index.ts      # テーマエクスポート
│   ├── lightTheme.ts
│   └── darkTheme.ts
└── utils/
```

### features/ の内部構成（機能ごとに統一）

```
features/posts/
├── components/    # その機能専用コンポーネント
├── hooks/         # TanStack Query フック（usePostsQuery など）
├── types.ts       # 機能固有の型（生成型の補助）
└── index.ts       # 外部向けエクスポート
```

---

## TypeScript 規約

- `any` は禁止。`unknown` を使い型ガードで絞る。
- API レスポンス型は openapi-typescript の生成型のみを使う。手書きの型は作らない。
- Props 型は `interface` で定義し、コンポーネントファイル内に置く。
- `export default` はページコンポーネントのみ。その他は named export。

---

## コンポーネント規約

- 関数コンポーネント（アロー関数）のみ使う。クラスコンポーネントは使わない。
- 1 ファイル 1 コンポーネントを原則とする。
- コンポーネント名はパスカルケース（`PostCard`, `UserAvatar`）。
- ファイル名はコンポーネント名と一致させる（`PostCard.tsx`）。

---

## MUI 規約

- スタイリングは MUI の `sx` prop を使う。インライン `style` や CSS Modules は使わない。
- カラーは MUI テーマの `theme.palette.*` を参照する。ハードコードの色指定は禁止。
- `Typography` コンポーネントを使い、`<p>` や `<h1>` を直接使わない。
- レスポンシブは MUI の `sx={{ display: { xs: '...', md: '...' } }}` パターンで書く。

### テーマ設定

`theme/lightTheme.ts` と `theme/darkTheme.ts` に MUI `createTheme` で定義する。

```ts
// theme/lightTheme.ts の例
import { createTheme } from '@mui/material/styles';

export const lightTheme = createTheme({
  palette: {
    mode: 'light',
    primary: { main: '#1976d2' },
    // ...
  },
  typography: {
    fontFamily: '"Noto Sans JP", "Roboto", sans-serif',
  },
});
```

テーマ切り替えは `ThemeContext` で管理し、設定値を `localStorage` または API 経由で永続化する。

---

## データフェッチ規約（TanStack Query）

- サーバーデータは TanStack Query の `useQuery` / `useMutation` で管理する。`useState` + `useEffect` で直接フェッチしない。
- Query Key は配列形式で定義し、`features/xxx/hooks/queryKeys.ts` に集約する。
- 無限スクロールは `useInfiniteQuery` を使い、cursor ベースのページネーションと合わせる。

```ts
// cursor ベースの無限スクロール例
useInfiniteQuery({
  queryKey: ['posts', 'home'],
  queryFn: ({ pageParam }) => fetchHomePosts({ cursor: pageParam, limit: 20 }),
  getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
});
```

---

## 認証・API クライアント

- Axios インスタンスは `api/client.ts` で一元管理する。
- Access Token は Axios インターセプターでヘッダーに付与する。
- 401 レスポンス時にリフレッシュトークンで再取得し、失敗したらログアウトする。

---

## ファイルアップロード

- 画像・動画は `multipart/form-data` でバックエンドに送る。Firebase Storage に直接アップロードしない。
- アップロード中は MUI の `LinearProgress` で進捗を表示する。
- 画像プレビューは `URL.createObjectURL` で生成する。

---

## レスポンシブ対応

MUI の Breakpoint を使う。

| Breakpoint | 対象 |
|-----------|------|
| `xs` (0-600px) | スマートフォン |
| `sm` (600-900px) | タブレット縦 |
| `md` (900-1200px) | タブレット横 / 小型 PC |
| `lg` (1200px+) | PC |

- サイドバーは `md` 以上で表示。
- 投稿フォームはスマホではボトムシートまたはフルスクリーンダイアログ。

---

## 禁止事項

- `console.log` をコミットしない（デバッグ後は削除）。
- `// eslint-disable` コメントは原則禁止。
- `any` 型の使用禁止。
- `useEffect` の依存配列に `[]` を書いて意図的に無視することは禁止。
