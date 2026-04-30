package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/constants"
	"github.com/RenaLio/tudou/internal/service"
	ty2 "github.com/RenaLio/tudou/internal/types"
	"github.com/RenaLio/tudou/pkg/provider/types"
	"github.com/gin-gonic/gin"
)

type RelayService interface {
	GetTokenModels(ctx context.Context, tokenId int64, groupId int64) (*v1.RelayListResp[v1.RelayModelItemResp], error)
	FetchModel(ctx context.Context, req *v1.FetchModelRequest) ([]string, error)
	Forward(ctx context.Context, meta ty2.RelayMeta, body []byte, header http.Header) (*types.Response, error)
}

var _ RelayService = (*service.RelayService)(nil)

type RelayHandler struct {
	*Handler
	relaySvc RelayService
}

func NewRelayHandler(base *Handler, relaySvc RelayService) *RelayHandler {
	return &RelayHandler{
		Handler:  base,
		relaySvc: relaySvc,
	}
}

func (h *RelayHandler) RegisterRoutes(r gin.IRouter) {
	r.POST("/chat/completions", h.forward(types.FormatChatCompletion))
	r.POST("/messages", h.forward(types.FormatClaudeMessages))
	r.POST("/responses", h.forward(types.FormatOpenAIResponses))
	r.GET("/models", h.TokenModels)
}

func getTokenClaim(ctx *gin.Context) (*ty2.TokenClaim, error) {
	token, ok := ctx.Get(constants.TokenClaimKey())
	if !ok {
		return nil, errors.New("token claim not found")
	}
	value, ok := token.(*ty2.TokenClaim)
	if !ok {
		return nil, errors.New("token claim is not of type *ty2.TokenClaim")
	}
	return value, nil
}

func (h *RelayHandler) TokenModels(c *gin.Context) {
	tokenClaim, err := getTokenClaim(c)
	if err != nil {
		v1.Fail(c, v1.ErrUnauthorized.WithMessage("invalid token claim"), nil)
		return
	}
	resp, err := h.relaySvc.GetTokenModels(c.Request.Context(), tokenClaim.TokenId, tokenClaim.GroupId)
	if err != nil {
		v1.Fail(c, err, "")
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *RelayHandler) forward(format types.Format) gin.HandlerFunc {
	return func(ctx *gin.Context) {

		tokenClaim, err := getTokenClaim(ctx)
		if err != nil {
			v1.Fail(ctx, v1.ErrUnauthorized.WithMessage("invalid token claim"), nil)
			return
		}

		meta := ty2.RelayMeta{
			Format:    format,
			TokenID:   tokenClaim.TokenId,
			TokenName: tokenClaim.TokenName,
			UserID:    tokenClaim.UserId,
			GroupID:   tokenClaim.GroupId,
			GroupName: tokenClaim.GroupName,
			Strategy:  tokenClaim.Strategy,
		}

		body, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			v1.Fail(ctx, v1.ErrInternalServerError.WithMessage("failed to read request body"), nil)
			return
		}

		resp, err := h.relaySvc.Forward(ctx.Request.Context(), meta, body, ctx.Request.Header)
		if err != nil {
			h.handleError(ctx, err)
			return
		}

		if resp.IsStream && resp.Stream != nil {
			h.handleStreamResponse(ctx, resp)
			return
		}

		h.handleNonStreamResponse(ctx, resp)
	}
}

func (h *RelayHandler) handleNonStreamResponse(ctx *gin.Context, resp *types.Response) {
	for k, vals := range resp.Header {
		for _, v := range vals {
			ctx.Header(k, v)
		}
	}
	ctx.Data(resp.StatusCode, "application/json", resp.RawData)
}

func (h *RelayHandler) handleStreamResponse(ctx *gin.Context, resp *types.Response) {
	defer resp.Stream.Close()
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("Transfer-Encoding", "chunked")

	ctx.Status(resp.StatusCode)

	ctx.Stream(func(w io.Writer) bool {
		data, err := resp.Stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return false
			}
			fmt.Println("stream recv error:", err.Error())
			return false
		}
		//fmt.Println("stream recv data:", string(data.Content))
		if _, err := w.Write(data.Content); err != nil {
			fmt.Println("stream write error:", err.Error())
			return false
		}
		ctx.Writer.Flush()

		return !data.Finished
	})
}

func (h *RelayHandler) handleError(ctx *gin.Context, err error) {
	msg := err.Error()
	if strings.Contains(msg, "no available channel") {
		v1.Fail(ctx, v1.ErrServiceUnavailable.WithMessage("no available channel"), nil)
		return
	}
	if strings.Contains(msg, "missing model") {
		v1.Fail(ctx, v1.ErrBadRequest.WithMessage(msg), nil)
		return
	}
	v1.Fail(ctx, v1.ErrInternalServerError.WithMessage(msg), nil)
}

func toInt64(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	default:
		return 0
	}
}
