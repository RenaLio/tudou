package middleware

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

const (
	defaultGlobalRPS   = 1000
	defaultGlobalBurst = 2000
	defaultIPRPS       = 100
	defaultIPBurst     = 200
	defaultIPTTL       = 10 * time.Minute
)

// RateLimitConfig controls the process-wide limiter and the per-client-IP limiter.
// RPS values are requests per second; Burst values are token bucket burst sizes.
type RateLimitConfig struct {
	Enabled       bool
	GlobalEnabled bool
	GlobalRPS     float64
	GlobalBurst   int
	IPEnabled     bool
	IPRPS         float64
	IPBurst       int
	IPTTL         time.Duration
}

// RateLimit applies per-IP limiting and process-wide limiting to every request.
// It rejects immediately with HTTP 429 instead of blocking while waiting for tokens.
func RateLimit(cfg RateLimitConfig) gin.HandlerFunc {
	cfg = normalizeRateLimitConfig(cfg)
	if !cfg.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	var global *rate.Limiter
	if cfg.GlobalEnabled {
		global = rate.NewLimiter(rate.Limit(cfg.GlobalRPS), cfg.GlobalBurst)
	}

	ipLimiter := newIPLimiter(cfg.IPRPS, cfg.IPBurst, cfg.IPTTL)

	return func(c *gin.Context) {
		if cfg.IPEnabled && !ipLimiter.allow(c.ClientIP()) {
			abortRateLimited(c)
			return
		}
		if global != nil && !global.Allow() {
			abortRateLimited(c)
			return
		}
		c.Next()
	}
}

func normalizeRateLimitConfig(cfg RateLimitConfig) RateLimitConfig {
	if cfg.GlobalRPS <= 0 {
		cfg.GlobalRPS = defaultGlobalRPS
	}
	if cfg.GlobalBurst <= 0 {
		cfg.GlobalBurst = defaultGlobalBurst
	}
	if cfg.IPRPS <= 0 {
		cfg.IPRPS = defaultIPRPS
	}
	if cfg.IPBurst <= 0 {
		cfg.IPBurst = defaultIPBurst
	}
	if cfg.IPTTL <= 0 {
		cfg.IPTTL = defaultIPTTL
	}
	return cfg
}

func abortRateLimited(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusTooManyRequests, v1.Response{
		Code:    http.StatusTooManyRequests,
		Message: "rate limit exceeded",
		Data:    map[string]any{},
	})
}

// ipLimiter stores one rate.Limiter per client IP. sync.Map avoids a global map lock
// on the hot path; timestamps are atomic so cleanup can run opportunistically.
type ipLimiter struct {
	rate        rate.Limit
	burst       int
	ttl         time.Duration
	lastCleanup atomic.Int64
	clients     sync.Map
}

type ipClientLimiter struct {
	limiter      *rate.Limiter
	lastSeenUnix atomic.Int64
}

func newIPLimiter(rps float64, burst int, ttl time.Duration) *ipLimiter {
	l := &ipLimiter{
		rate:  rate.Limit(rps),
		burst: burst,
		ttl:   ttl,
	}
	l.lastCleanup.Store(time.Now().UnixNano())
	return l
}

func (l *ipLimiter) allow(ip string) bool {
	now := time.Now()
	l.cleanup(now)
	client := l.getClient(ip, now)
	return client.limiter.Allow()
}

func (l *ipLimiter) getClient(ip string, now time.Time) *ipClientLimiter {
	// LoadOrStore may discard newClient when another goroutine created the same IP first.
	newClient := &ipClientLimiter{limiter: rate.NewLimiter(l.rate, l.burst)}
	newClient.lastSeenUnix.Store(now.UnixNano())

	actual, _ := l.clients.LoadOrStore(ip, newClient)
	client := actual.(*ipClientLimiter)
	client.lastSeenUnix.Store(now.UnixNano())
	return client
}

func (l *ipLimiter) cleanup(now time.Time) {
	// Cleanup is lazy and best-effort: the first request after ttl wins the CAS and
	// scans stale IP entries. Other concurrent requests skip the scan.
	lastCleanup := time.Unix(0, l.lastCleanup.Load())
	if now.Sub(lastCleanup) < l.ttl {
		return
	}
	if !l.lastCleanup.CompareAndSwap(lastCleanup.UnixNano(), now.UnixNano()) {
		return
	}
	l.clients.Range(func(key, value any) bool {
		client := value.(*ipClientLimiter)
		lastSeen := time.Unix(0, client.lastSeenUnix.Load())
		if now.Sub(lastSeen) >= l.ttl {
			l.clients.Delete(key)
		}
		return true
	})
}
