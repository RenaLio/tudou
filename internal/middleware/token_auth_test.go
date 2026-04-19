package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/constants"
	"github.com/RenaLio/tudou/internal/middleware"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type fakeTokenLookup struct {
	token *v1.TokenWithRelationsResponse
	err   error
}

func (f *fakeTokenLookup) GetAvailableByToken(_ context.Context, _ string) (*v1.TokenWithRelationsResponse, error) {
	return f.token, f.err
}

func newTokenResp(id, userID, groupID int64, status models.TokenStatus) *v1.TokenWithRelationsResponse {
	return &v1.TokenWithRelationsResponse{
		TokenResponse: v1.TokenResponse{
			ID:      id,
			UserID:  userID,
			GroupID: groupID,
			Status:  status,
		},
	}
}

type captured struct {
	tokenID int64
	userID  int64
	groupID int64
}

func runMiddleware(t *testing.T, lookup middleware.TokenLookup, authHeader string) (*httptest.ResponseRecorder, *captured) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c := &captured{}

	r := gin.New()
	r.Use(middleware.RequireToken(lookup))
	r.GET("/ping", func(ctx *gin.Context) {
		if v, ok := ctx.Get(constants.TokenIdKey()); ok {
			if id, ok := v.(int64); ok {
				c.tokenID = id
			}
		}
		if v, ok := ctx.Get(constants.UserIdKey()); ok {
			if id, ok := v.(int64); ok {
				c.userID = id
			}
		}
		if v, ok := ctx.Get(constants.GroupIdKey()); ok {
			if id, ok := v.(int64); ok {
				c.groupID = id
			}
		}
		ctx.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	return w, c
}

func TestRequireToken_MissingHeader(t *testing.T) {
	w, _ := runMiddleware(t, &fakeTokenLookup{}, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireToken_WrongScheme(t *testing.T) {
	w, _ := runMiddleware(t, &fakeTokenLookup{}, "Basic abc")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireToken_EmptyToken(t *testing.T) {
	w, _ := runMiddleware(t, &fakeTokenLookup{}, "Bearer ")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireToken_LookupNotFound(t *testing.T) {
	lookup := &fakeTokenLookup{err: gorm.ErrRecordNotFound}
	w, _ := runMiddleware(t, lookup, "Bearer sk-does-not-exist")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireToken_LookupError(t *testing.T) {
	lookup := &fakeTokenLookup{err: errors.New("db boom")}
	w, _ := runMiddleware(t, lookup, "Bearer sk-abc")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestRequireToken_Disabled(t *testing.T) {
	lookup := &fakeTokenLookup{token: newTokenResp(1, 2, 3, models.TokenStatusDisabled)}
	w, _ := runMiddleware(t, lookup, "Bearer sk-disabled")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireToken_Success(t *testing.T) {
	lookup := &fakeTokenLookup{token: newTokenResp(100, 200, 300, models.TokenStatusEnabled)}
	w, c := runMiddleware(t, lookup, "Bearer sk-ok")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, strings.TrimSpace(w.Body.String()))
	}
	if c.tokenID != 100 {
		t.Errorf("expected token_id=100, got %d", c.tokenID)
	}
	if c.userID != 200 {
		t.Errorf("expected user_id=200, got %d", c.userID)
	}
	if c.groupID != 300 {
		t.Errorf("expected group_id=300, got %d", c.groupID)
	}
}
