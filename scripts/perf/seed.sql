-- 性能計測用のテストデータ（テスト用DB db_test 専用。開発用DBには絶対に流さない）
--
-- psql 変数 viewer_id に計測用ユーザー（APIで登録済み）のIDを渡して実行する:
--   psql -v viewer_id=<uuid> -f seed.sql
--
-- テスト用DBのスキーマは AutoMigrate で作られ id / created_at に DB 側のデフォルトが無いため、
-- すべてのカラムを明示して INSERT する。
--
-- 投入するデータ（タイムライン1ページ=20件の各投稿に いいね・コメント・リポスト・ブックマーク・メディアが付く構成）:
--   ユーザー     : 計測用ユーザー + 著者 9 人
--   投稿         : 300 件（3 件に 1 件は画像付き）
--   返信         : 200 件
--   リポスト     : 50 件
--   いいね       : 新しい 100 投稿に対してユーザーの約 2/3 が いいね
--   ブックマーク : 計測用ユーザーが 4 件に 1 件
--   フォロー     : 計測用ユーザー → 著者 9 人
\set ON_ERROR_STOP on

BEGIN;

INSERT INTO users (id, email, handle, display_name, theme, role, is_suspended, created_at, updated_at)
SELECT gen_random_uuid(), 'perf_author_' || g || '@example.com', 'perf_author_' || g, '計測用ユーザー' || g,
       'light', 'user', false, now(), now()
FROM generate_series(1, 9) AS g;

CREATE TEMP TABLE perf_users ON COMMIT DROP AS
SELECT id, row_number() OVER (ORDER BY handle) AS idx
FROM users
WHERE id = :'viewer_id'::uuid OR handle LIKE 'perf\_author\_%';

-- 投稿: 1 分間隔で過去に遡る
INSERT INTO posts (id, user_id, content, is_edited, is_deleted, created_at, updated_at)
SELECT gen_random_uuid(), u.id, '性能計測用の投稿 #' || g || ' #perf', false, false,
       now() - make_interval(mins => g), now() - make_interval(mins => g)
FROM generate_series(1, 300) AS g
JOIN perf_users u ON u.idx = (g % 10) + 1;

CREATE TEMP TABLE perf_posts ON COMMIT DROP AS
SELECT id, row_number() OVER (ORDER BY created_at DESC) AS idx
FROM posts
WHERE content LIKE '性能計測用の投稿 #%';

INSERT INTO media (id, post_id, url, type, sort_order, created_at)
SELECT gen_random_uuid(), p.id, 'https://example.com/perf/' || p.idx || '.jpg', 'image', 0, now()
FROM perf_posts p
WHERE p.idx % 3 = 0;

-- 返信: 新しい 40 投稿に分散させ、投稿の合間（30 秒ずらし）に並べる
INSERT INTO posts (id, user_id, content, reply_to, is_edited, is_deleted, created_at, updated_at)
SELECT gen_random_uuid(), u.id, '返信 #' || g, p.id, false, false,
       now() - make_interval(mins => g, secs => 30), now() - make_interval(mins => g, secs => 30)
FROM generate_series(1, 200) AS g
JOIN perf_users u ON u.idx = (g % 10) + 1
JOIN perf_posts p ON p.idx = (g % 40) + 1;

-- リポスト: 新しい 25 投稿を 2 人ずつリポスト（45 秒ずらし）
INSERT INTO posts (id, user_id, content, repost_of, is_edited, is_deleted, created_at, updated_at)
SELECT gen_random_uuid(), u.id, '', p.id, false, false,
       now() - make_interval(mins => g, secs => 45), now() - make_interval(mins => g, secs => 45)
FROM generate_series(1, 50) AS g
JOIN perf_users u ON u.idx = (g % 10) + 1
JOIN perf_posts p ON p.idx = (g % 25) + 1;

INSERT INTO likes (id, user_id, post_id, created_at)
SELECT gen_random_uuid(), u.id, p.id, now()
FROM perf_users u
JOIN perf_posts p ON (u.idx + p.idx) % 3 <> 0
WHERE p.idx <= 100;

INSERT INTO bookmarks (id, user_id, post_id, created_at)
SELECT gen_random_uuid(), :'viewer_id'::uuid, p.id, now()
FROM perf_posts p
WHERE p.idx % 4 = 0;

INSERT INTO follows (id, follower_id, following_id, created_at)
SELECT gen_random_uuid(), :'viewer_id'::uuid, u.id, now()
FROM perf_users u
WHERE u.id <> :'viewer_id'::uuid;

COMMIT;

ANALYZE;

SELECT
    (SELECT count(*) FROM users)     AS users,
    (SELECT count(*) FROM posts)     AS posts,
    (SELECT count(*) FROM media)     AS media,
    (SELECT count(*) FROM likes)     AS likes,
    (SELECT count(*) FROM bookmarks) AS bookmarks,
    (SELECT count(*) FROM follows)   AS follows;
