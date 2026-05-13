package http

import (
	"context"
	"testing"

	ilog "github.com/RenaLio/tudou/internal/pkg/log"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestServerStart_UsesDefaultHTTPServerTimeouts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	srv := NewServer(
		gin.New(),
		&ilog.Logger{Logger: zap.NewNop()},
		WithServerHost("127.0.0.1"),
		WithServerPort(0),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := srv.Start(ctx)
	if err == nil {
		t.Fatal("expected Start to return context cancellation")
	}
	if srv.httpSrv == nil {
		t.Fatal("expected http server to be initialized")
	}
	if srv.httpSrv.ReadHeaderTimeout != defaultReadHeaderTimeout {
		t.Fatalf("unexpected ReadHeaderTimeout: got=%s want=%s", srv.httpSrv.ReadHeaderTimeout, defaultReadHeaderTimeout)
	}
	if srv.httpSrv.IdleTimeout != defaultIdleTimeout {
		t.Fatalf("unexpected IdleTimeout: got=%s want=%s", srv.httpSrv.IdleTimeout, defaultIdleTimeout)
	}
	if srv.httpSrv.MaxHeaderBytes != defaultMaxHeaderBytes {
		t.Fatalf("unexpected MaxHeaderBytes: got=%d want=%d", srv.httpSrv.MaxHeaderBytes, defaultMaxHeaderBytes)
	}
}
