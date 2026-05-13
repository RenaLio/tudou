package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeaders_SetsDefaultProtections(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	r.ServeHTTP(w, req)

	assertHeader(t, w, "X-Content-Type-Options", "nosniff")
	assertHeader(t, w, "X-Frame-Options", "DENY")
	assertHeader(t, w, "X-XSS-Protection", "1; mode=block")
	assertHeader(t, w, "Referrer-Policy", "strict-origin-when-cross-origin")
	assertHeader(t, w, "Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	assertHeader(t, w, "Content-Security-Policy", "frame-ancestors 'none'")
}

func TestSecurityHeaders_AppliesBeforeCORSPreflightAbort(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(SecurityHeaders(), CORS())
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set("Origin", "https://example.test")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: got=%d want=%d", w.Code, http.StatusNoContent)
	}
	assertHeader(t, w, "X-Content-Type-Options", "nosniff")
	assertHeader(t, w, "X-Frame-Options", "DENY")
}

func assertHeader(t *testing.T, w *httptest.ResponseRecorder, key string, want string) {
	t.Helper()
	if got := w.Header().Get(key); got != want {
		t.Fatalf("unexpected %s: got=%q want=%q", key, got, want)
	}
}
