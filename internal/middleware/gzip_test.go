package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGzip_CompressesWhenClientAcceptsGzip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Gzip(gzip.DefaultCompression))
	r.GET("/data", func(c *gin.Context) {
		c.String(http.StatusOK, strings.Repeat("hello ", 100))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if got := w.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", got)
	}

	zr, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer zr.Close()
	body, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("read gzip body: %v", err)
	}
	want := strings.Repeat("hello ", 100)
	if string(body) != want {
		t.Fatalf("unexpected decompressed body")
	}
}

func TestGzip_SkipsWithoutAcceptEncoding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Gzip(gzip.DefaultCompression))
	r.GET("/data", func(c *gin.Context) { c.String(http.StatusOK, "hello") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding = %q, want empty", got)
	}
	if got := w.Body.String(); got != "hello" {
		t.Fatalf("body = %q, want hello", got)
	}
}

func TestGzip_SkipsEventStream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Gzip(gzip.DefaultCompression))
	r.GET("/stream", func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.String(http.StatusOK, "data: hello\n\n")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding = %q, want empty", got)
	}
	if got := w.Body.String(); got != "data: hello\n\n" {
		t.Fatalf("body = %q, want event stream body", got)
	}
}
