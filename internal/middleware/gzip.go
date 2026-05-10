package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Gzip compresses responses only when the client advertises gzip support.
// Streaming responses such as SSE are intentionally left uncompressed.
func Gzip(level int) gin.HandlerFunc {
	if level == 0 {
		level = gzip.DefaultCompression
	}
	return func(c *gin.Context) {
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		gz, err := gzip.NewWriterLevel(c.Writer, level)
		if err != nil {
			c.Next()
			return
		}
		writer := &gzipResponseWriter{ResponseWriter: c.Writer, gzipWriter: gz}
		defer func() {
			// Closing a gzip writer emits a footer, so only close it when compression was used.
			if writer.useGzip {
				_ = gz.Close()
			}
		}()

		c.Writer = writer
		c.Next()
	}
}

// gzipResponseWriter decides whether to compress at the moment headers are written,
// because handlers may set Content-Type after the middleware is entered.
type gzipResponseWriter struct {
	gin.ResponseWriter
	gzipWriter     *gzip.Writer
	useGzip        bool
	decisionMade   bool
	headersWritten bool
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	w.decide()
	if w.useGzip {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
		w.Header().Del("Content-Length")
	}
	w.headersWritten = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipResponseWriter) Write(data []byte) (int, error) {
	if !w.headersWritten {
		w.WriteHeader(http.StatusOK)
	}
	if !w.useGzip {
		return w.ResponseWriter.Write(data)
	}
	return w.gzipWriter.Write(data)
}

func (w *gzipResponseWriter) WriteString(s string) (int, error) {
	if !w.headersWritten {
		w.WriteHeader(http.StatusOK)
	}
	if !w.useGzip {
		return w.ResponseWriter.WriteString(s)
	}
	return w.gzipWriter.Write([]byte(s))
}

func (w *gzipResponseWriter) decide() {
	// Decide once so WriteHeader and WriteString/Write agree on compression behavior.
	if w.decisionMade {
		return
	}
	w.decisionMade = true

	if w.Header().Get("Content-Encoding") != "" {
		return
	}
	contentType := strings.ToLower(w.Header().Get("Content-Type"))
	if strings.HasPrefix(contentType, "text/event-stream") {
		return
	}
	w.useGzip = true
}
