package middleware

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/constants"
	"github.com/RenaLio/tudou/internal/pkg/jwt"
	"github.com/gin-gonic/gin"
)

func RequireAuth(jwt *jwt.JWT) gin.HandlerFunc {

	return func(ctx *gin.Context) {

		authHeader := strings.TrimSpace(ctx.GetHeader("Authorization"))
		if authHeader == "" {
			v1.Fail(ctx, v1.ErrUnauthorized.WithMessage("missing Authorization header"), nil)
			ctx.Abort()
			return
		}

		claims, err := jwt.ParseToken(authHeader)
		if err != nil {
			fmt.Println("Error parsing token:", err)
			v1.Fail(ctx, v1.ErrUnauthorized.WithMessage("invalid token"), nil)
			ctx.Abort()
			return
		}

		userID, err := strconv.ParseInt(strings.TrimSpace(claims.UserId), 10, 64)
		if err != nil || userID <= 0 {
			v1.Fail(ctx, v1.ErrUnauthorized.WithMessage("invalid token user id"), nil)
			ctx.Abort()
			return
		}

		ctx.Set(constants.ClaimsKey(), claims)
		ctx.Set(constants.UserIdKey(), userID)

		reqCtx := context.WithValue(ctx.Request.Context(), constants.ClaimsKey(), claims)
		reqCtx = context.WithValue(reqCtx, constants.UserIdKey(), userID)
		ctx.Request = ctx.Request.WithContext(reqCtx)

		ctx.Next()
	}
}
