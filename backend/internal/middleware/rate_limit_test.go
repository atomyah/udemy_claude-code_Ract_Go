package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/atyahara/sns-backend/internal/config"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 12:00:10 に固定した時計。ウィンドウは 12:00:00〜12:01:00 で、リセットまで 50 秒
var rateLimitBaseTime = time.Date(2026, 10, 11, 12, 0, 10, 0, time.UTC)

type fakeClock struct{ now time.Time }

func (f *fakeClock) Now() time.Time { return f.now }

// newRateLimitedServer は cmd/server/main.go と同じ構成（認証系は IP 単位、
// それ以外は OptionalJWTAuth の後にユーザーID / IP 単位）でルートを組んだ Echo を返す
func newRateLimitedServer(authLimit, generalLimit int, clock *fakeClock) *echo.Echo {
	e := echo.New()
	e.IPExtractor = ClientIPExtractor(0)

	authRL := newAuthRateLimit(authLimit, clock.Now)
	generalRL := newGeneralRateLimit(generalLimit, clock.Now)

	v1 := e.Group("/api/v1")
	auth := v1.Group("/auth")
	auth.POST("/login", okHandler, authRL)
	auth.POST("/register", okHandler, authRL)

	api := v1.Group("", OptionalJWTAuth(testSecret), generalRL)
	api.GET("/posts", okHandler)
	api.GET("/search/posts", okHandler)
	api.GET("/notifications", okHandler, JWTAuth(testSecret))
	return e
}

func doRateLimitedRequest(e *echo.Echo, method, path, ip, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.RemoteAddr = ip + ":54321"
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// sendN は同じリクエストを n 回送り、最後のレスポンスを返す。途中で 200 以外が返ったら失敗にする
func sendN(t *testing.T, e *echo.Echo, n int, method, path, ip, token string) *httptest.ResponseRecorder {
	t.Helper()
	var rec *httptest.ResponseRecorder
	for i := 1; i <= n; i++ {
		rec = doRateLimitedRequest(e, method, path, ip, token)
		require.Equalf(t, http.StatusOK, rec.Code, "%d 回目のリクエストが拒否された", i)
	}
	return rec
}

func TestRateLimit_AllowsRequestsWithinLimit(t *testing.T) {
	clock := &fakeClock{now: rateLimitBaseTime}
	e := newRateLimitedServer(5, 60, clock)
	resetAt := strconv.FormatInt(time.Date(2026, 10, 11, 12, 1, 0, 0, time.UTC).Unix(), 10)

	for i := 1; i <= 5; i++ {
		rec := doRateLimitedRequest(e, http.MethodPost, "/api/v1/auth/login", "203.0.113.1", "")
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "5", rec.Header().Get(HeaderRateLimitLimit))
		assert.Equal(t, strconv.Itoa(5-i), rec.Header().Get(HeaderRateLimitRemaining))
		assert.Equal(t, resetAt, rec.Header().Get(HeaderRateLimitReset))
		assert.Empty(t, rec.Header().Get(echo.HeaderRetryAfter), "制限内では Retry-After を付けない")
	}
}

func TestRateLimit_Returns429WhenLimitExceeded(t *testing.T) {
	clock := &fakeClock{now: rateLimitBaseTime}
	e := newRateLimitedServer(5, 60, clock)
	sendN(t, e, 5, http.MethodPost, "/api/v1/auth/login", "203.0.113.1", "")

	rec := doRateLimitedRequest(e, http.MethodPost, "/api/v1/auth/login", "203.0.113.1", "")

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "5", rec.Header().Get(HeaderRateLimitLimit))
	assert.Equal(t, "0", rec.Header().Get(HeaderRateLimitRemaining))
	assert.Equal(t, strconv.FormatInt(time.Date(2026, 10, 11, 12, 1, 0, 0, time.UTC).Unix(), 10), rec.Header().Get(HeaderRateLimitReset))
	assert.Equal(t, "50", rec.Header().Get(echo.HeaderRetryAfter), "次のウィンドウまでの秒数")
	body := decodeError(t, rec)
	assert.Equal(t, "RATE_LIMIT_EXCEEDED", body["code"])
	assert.Equal(t, "リクエストが多すぎます。しばらく待ってから再度お試しください", body["message"])

	// 超過後も同じウィンドウ内は拒否され続ける
	assert.Equal(t, http.StatusTooManyRequests, doRateLimitedRequest(e, http.MethodPost, "/api/v1/auth/login", "203.0.113.1", "").Code)
}

func TestRateLimit_RetryAfterIsRoundedUpToAtLeastOneSecond(t *testing.T) {
	// ウィンドウ終了の 0.3 秒前に超過した場合も Retry-After は 1 秒になる
	clock := &fakeClock{now: time.Date(2026, 10, 11, 12, 0, 59, 700_000_000, time.UTC)}
	e := newRateLimitedServer(1, 60, clock)
	sendN(t, e, 1, http.MethodPost, "/api/v1/auth/login", "203.0.113.1", "")

	rec := doRateLimitedRequest(e, http.MethodPost, "/api/v1/auth/login", "203.0.113.1", "")

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "1", rec.Header().Get(echo.HeaderRetryAfter))
}

// 非機能要件の値（認証系 5 回/分、一般 60 回/分）で、認証系と一般APIの上限が異なること
func TestRateLimit_AuthAndGeneralAPIsHaveDifferentLimits(t *testing.T) {
	clock := &fakeClock{now: rateLimitBaseTime}
	e := newRateLimitedServer(config.DefaultAuthRateLimit, config.DefaultGeneralRateLimit, clock)
	const ip = "203.0.113.1"

	// 認証系: 5 回まで通り、6 回目で 429
	rec := sendN(t, e, 5, http.MethodPost, "/api/v1/auth/login", ip, "")
	assert.Equal(t, "5", rec.Header().Get(HeaderRateLimitLimit))
	assert.Equal(t, http.StatusTooManyRequests, doRateLimitedRequest(e, http.MethodPost, "/api/v1/auth/login", ip, "").Code)

	// 一般: 同じ IP でも 6 回目以降も通り、60 回まで通って 61 回目で 429
	rec = sendN(t, e, 60, http.MethodGet, "/api/v1/posts", ip, "")
	assert.Equal(t, "60", rec.Header().Get(HeaderRateLimitLimit))
	assert.Equal(t, "0", rec.Header().Get(HeaderRateLimitRemaining))
	assert.Equal(t, http.StatusTooManyRequests, doRateLimitedRequest(e, http.MethodGet, "/api/v1/posts", ip, "").Code)
}

func TestRateLimit_LimitsAreCountedPerAPI(t *testing.T) {
	clock := &fakeClock{now: rateLimitBaseTime}
	e := newRateLimitedServer(2, 2, clock)
	const ip = "203.0.113.1"

	sendN(t, e, 2, http.MethodPost, "/api/v1/auth/login", ip, "")
	assert.Equal(t, http.StatusTooManyRequests, doRateLimitedRequest(e, http.MethodPost, "/api/v1/auth/login", ip, "").Code)
	// ログインを使い切っても登録は別枠
	sendN(t, e, 2, http.MethodPost, "/api/v1/auth/register", ip, "")

	sendN(t, e, 2, http.MethodGet, "/api/v1/posts", ip, "")
	assert.Equal(t, http.StatusTooManyRequests, doRateLimitedRequest(e, http.MethodGet, "/api/v1/posts", ip, "").Code)
	// タイムラインを使い切っても検索は別枠
	sendN(t, e, 2, http.MethodGet, "/api/v1/search/posts", ip, "")
}

func TestRateLimit_AuthAPIsAreLimitedPerIP(t *testing.T) {
	clock := &fakeClock{now: rateLimitBaseTime}
	e := newRateLimitedServer(2, 60, clock)
	token := makeToken(t, testSecret, uuid.NewString(), "user", rateLimitBaseTime.Add(time.Hour))

	sendN(t, e, 2, http.MethodPost, "/api/v1/auth/login", "203.0.113.1", "")
	// トークンを付けても認証系は IP 単位なので同じ IP からは拒否される
	assert.Equal(t, http.StatusTooManyRequests, doRateLimitedRequest(e, http.MethodPost, "/api/v1/auth/login", "203.0.113.1", token).Code)
	// 別の IP は別枠
	sendN(t, e, 2, http.MethodPost, "/api/v1/auth/login", "203.0.113.2", "")
}

func TestRateLimit_GeneralAPIsAreLimitedPerUserWhenAuthenticated(t *testing.T) {
	clock := &fakeClock{now: rateLimitBaseTime}
	e := newRateLimitedServer(5, 2, clock)
	exp := rateLimitBaseTime.Add(time.Hour)
	userA := makeToken(t, testSecret, uuid.NewString(), "user", exp)
	userB := makeToken(t, testSecret, uuid.NewString(), "user", exp)

	// 同じ IP でもユーザーが違えば別枠
	sendN(t, e, 2, http.MethodGet, "/api/v1/posts", "203.0.113.1", userA)
	assert.Equal(t, http.StatusTooManyRequests, doRateLimitedRequest(e, http.MethodGet, "/api/v1/posts", "203.0.113.1", userA).Code)
	sendN(t, e, 2, http.MethodGet, "/api/v1/posts", "203.0.113.1", userB)

	// 同じユーザーは IP を変えても同じ枠
	assert.Equal(t, http.StatusTooManyRequests, doRateLimitedRequest(e, http.MethodGet, "/api/v1/posts", "198.51.100.9", userA).Code)

	// 必須認証のAPIもユーザー単位
	sendN(t, e, 2, http.MethodGet, "/api/v1/notifications", "203.0.113.1", userA)
	assert.Equal(t, http.StatusTooManyRequests, doRateLimitedRequest(e, http.MethodGet, "/api/v1/notifications", "203.0.113.1", userA).Code)
}

func TestRateLimit_GeneralAPIsAreLimitedPerIPWhenUnauthenticated(t *testing.T) {
	clock := &fakeClock{now: rateLimitBaseTime}
	e := newRateLimitedServer(5, 2, clock)

	sendN(t, e, 2, http.MethodGet, "/api/v1/posts", "203.0.113.1", "")
	assert.Equal(t, http.StatusTooManyRequests, doRateLimitedRequest(e, http.MethodGet, "/api/v1/posts", "203.0.113.1", "").Code)
	// 無効なトークンは未認証扱いで、同じ IP の枠で数えられる
	assert.Equal(t, http.StatusTooManyRequests, doRateLimitedRequest(e, http.MethodGet, "/api/v1/posts", "203.0.113.1", "invalid-token").Code)
	// 別の IP は別枠
	sendN(t, e, 2, http.MethodGet, "/api/v1/posts", "203.0.113.2", "")
}

func TestRateLimit_InvalidTokenOnProtectedAPIIsStillRateLimited(t *testing.T) {
	clock := &fakeClock{now: rateLimitBaseTime}
	e := newRateLimitedServer(5, 2, clock)

	// 無効なトークンは 401 になるが、レート制限は JWT 検証より前に数える
	for i := 0; i < 2; i++ {
		rec := doRateLimitedRequest(e, http.MethodGet, "/api/v1/notifications", "203.0.113.1", "invalid-token")
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, "2", rec.Header().Get(HeaderRateLimitLimit))
	}
	assert.Equal(t, http.StatusTooManyRequests, doRateLimitedRequest(e, http.MethodGet, "/api/v1/notifications", "203.0.113.1", "invalid-token").Code)
}

func TestRateLimit_CountResetsInNextWindow(t *testing.T) {
	clock := &fakeClock{now: rateLimitBaseTime}
	e := newRateLimitedServer(2, 60, clock)
	sendN(t, e, 2, http.MethodPost, "/api/v1/auth/login", "203.0.113.1", "")
	assert.Equal(t, http.StatusTooManyRequests, doRateLimitedRequest(e, http.MethodPost, "/api/v1/auth/login", "203.0.113.1", "").Code)

	// 同じウィンドウの終わり（12:00:59.999）までは拒否
	clock.now = time.Date(2026, 10, 11, 12, 0, 59, 999_000_000, time.UTC)
	assert.Equal(t, http.StatusTooManyRequests, doRateLimitedRequest(e, http.MethodPost, "/api/v1/auth/login", "203.0.113.1", "").Code)

	// 次のウィンドウ（12:01:00）で再び上限まで通る
	clock.now = time.Date(2026, 10, 11, 12, 1, 0, 0, time.UTC)
	rec := sendN(t, e, 2, http.MethodPost, "/api/v1/auth/login", "203.0.113.1", "")
	assert.Equal(t, strconv.FormatInt(time.Date(2026, 10, 11, 12, 2, 0, 0, time.UTC).Unix(), 10), rec.Header().Get(HeaderRateLimitReset))
}

func TestFixedWindowLimiter_ConcurrentRequestsNeverExceedLimit(t *testing.T) {
	const limit, goroutines = 50, 200
	limiter := NewFixedWindowLimiter(limit, time.Minute, (&fakeClock{now: rateLimitBaseTime}).Now)

	var allowed atomic.Int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if limiter.Allow("same-key").Allowed {
				allowed.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()

	assert.Equal(t, int64(limit), allowed.Load(), "同時アクセスでも許可数はちょうど上限になる")
}

func TestFixedWindowLimiter_RemovesCountersOfPastWindows(t *testing.T) {
	clock := &fakeClock{now: rateLimitBaseTime}
	limiter := NewFixedWindowLimiter(5, time.Minute, clock.Now)
	limiter.Allow("a")
	limiter.Allow("b")
	assert.Len(t, limiter.counters, 2)

	clock.now = rateLimitBaseTime.Add(time.Minute)
	limiter.Allow("c")

	assert.Len(t, limiter.counters, 1, "前のウィンドウのカウンターは削除される")
	assert.Contains(t, limiter.counters, "c")
}

func TestClientIPExtractor(t *testing.T) {
	tests := []struct {
		name        string
		trustedHops int
		remoteAddr  string
		xff         []string
		want        string
	}{
		{"0段: X-Forwarded-For を無視して接続元を使う", 0, "10.0.0.5:1234", []string{"198.51.100.7"}, "10.0.0.5"},
		{"1段: 末尾（Cloud Run が追記した値）を使う", 1, "169.254.1.1:1234", []string{"198.51.100.7"}, "198.51.100.7"},
		{"1段: 先頭の偽装値は無視する", 1, "169.254.1.1:1234", []string{"1.2.3.4, 198.51.100.7"}, "198.51.100.7"},
		{"1段: 複数行のヘッダーは連結して扱う", 1, "169.254.1.1:1234", []string{"1.2.3.4", "198.51.100.7"}, "198.51.100.7"},
		{"2段: 末尾から2番目（外部LB構成）", 2, "169.254.1.1:1234", []string{"1.2.3.4, 198.51.100.7, 35.191.0.1"}, "198.51.100.7"},
		{"IPv6 も扱える", 1, "169.254.1.1:1234", []string{"2001:db8::1"}, "2001:db8::1"},
		{"ヘッダーなしは接続元にフォールバック", 1, "10.0.0.5:1234", nil, "10.0.0.5"},
		{"要素数不足は接続元にフォールバック", 2, "10.0.0.5:1234", []string{"198.51.100.7"}, "10.0.0.5"},
		{"IP でない値は接続元にフォールバック", 1, "10.0.0.5:1234", []string{"not-an-ip"}, "10.0.0.5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for _, v := range tt.xff {
				req.Header.Add(echo.HeaderXForwardedFor, v)
			}
			assert.Equal(t, tt.want, ClientIPExtractor(tt.trustedHops)(req))
		})
	}
}

func TestRateLimit_UsesClientIPFromForwardedHeader(t *testing.T) {
	// Cloud Run 構成（1段）では、先頭の偽装値を変えても末尾の実IPで数えられる
	clock := &fakeClock{now: rateLimitBaseTime}
	e := echo.New()
	e.IPExtractor = ClientIPExtractor(1)
	e.POST("/api/v1/auth/login", okHandler, newAuthRateLimit(1, clock.Now))

	send := func(spoofed string) int {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		req.RemoteAddr = "169.254.1.1:1234"
		req.Header.Set(echo.HeaderXForwardedFor, spoofed+", 198.51.100.7")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec.Code
	}

	assert.Equal(t, http.StatusOK, send("1.1.1.1"))
	assert.Equal(t, http.StatusTooManyRequests, send("2.2.2.2"))
}
