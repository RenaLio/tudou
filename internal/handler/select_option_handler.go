package handler

import (
	v1 "github.com/RenaLio/tudou/api/v1"
	ecloudcoding "github.com/RenaLio/tudou/pkg/provider/platforms/ecloud_coding"
	mimocoding "github.com/RenaLio/tudou/pkg/provider/platforms/mimo_coding"
	"github.com/RenaLio/tudou/pkg/provider/platforms/openai"
	ptypes "github.com/RenaLio/tudou/pkg/provider/types"
	"github.com/gin-gonic/gin"
)

type SelectOptionHandler struct {
	*Handler
}

func NewSelectOptionHandler(handler *Handler) *SelectOptionHandler {
	return &SelectOptionHandler{
		Handler: handler,
	}
}

func (h *SelectOptionHandler) RegisterRoutes(r gin.IRouter) {
	group := r.Group("/select-option")
	group.GET("/platform_options", h.PlatformOptions)
}

func (h *Handler) PlatformOptions(c *gin.Context) {
	options := v1.SelectOptions[string, string]{
		Options: []v1.SelectOptionItem[string, string]{
			{
				Key:   "ChatCompletion兼容",
				Value: "chat-completion",
				Extra: map[string]any{
					"exampleBaseUrl": "https://api.example.com",
					"paths": map[ptypes.Format]string{
						ptypes.FormatChatCompletion: "/v1/chat/completions",
					},
				},
			},
			{
				Key:   "CodingPlan兼容",
				Value: "coding-plan-adapter",
				Extra: map[string]any{
					"exampleBaseUrl": "https://api.example.com/v1",
					"paths": map[ptypes.Format]string{
						ptypes.FormatChatCompletion:  "/chat/completions",
						ptypes.AbilityClaudeMessages: "/messages",
					},
				},
			},
			{
				Key:   "移动云Coding",
				Value: ecloudcoding.PlatformId,
				Extra: map[string]any{
					"exampleBaseUrl": ecloudcoding.DefaultBaseURL,
					"paths":          ecloudcoding.DefaultFormatPathMap,
				},
			},
			{
				Key:   "OpenAI",
				Value: openai.PlatformId,
				Extra: map[string]any{
					"exampleBaseUrl": openai.DefaultBaseURL,
					"paths":          openai.DefaultFormatPathMap,
				},
			},
			{
				Key:   "小米MimoPlan",
				Value: mimocoding.PlatformId,
				Extra: map[string]any{
					"exampleBaseUrl": mimocoding.DefaultBaseURL,
					"paths":          mimocoding.DefaultFormatPathMap,
				},
			},
		},
	}
	v1.Success(c, options)
}
