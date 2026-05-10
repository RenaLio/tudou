package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimit_GlobalLimitRejectsSecondImmediateRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit(RateLimitConfig{
		Enabled:       true,
		GlobalEnabled: true,
		GlobalRPS:     1,
		GlobalBurst:   1,
	}))
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	first := performRateLimitRequest(r, "/ping", "1.1.1.1")
	if first.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want %d", first.Code, http.StatusOK)
	}

	second := performRateLimitRequest(r, "/ping", "1.1.1.1")
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want %d", second.Code, http.StatusTooManyRequests)
	}
}

func TestRateLimit_IPLimitIsPerClientIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit(RateLimitConfig{
		Enabled:   true,
		IPEnabled: true,
		IPRPS:     1,
		IPBurst:   1,
		IPTTL:     time.Minute,
	}))
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	first := performRateLimitRequest(r, "/ping", "1.1.1.1")
	if first.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want %d", first.Code, http.StatusOK)
	}

	sameIP := performRateLimitRequest(r, "/ping", "1.1.1.1")
	if sameIP.Code != http.StatusTooManyRequests {
		t.Fatalf("same IP status = %d, want %d", sameIP.Code, http.StatusTooManyRequests)
	}

	differentIP := performRateLimitRequest(r, "/ping", "2.2.2.2")
	if differentIP.Code != http.StatusOK {
		t.Fatalf("different IP status = %d, want %d", differentIP.Code, http.StatusOK)
	}
}

func TestRateLimit_DisabledAllowsRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit(RateLimitConfig{
		Enabled:       false,
		GlobalEnabled: true,
		GlobalRPS:     1,
		GlobalBurst:   1,
	}))
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	for i := 0; i < 3; i++ {
		resp := performRateLimitRequest(r, "/ping", "1.1.1.1")
		if resp.Code != http.StatusOK {
			t.Fatalf("request %d status = %d, want %d", i+1, resp.Code, http.StatusOK)
		}
	}
}

func performRateLimitRequest(r http.Handler, path string, ip string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.RemoteAddr = ip + ":12345"
	r.ServeHTTP(w, req)
	return w
}
