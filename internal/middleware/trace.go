package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/RenaLio/tudou/internal/constants"
	"github.com/RenaLio/tudou/internal/pkg/log"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const defaultRequestIDHeader = "X-Request-Id"

func RequestID(logger *log.Logger) gin.HandlerFunc {
	key := defaultRequestIDHeader

	return func(ctx *gin.Context) {
		requestID := strings.TrimSpace(ctx.GetHeader(key))
		if requestID == "" && !strings.EqualFold(key, defaultRequestIDHeader) {
			requestID = strings.TrimSpace(ctx.GetHeader(defaultRequestIDHeader))
		}
		if requestID == "" {
			requestID = newRequestID()
		}

		ctx.Set(constants.RequestIdKey(), requestID)
		reqCtx := context.WithValue(ctx.Request.Context(), constants.RequestIdKey(), requestID)
		logger.Inject(reqCtx, zap.String("request_id", requestID))
		ctx.Request = ctx.Request.WithContext(reqCtx)
		ctx.Header(key, requestID)

		ctx.Next()
	}
}

func newRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(buf)
}
