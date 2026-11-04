package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// DashboardTimeZone は日別集計の日付の区切りに使うタイムゾーン（運営者向けに日本時間で集計する）
const DashboardTimeZone = "Asia/Tokyo"

// DashboardSummary は管理画面ダッシュボードの集計値
type DashboardSummary struct {
	TotalUsers     int64
	PendingUsers   int64
	ApprovedUsers  int64
	RejectedUsers  int64
	SuspendedUsers int64
	NewUsers7d     int64
	TotalPosts     int64
	TotalComments  int64
	PostsToday     int64
	ActiveUsers7d  int64
	Likes7d        int64
	PendingResets  int64
	IssuedResets   int64
	TotalAdmins    int64
}

// DailyCount は 1 日分の投稿数・新規登録数
type DailyCount struct {
	Day   time.Time
	Posts int64
	Users int64
}

// DashboardRepository は管理画面のダッシュボード用の集計を行う。
// 各集計は GROUP BY / FILTER で 1 クエリにまとめ、件数に比例してクエリが増えないようにする
type DashboardRepository interface {
	Summary(ctx context.Context, now time.Time) (*DashboardSummary, error)
	// Daily は直近 days 日分（今日を含む）の日別件数を古い順に返す。件数 0 の日も含む
	Daily(ctx context.Context, days int) ([]DailyCount, error)
}

type dashboardRepository struct {
	db *gorm.DB
}

func NewDashboardRepository(db *gorm.DB) DashboardRepository {
	return &dashboardRepository{db: db}
}

func (r *dashboardRepository) Summary(ctx context.Context, now time.Time) (*DashboardSummary, error) {
	db := r.db.WithContext(ctx)
	var s DashboardSummary
	weekAgo := now.Add(-7 * 24 * time.Hour)

	if err := db.Raw(`
		SELECT
			COUNT(*)                                        AS total_users,
			COUNT(*) FILTER (WHERE status = 'pending')      AS pending_users,
			COUNT(*) FILTER (WHERE status = 'approved')     AS approved_users,
			COUNT(*) FILTER (WHERE status = 'rejected')     AS rejected_users,
			COUNT(*) FILTER (WHERE is_suspended)            AS suspended_users,
			COUNT(*) FILTER (WHERE created_at >= ?)         AS new_users7d
		FROM users`, weekAgo).Scan(&s).Error; err != nil {
		return nil, err
	}

	var posts struct {
		TotalPosts    int64
		TotalComments int64
		PostsToday    int64
		ActiveUsers7d int64
	}
	if err := db.Raw(`
		SELECT
			COUNT(*) FILTER (WHERE reply_to IS NULL AND repost_of IS NULL)  AS total_posts,
			COUNT(*) FILTER (WHERE reply_to IS NOT NULL)                    AS total_comments,
			COUNT(*) FILTER (WHERE reply_to IS NULL AND repost_of IS NULL
				AND (created_at AT TIME ZONE ?)::date = (?::timestamptz AT TIME ZONE ?)::date) AS posts_today,
			COUNT(DISTINCT user_id) FILTER (WHERE created_at >= ?)           AS active_users7d
		FROM posts
		WHERE is_deleted = false`, DashboardTimeZone, now, DashboardTimeZone, weekAgo).Scan(&posts).Error; err != nil {
		return nil, err
	}
	s.TotalPosts, s.TotalComments, s.PostsToday, s.ActiveUsers7d = posts.TotalPosts, posts.TotalComments, posts.PostsToday, posts.ActiveUsers7d

	var others struct {
		Likes7d       int64
		PendingResets int64
		IssuedResets  int64
		TotalAdmins   int64
	}
	if err := db.Raw(`
		SELECT
			(SELECT COUNT(*) FROM likes WHERE created_at >= ?)                               AS likes7d,
			(SELECT COUNT(*) FROM password_reset_requests WHERE status = 'pending')         AS pending_resets,
			(SELECT COUNT(*) FROM password_reset_requests WHERE status = 'issued')          AS issued_resets,
			(SELECT COUNT(*) FROM admins WHERE is_active)                                    AS total_admins`,
		weekAgo).Scan(&others).Error; err != nil {
		return nil, err
	}
	s.Likes7d, s.PendingResets, s.IssuedResets, s.TotalAdmins = others.Likes7d, others.PendingResets, others.IssuedResets, others.TotalAdmins

	return &s, nil
}

func (r *dashboardRepository) Daily(ctx context.Context, days int) ([]DailyCount, error) {
	var rows []DailyCount
	// 集計対象は「今日を含む直近 days 日」。範囲の開始は日本時間の日付で切り、その前日分まで余裕をもって絞り込む
	err := r.db.WithContext(ctx).Raw(`
		WITH d AS (
			SELECT generate_series(
				(NOW() AT TIME ZONE ?)::date - CAST(? AS int),
				(NOW() AT TIME ZONE ?)::date,
				INTERVAL '1 day'
			)::date AS day
		),
		p AS (
			SELECT (created_at AT TIME ZONE ?)::date AS day, COUNT(*) AS cnt
			FROM posts
			WHERE is_deleted = false AND reply_to IS NULL AND repost_of IS NULL
			  AND created_at >= NOW() - make_interval(days => CAST(? AS int))
			GROUP BY 1
		),
		u AS (
			SELECT (created_at AT TIME ZONE ?)::date AS day, COUNT(*) AS cnt
			FROM users
			WHERE created_at >= NOW() - make_interval(days => CAST(? AS int))
			GROUP BY 1
		)
		SELECT d.day AS day, COALESCE(p.cnt, 0) AS posts, COALESCE(u.cnt, 0) AS users
		FROM d
		LEFT JOIN p ON p.day = d.day
		LEFT JOIN u ON u.day = d.day
		ORDER BY d.day`,
		DashboardTimeZone, days-1, DashboardTimeZone,
		DashboardTimeZone, days+1,
		DashboardTimeZone, days+1,
	).Scan(&rows).Error
	return rows, err
}
