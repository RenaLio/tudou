package httpclient

import (
	"net/http"
	"testing"
)

func TestBuildPoolKey_DiffersByDisableHTTP2(t *testing.T) {
	base := Config{Timeout: -1}
	k1 := buildPoolKey(normalizeConfigForPool(base))
	base.DisableHTTP2 = true
	k2 := buildPoolKey(normalizeConfigForPool(base))
	if k1 == k2 {
		t.Fatalf("expected different pool keys when DisableHTTP2 differs, got same key: %s", k1)
	}
}

func TestCreateClient_RespectDisableHTTP2(t *testing.T) {
	c1, err := createClient(Config{DisableHTTP2: false})
	if err != nil {
		t.Fatalf("create client failed: %v", err)
	}
	tr1, ok := c1.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	if !tr1.ForceAttemptHTTP2 {
		t.Fatal("expected ForceAttemptHTTP2=true when DisableHTTP2=false")
	}

	c2, err := createClient(Config{DisableHTTP2: true})
	if err != nil {
		t.Fatalf("create client failed: %v", err)
	}
	tr2, ok := c2.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	if tr2.ForceAttemptHTTP2 {
		t.Fatal("expected ForceAttemptHTTP2=false when DisableHTTP2=true")
	}
}
