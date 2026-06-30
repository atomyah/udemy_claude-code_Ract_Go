# データベース規約（PostgreSQL / GORM）

## 基本方針

- DB: PostgreSQL 16
- ORM: GORM v2
- マイグレーション: `golang-migrate`（`AutoMigrate` は開発初期のみ許可、本番禁止）
- 主キー: 全テーブルで UUID (`uuid` 型)
- タイムスタンプ: 全テーブルに `created_at`, `updated_at`（`TIMESTAMP WITH TIME ZONE`）

---

## テーブル設計

### users
```sql
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),                     -- Google OAuth ユーザーは NULL
    handle        VARCHAR(50) UNIQUE NOT NULL,       -- @handle
    display_name  VARCHAR(50) NOT NULL,
    avatar_url    VARCHAR(500),
    banner_url    VARCHAR(500),
    bio           VARCHAR(160),
    location      VARCHAR(30),
    website_url   VARCHAR(100),
    birthday      DATE,
    theme         VARCHAR(10) NOT NULL DEFAULT 'light',
    role          VARCHAR(10) NOT NULL DEFAULT 'user', -- 'user' | 'admin'
    is_suspended  BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
```

### posts
```sql
CREATE TABLE posts (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content    VARCHAR(280) NOT NULL,
    is_edited  BOOLEAN NOT NULL DEFAULT false,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    repost_of  UUID REFERENCES posts(id) ON DELETE SET NULL, -- リポスト元
    reply_to   UUID REFERENCES posts(id) ON DELETE SET NULL, -- コメント先
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
```

### media
```sql
CREATE TABLE media (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id    UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    url        VARCHAR(500) NOT NULL,
    type       VARCHAR(10) NOT NULL,  -- 'image' | 'video'
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
```

### likes
```sql
CREATE TABLE likes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_id    UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, post_id)
);
```

### bookmarks
```sql
CREATE TABLE bookmarks (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_id    UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, post_id)
);
```

### follows
```sql
CREATE TABLE follows (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    follower_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    following_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(follower_id, following_id),
    CHECK(follower_id != following_id)
);
```

### hashtags
```sql
CREATE TABLE hashtags (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
```

### post_hashtags
```sql
CREATE TABLE post_hashtags (
    post_id    UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    hashtag_id UUID NOT NULL REFERENCES hashtags(id) ON DELETE CASCADE,
    PRIMARY KEY(post_id, hashtag_id)
);
```

### notifications
```sql
CREATE TABLE notifications (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,   -- 受信者
    actor_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,   -- 送信者
    type       VARCHAR(20) NOT NULL,  -- 'like' | 'comment' | 'follow' | 'repost' | 'mention'
    post_id    UUID REFERENCES posts(id) ON DELETE CASCADE,
    is_read    BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
```

---

## インデックス設計

```sql
-- posts: タイムライン取得のため
CREATE INDEX idx_posts_user_id ON posts(user_id);
CREATE INDEX idx_posts_created_at ON posts(created_at DESC);
CREATE INDEX idx_posts_reply_to ON posts(reply_to) WHERE reply_to IS NOT NULL;

-- likes: いいね数カウントとユーザー確認のため
CREATE INDEX idx_likes_post_id ON likes(post_id);

-- follows: フォロー中タイムライン取得のため
CREATE INDEX idx_follows_follower_id ON follows(follower_id);
CREATE INDEX idx_follows_following_id ON follows(following_id);

-- notifications: 未読通知取得のため
CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_is_read ON notifications(user_id, is_read) WHERE is_read = false;

-- 検索用
CREATE INDEX idx_users_handle ON users(handle);
CREATE INDEX idx_hashtags_name ON hashtags(name);
```

---

## マイグレーション規約

- `golang-migrate` を使う。マイグレーションファイルは `backend/migrations/` に置く。
- ファイル名: `{連番}_{説明}.up.sql` / `{連番}_{説明}.down.sql`（例: `001_create_users.up.sql`）。
- `down.sql` は必ず書く（ロールバック可能にする）。
- `AutoMigrate` は開発初期のスキーマ確認用のみ。本番・CI では使わない。

---

## GORM モデル規約

- モデルファイルは `internal/model/` に置く。1 テーブル 1 ファイル。
- `gorm.Model` は使わない。UUID + 明示的なタイムスタンプで定義する。
- タグで `not null`、`unique`、`default` を明示する。

```go
// 例: internal/model/post.go
package model

import (
    "time"
    "github.com/google/uuid"
)

type Post struct {
    ID        uuid.UUID  `gorm:"type:uuid;primaryKey"`
    UserID    uuid.UUID  `gorm:"type:uuid;not null"`
    Content   string     `gorm:"size:280;not null"`
    IsEdited  bool       `gorm:"not null;default:false"`
    IsDeleted bool       `gorm:"not null;default:false"`
    RepostOf  *uuid.UUID `gorm:"type:uuid"`
    ReplyTo   *uuid.UUID `gorm:"type:uuid"`
    CreatedAt time.Time
    UpdatedAt time.Time
    User      User       `gorm:"foreignKey:UserID"`
    Media     []Media    `gorm:"foreignKey:PostID"`
}
```

---

## クエリ規約

- 削除済み投稿を除外するには `WHERE is_deleted = false` を必ず付ける。
- 停止ユーザーの投稿を除外するには `JOIN users ON ... WHERE users.is_suspended = false` を付ける。
- タイムラインの cursor ページネーション: `WHERE created_at < :cursor ORDER BY created_at DESC LIMIT :limit`。
- N+1 問題を防ぐために関連データは `Preload` または JOIN で一括取得する。

```go
// タイムライン取得の例
db.Where("user_id IN ? AND is_deleted = false AND created_at < ?", followingIDs, cursor).
    Order("created_at DESC").
    Limit(limit).
    Preload("User").
    Preload("Media").
    Find(&posts)
```

---

## 禁止事項

- 本番環境での `AutoMigrate` 使用。
- `DELETE` による物理削除（`is_deleted = true` の論理削除を使う）。
- インデックスなしの大量テーブルへの全件スキャン。
- マイグレーションの `down.sql` を書かずに `up.sql` だけコミットする。
