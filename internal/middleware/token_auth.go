package middleware

import (
	"context"
	"errors"
	"strings"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/constants"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/types"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TokenLookup 抽象 Token 查询能力，便于测试注入。
// 生产实现由 service.TokenService 提供。
type TokenLookup interface {
	GetAvailableByToken(ctx context.Context, token string) (*v1.TokenWithRelationsResponse, error)
}

const bearerPrefix = "Bearer "

// RequireToken 解析 Authorization: Bearer <token>，查 Token 并注入 ctx。
// 失败时直接 abort；成功时注入 token_id / user_id / group_id 到 gin ctx 与 request context。
func RequireToken(lookup TokenLookup) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := strings.TrimSpace(ctx.GetHeader("Authorization"))
		if authHeader == "" {
			v1.Fail(ctx, v1.ErrUnauthorized.WithMessage("missing Authorization header"), nil)
			ctx.Abort()
			return
		}
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			v1.Fail(ctx, v1.ErrUnauthorized.WithMessage("Authorization must use Bearer scheme"), nil)
			ctx.Abort()
			return
		}
		tokenStr := strings.TrimSpace(strings.TrimPrefix(authHeader, bearerPrefix))
		if tokenStr == "" {
			v1.Fail(ctx, v1.ErrUnauthorized.WithMessage("empty token"), nil)
			ctx.Abort()
			return
		}

		token, err := lookup.GetAvailableByToken(ctx.Request.Context(), tokenStr)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				v1.Fail(ctx, v1.ErrUnauthorized.WithMessage("token not found or not available"), nil)
				ctx.Abort()
				return
			}
			v1.Fail(ctx, v1.ErrInternalServerError.WithMessage(err.Error()), nil)
			ctx.Abort()
			return
		}
		if token == nil {
			v1.Fail(ctx, v1.ErrUnauthorized.WithMessage("token not available"), nil)
			ctx.Abort()
			return
		}
		if token.Status != models.TokenStatusEnabled {
			v1.Fail(ctx, v1.ErrUnauthorized.WithMessage("token is "+string(token.Status)), nil)
			ctx.Abort()
			return
		}
		tokenClaim := &types.TokenClaim{
			TokenId:   token.ID,
			TokenName: token.Name,
			GroupId:   token.GroupID,
			GroupName: token.Group.Name,
			UserId:    token.UserID,
			Strategy:  string(token.LoadBalanceStrategy),
		}
		if tokenClaim.Strategy == "" {
			tokenClaim.Strategy = string(token.Group.LoadBalanceStrategy)
		}

		ctx.Set(constants.TokenClaimKey(), tokenClaim)

		reqCtx := context.WithValue(ctx.Request.Context(), constants.TokenClaimKey(), tokenClaim)

		ctx.Request = ctx.Request.WithContext(reqCtx)

		ctx.Next()
	}
}
