package handler

import (
	v1 "github.com/RenaLio/tudou/api/v1"
	ecloudcoding "github.com/RenaLio/tudou/pkg/provider/platforms/ecloud_coding"
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
					"paths": map[ptypes.Ability]string{
						ptypes.AbilityChatCompletions: "/v1/chat/completions",
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
				Value: "openai",
				Extra: map[string]any{
					"exampleBaseUrl": "https://api.openai.com",
					"paths": map[ptypes.Ability]string{
						ptypes.AbilityChatCompletions: "/v1/chat/completions",
						ptypes.AbilityResponses:       "/v1/responses",
					},
				},
			},
		},
	}
	v1.Success(c, options)
}
