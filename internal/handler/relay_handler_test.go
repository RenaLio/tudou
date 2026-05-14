package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	ptypes "github.com/RenaLio/tudou/pkg/provider/types"
	"github.com/gin-gonic/gin"
)

func TestHandleNonStreamResponse_FiltersHopByHopHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	resp := &ptypes.Response{
		StatusCode: http.StatusAccepted,
		RawData:    []byte(`{"ok":true}`),
		Header: http.Header{
			"Content-Type":      {"application/problem+json"},
			"Content-Length":    {"999"},
			"Transfer-Encoding": {"chunked"},
			"Connection":        {"keep-alive"},
			"X-Upstream":        {"yes"},
		},
	}

	h := &RelayHandler{}
	h.handleNonStreamResponse(c, resp)

	if w.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: got=%d want=%d", w.Code, http.StatusAccepted)
	}
	if got := w.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("unexpected content type: got=%q want=%q", got, "application/problem+json")
	}
	if got := w.Header().Get("X-Upstream"); got != "yes" {
		t.Fatalf("unexpected upstream header: got=%q want=%q", got, "yes")
	}
	if got := w.Header().Get("Content-Length"); got == "999" {
		t.Fatalf("expected Content-Length to be recalculated instead of reusing upstream value, got=%q", got)
	}
	if got := w.Header().Get("Transfer-Encoding"); got != "" {
		t.Fatalf("expected Transfer-Encoding to be filtered, got=%q", got)
	}
	if got := w.Header().Get("Connection"); got != "" {
		t.Fatalf("expected Connection to be filtered, got=%q", got)
	}
	if body := w.Body.String(); body != `{"ok":true}` {
		t.Fatalf("unexpected body: got=%q want=%q", body, `{"ok":true}`)
	}
}
