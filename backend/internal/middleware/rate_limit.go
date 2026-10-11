package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// レートリミット関連のレスポンスヘッダー
const (
	HeaderRateLimitLimit     = "X-RateLimit-Limit"
	HeaderRateLimitRemaining = "X-RateLimit-Remaining"
	// HeaderRateLimitReset は現在のウィンドウがリセットされる時刻（UNIX 秒）
	HeaderRateLimitReset = "X-RateLimit-Reset"
)

// RateLimitWindow はレートリミットのウィンドウ幅（上限値はすべて「回/分」）
const RateLimitWindow = time.Minute

// RateLimitResult は 1 リクエスト分の判定結果
type RateLimitResult struct {
	Allowed   bool
	Limit     int
	Remaining int
	ResetAt   time.Time
	// RetryAfter は拒否時に次のウィンドウまで待つ秒数（1 以上）
	RetryAfter int
}

type windowCounter struct {
	start time.Time
	count int
}

// FixedWindowLimiter は fixed window 方式のインメモリ・レートリミッター。
// Cloud Run の max instances = 1 構成を前提にプロセス内のマップで状態を持つ（再起動で状態はリセットされる）。
// 全操作を mutex で保護しており、複数 goroutine から同時に呼んでもよい。
type FixedWindowLimiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	mu        sync.Mutex
	counters  map[string]*windowCounter
	lastSweep time.Time
}

// NewFixedWindowLimiter はリミッターを作る。now が nil のときは time.Now を使う（テストで時刻を差し替えるため）
func NewFixedWindowLimiter(limit int, window time.Duration, now func() time.Time) *FixedWindowLimiter {
	if now == nil {
		now = time.Now
	}
	return &FixedWindowLimiter{
		limit:    limit,
		window:   window,
		now:      now,
		counters: make(map[string]*windowCounter),
	}
}

// Allow は key のリクエストを 1 回分数え、許可するかどうかを返す。
// ウィンドウは時刻をウィンドウ幅で切り捨てた区間（例: 毎分 0 秒〜59 秒）で、境界でカウントが 0 に戻る
func (l *FixedWindowLimiter) Allow(key string) RateLimitResult {
	now := l.now()
	start := now.Truncate(l.window)
	resetAt := start.Add(l.window)

	l.mu.Lock()
	defer l.mu.Unlock()

	l.sweep(start)

	c, ok := l.counters[key]
	if !ok || c.start.Before(start) {
		c = &windowCounter{start: start}
		l.counters[key] = c
	}

	res := RateLimitResult{Limit: l.limit, ResetAt: resetAt}
	if c.count >= l.limit {
		res.RetryAfter = retryAfterSeconds(resetAt.Sub(now))
		return res
	}
	c.count++
	res.Allowed = true
	res.Remaining = l.limit - c.count
	return res
}

// sweep は前のウィンドウ以前のカウンターを削除してメモリが増え続けないようにする。
// 走査はウィンドウが切り替わった最初の呼び出しで 1 回だけ行う（呼び出し側でロック済み）
func (l *FixedWindowLimiter) sweep(current time.Time) {
	if !l.lastSweep.Before(current) {
		return
	}
	for k, c := range l.counters {
		if c.start.Before(current) {
			delete(l.counters, k)
		}
	}
	l.lastSweep = current
}

func retryAfterSeconds(d time.Duration) int {
	sec := int((d + time.Second - 1) / time.Second) // 切り上げ
	if sec < 1 {
		return 1
	}
	return sec
}

// RateLimitKeyFunc はリクエストの識別子（IP やユーザーID）を返す
type RateLimitKeyFunc func(c echo.Context) string

// KeyByIP はクライアントIP単位で識別する。IP は e.IPExtractor（ClientIPExtractor）で取得した値を使う
func KeyByIP(c echo.Context) string {
	return "ip:" + c.RealIP()
}

// KeyByUserOrIP は認証済みならユーザーID、未認証ならクライアントIPで識別する。
// userID は前段の JWTAuth / OptionalJWTAuth がセットした値を使う
func KeyByUserOrIP(c echo.Context) string {
	if userID, ok := c.Get("userID").(uuid.UUID); ok {
		return "user:" + userID.String()
	}
	return KeyByIP(c)
}

// RateLimit は limiter でリクエスト数を制限するミドルウェアを返す。
// カウントは「scope・メソッド・ルート・識別子」ごとに分けるため、API ごとに上限が独立する。
// 全レスポンスに X-RateLimit-* ヘッダーを付け、超過時は 429 と Retry-After を返す
func RateLimit(scope string, limiter *FixedWindowLimiter, keyFunc RateLimitKeyFunc) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := scope + "|" + c.Request().Method + " " + c.Path() + "|" + keyFunc(c)
			res := limiter.Allow(key)

			h := c.Response().Header()
			h.Set(HeaderRateLimitLimit, strconv.Itoa(res.Limit))
			h.Set(HeaderRateLimitRemaining, strconv.Itoa(res.Remaining))
			h.Set(HeaderRateLimitReset, strconv.FormatInt(res.ResetAt.Unix(), 10))

			if !res.Allowed {
				h.Set(echo.HeaderRetryAfter, strconv.Itoa(res.RetryAfter))
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "リクエストが多すぎます。しばらく待ってから再度お試しください",
				})
			}
			return next(c)
		}
	}
}

// AuthRateLimit は認証系API（ログイン・登録・パスワードリセット）用。IP 単位で limit 回/分に制限する
func AuthRateLimit(limit int) echo.MiddlewareFunc {
	return newAuthRateLimit(limit, nil)
}

// GeneralRateLimit は一般API用。認証済みはユーザーID、未認証は IP 単位で limit 回/分に制限する
func GeneralRateLimit(limit int) echo.MiddlewareFunc {
	return newGeneralRateLimit(limit, nil)
}

func newAuthRateLimit(limit int, now func() time.Time) echo.MiddlewareFunc {
	return RateLimit("auth", NewFixedWindowLimiter(limit, RateLimitWindow, now), KeyByIP)
}

func newGeneralRateLimit(limit int, now func() time.Time) echo.MiddlewareFunc {
	return RateLimit("general", NewFixedWindowLimiter(limit, RateLimitWindow, now), KeyByUserOrIP)
}
