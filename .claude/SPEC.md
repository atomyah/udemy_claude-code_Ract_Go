# SNSアプリ 要件定義・仕様書

## プロジェクト概要

Twitterライクな SNS アプリ。ユーザーがテキスト・画像・動画を投稿し、いいね・コメント・リポスト・ブックマークでインタラクションできる。

---

## 機能要件

### 1. ユーザー認証

| 機能 | 仕様 |
|------|------|
| サインアップ | メール＋パスワード、または Google OAuth |
| ログイン | 同上 |
| ログアウト | JWT トークン破棄 |
| パスワードリセット | メール送信で再設定 |
| 認証方式 | JWT（Access Token + Refresh Token）、Firebase Auth 経由の Google OAuth |

### 2. プロフィール

| フィールド | 詳細 |
|-----------|------|
| アイコン画像 | Firebase Storage に保存。デフォルトアバターあり |
| バナー画像 | Firebase Storage に保存。Twitter ヘッダー相当 |
| 表示名 | 任意の表示名（変更可） |
| ユーザー ID（@handle） | ユニーク。英数字＋アンダースコア。変更可 |
| 自己紹介（bio） | 最大 160 文字 |
| 場所 | 最大 30 文字 |
| ウェブサイト URL | 最大 100 文字 |
| 誕生日 | 日付。公開範囲は自分のみ or 全体 |
| フォロワー数 / フォロー数 | プロフィールページに表示 |

### 3. 投稿（Post）

| 項目 | 仕様 |
|------|------|
| テキスト | 必須。最大 280 文字 |
| 画像 | 最大 4 枚。JPEG / PNG / WebP。1 枚あたり最大 5MB |
| 動画 | 最大 1 本。MP4 / MOV。最大 100MB、最大 2 分 20 秒 |
| ハッシュタグ | `#tag` 形式を自動リンク |
| メンション | `@handle` 形式を自動リンク |
| 編集 | 投稿後に内容を編集可能（編集済みラベルを表示） |
| 削除 | 自分の投稿を削除可能（論理削除） |

### 4. タイムライン

| 種別 | 仕様 |
|------|------|
| ホーム | フォロー中ユーザー＋自分の投稿。新着順 |
| 探索（全体） | 全ユーザーの公開投稿。新着順 |
| スクロール方式 | 無限スクロール（cursor ベースのページネーション） |

### 5. インタラクション

| 機能 | 仕様 |
|------|------|
| いいね | トグル式。いいね数をカウント表示 |
| コメント（返信） | 投稿に対してテキストで返信。ネストは 1 段階 |
| リポスト | 他ユーザーの投稿を自分のタイムラインに共有 |
| ブックマーク | 後で見返す投稿を保存。自分だけに見える |

### 6. フォロー / フォロワー

| 機能 | 仕様 |
|------|------|
| フォロー | 他ユーザーをフォロー / アンフォロー |
| フォロー一覧 | 自分または他ユーザーのフォロー中一覧 |
| フォロワー一覧 | 自分または他ユーザーのフォロワー一覧 |

### 7. 検索

| 種別 | 仕様 |
|------|------|
| ユーザー検索 | 表示名 / @handle で前方一致検索 |
| 投稿内容検索 | テキスト全文検索（PostgreSQL の `tsvector` or LIKE） |
| ハッシュタグ検索 | `#tag` をクリックまたは検索でまとめ表示 |

### 8. 通知

| 通知種別 | トリガー |
|---------|---------|
| いいね | 自分の投稿にいいねが付いた |
| コメント | 自分の投稿にコメントが付いた |
| フォロー | 自分がフォローされた |
| リポスト | 自分の投稿がリポストされた |
| メンション | 投稿内で `@handle` されたとき |

- 未読バッジ表示。既読マーク機能あり。

### 9. テーマ切り替え

- ライトモード / ダークモード の 2 種類
- ヘッダー右上のボタンまたは設定画面から切り替え
- ユーザー設定として DB に保存（ログイン時に引き継ぎ）
- 未ログイン時は `localStorage` に保存

### 10. 管理者機能

| 機能 | 仕様 |
|------|------|
| 投稿削除 | 管理者が任意の投稿を強制削除 |
| ユーザー停止 | 管理者がユーザーをアカウント停止（ログイン不可） |
| ユーザー停止解除 | 停止を解除 |
| 管理者ロール | `users.role = 'admin'` で判定 |

---

## 画面一覧

| 画面名 | パス | 説明 |
|-------|------|------|
| ホーム | `/` | フォロー中のタイムライン |
| 探索 | `/explore` | 全体の投稿・検索 |
| 通知 | `/notifications` | 通知一覧 |
| ブックマーク | `/bookmarks` | ブックマーク一覧 |
| プロフィール | `/:handle` | ユーザーのプロフィールと投稿 |
| 投稿詳細 | `/posts/:id` | 投稿＋コメントスレッド |
| 設定 | `/settings` | プロフィール編集・テーマ・アカウント設定 |
| ログイン | `/login` | ログイン画面 |
| サインアップ | `/signup` | 新規登録画面 |
| 管理者 | `/admin` | 管理者専用ダッシュボード（ロール制限） |

---

## データモデル（概要）

### users
```
id            UUID PK
email         VARCHAR UNIQUE NOT NULL
password_hash VARCHAR (Google OAuth ユーザーは NULL)
handle        VARCHAR UNIQUE NOT NULL
display_name  VARCHAR NOT NULL
avatar_url    VARCHAR
banner_url    VARCHAR
bio           VARCHAR(160)
location      VARCHAR(30)
website_url   VARCHAR(100)
birthday      DATE
theme         VARCHAR DEFAULT 'light'
role          VARCHAR DEFAULT 'user'  -- 'user' | 'admin'
is_suspended  BOOLEAN DEFAULT false
created_at    TIMESTAMP
updated_at    TIMESTAMP
```

### posts
```
id            UUID PK
user_id       UUID FK -> users
content       VARCHAR(280) NOT NULL
is_edited     BOOLEAN DEFAULT false
is_deleted    BOOLEAN DEFAULT false
repost_of     UUID FK -> posts (NULL でなければリポスト)
reply_to      UUID FK -> posts (NULL でなければコメント)
created_at    TIMESTAMP
updated_at    TIMESTAMP
```

### media
```
id            UUID PK
post_id       UUID FK -> posts
url           VARCHAR NOT NULL
type          VARCHAR  -- 'image' | 'video'
order         INT
created_at    TIMESTAMP
```

### likes
```
id            UUID PK
user_id       UUID FK -> users
post_id       UUID FK -> posts
created_at    TIMESTAMP
UNIQUE(user_id, post_id)
```

### bookmarks
```
id            UUID PK
user_id       UUID FK -> users
post_id       UUID FK -> posts
created_at    TIMESTAMP
UNIQUE(user_id, post_id)
```

### follows
```
id            UUID PK
follower_id   UUID FK -> users
following_id  UUID FK -> users
created_at    TIMESTAMP
UNIQUE(follower_id, following_id)
```

### hashtags
```
id            UUID PK
name          VARCHAR UNIQUE NOT NULL
```

### post_hashtags
```
post_id       UUID FK -> posts
hashtag_id    UUID FK -> hashtags
PRIMARY KEY(post_id, hashtag_id)
```

### notifications
```
id            UUID PK
user_id       UUID FK -> users (通知受信者)
actor_id      UUID FK -> users (アクション実行者)
type          VARCHAR  -- 'like' | 'comment' | 'follow' | 'repost' | 'mention'
post_id       UUID FK -> posts (NULL の場合もあり)
is_read       BOOLEAN DEFAULT false
created_at    TIMESTAMP
```

---

## API 設計（概要）

### 認証
```
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/logout
POST /api/v1/auth/refresh
POST /api/v1/auth/google
```

### ユーザー
```
GET    /api/v1/users/:handle
PUT    /api/v1/users/me
GET    /api/v1/users/:handle/posts
GET    /api/v1/users/:handle/followers
GET    /api/v1/users/:handle/following
POST   /api/v1/users/:handle/follow
DELETE /api/v1/users/:handle/follow
```

### 投稿
```
GET    /api/v1/posts              (探索: 全体タイムライン)
GET    /api/v1/posts/home         (ホーム: フォロー中)
POST   /api/v1/posts
GET    /api/v1/posts/:id
PUT    /api/v1/posts/:id
DELETE /api/v1/posts/:id
GET    /api/v1/posts/:id/comments
```

### インタラクション
```
POST   /api/v1/posts/:id/like
DELETE /api/v1/posts/:id/like
POST   /api/v1/posts/:id/repost
DELETE /api/v1/posts/:id/repost
POST   /api/v1/posts/:id/bookmark
DELETE /api/v1/posts/:id/bookmark
```

### 検索
```
GET /api/v1/search/users?q=
GET /api/v1/search/posts?q=
GET /api/v1/search/hashtags/:tag
```

### 通知
```
GET  /api/v1/notifications
PUT  /api/v1/notifications/read
```

### 管理者
```
DELETE /api/v1/admin/posts/:id
PUT    /api/v1/admin/users/:id/suspend
DELETE /api/v1/admin/users/:id/suspend
```

---

## 非機能要件

| 項目 | 仕様 |
|------|------|
| レスポンシブ | MUI の Breakpoints を使い、PC / タブレット / スマホ対応 |
| ページネーション | cursor ベース（`?cursor=<id>&limit=20`）で無限スクロール実装 |
| セキュリティ | JWT の有効期限（Access: 15 分、Refresh: 7 日）、HTTPS 必須 |
| ファイルアップロード | multipart/form-data でバックエンドが受け取り Firebase Storage へ転送 |
| エラーレスポンス | 統一フォーマット `{ "code": "...", "message": "..." }` |

---

## 技術スタック

### フロントエンド
- React 18 + TypeScript
- MUI (Material UI) v5（カスタムテーマ: ライト / ダーク）
- openapi-typescript（Swagger から型自動生成）
- React Router v6
- React Query（TanStack Query）でサーバー状態管理
- Axios（API クライアント）

### バックエンド
- Go 1.22+
- Echo v4（HTTP フレームワーク）
- GORM v2（ORM）
- swaggo/echo-swagger（OpenAPI 3.0 定義自動生成）
- golang-jwt/jwt（JWT 認証）
- air（ホットリロード、開発環境のみ）

### データベース
- PostgreSQL 16

### ファイルストレージ
- Firebase Storage（画像・動画・アバター・バナー）

### 認証
- JWT（自前発行）
- Firebase Auth（Google OAuth 連携）

### インフラ・デプロイ
| 環境 | 構成 |
|------|------|
| ローカル開発 | Docker + docker-compose（PostgreSQL + Go バックエンド）、フロントは `npm run dev` |
| バックエンド本番 | Render または Google Cloud Run |
| フロントエンド本番 | Firebase Hosting |

---

## ディレクトリ構成（予定）

```
project-root/
├── frontend/                 # React アプリ
│   ├── src/
│   │   ├── api/             # openapi-typescript 生成型 + Axios クライアント
│   │   ├── components/      # 共通コンポーネント
│   │   ├── features/        # 機能単位のモジュール（posts, auth, profile...）
│   │   ├── hooks/           # カスタムフック
│   │   ├── pages/           # ページコンポーネント
│   │   ├── theme/           # MUI テーマ定義
│   │   └── utils/
│   ├── package.json
│   └── tsconfig.json
├── backend/                  # Go アプリ
│   ├── cmd/server/          # エントリーポイント
│   ├── internal/
│   │   ├── handler/         # Echo ハンドラー
│   │   ├── service/         # ビジネスロジック
│   │   ├── repository/      # GORM リポジトリ
│   │   ├── model/           # GORM モデル
│   │   ├── middleware/       # JWT 認証ミドルウェアなど
│   │   └── dto/             # リクエスト / レスポンス構造体
│   ├── docs/                # swaggo 生成 OpenAPI ドキュメント
│   ├── .air.toml            # air 設定
│   └── go.mod
├── docker-compose.yml
└── .claude/
    ├── CLAUDE.md
    ├── SPEC.md               # この仕様書
    └── rules/
```
