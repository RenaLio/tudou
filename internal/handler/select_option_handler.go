package handler

import (
	v1 "github.com/RenaLio/tudou/api/v1"
	alibabacodingplancn "github.com/RenaLio/tudou/pkg/provider/platforms/alibaba_coding_plan_cn"
	baiducoding "github.com/RenaLio/tudou/pkg/provider/platforms/baidu_coding"
	codingplan "github.com/RenaLio/tudou/pkg/provider/platforms/coding_plan"
	ctyuncoding "github.com/RenaLio/tudou/pkg/provider/platforms/ctyuncoding"
	cucloudcoding "github.com/RenaLio/tudou/pkg/provider/platforms/cucloud_coding"
	ecloudcoding "github.com/RenaLio/tudou/pkg/provider/platforms/ecloud_coding"
	jdcoding "github.com/RenaLio/tudou/pkg/provider/platforms/jd_coding"
	kimiforcoding "github.com/RenaLio/tudou/pkg/provider/platforms/kimi_for_coding"
	mimocoding "github.com/RenaLio/tudou/pkg/provider/platforms/mimo_coding"
	minimaxcoding "github.com/RenaLio/tudou/pkg/provider/platforms/minimax_coding"
	"github.com/RenaLio/tudou/pkg/provider/platforms/openai"
	relaystation "github.com/RenaLio/tudou/pkg/provider/platforms/relay_station"
	tencentcodingplan "github.com/RenaLio/tudou/pkg/provider/platforms/tencent_coding_plan"
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
				Value: codingplan.PlatformId,
				Extra: map[string]any{
					"exampleBaseUrl": codingplan.DefaultBaseURL,
					"paths":          codingplan.DefaultFormatPathMap,
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
				Key:   "中转适配",
				Value: relaystation.PlatformId,
				Extra: map[string]any{
					"exampleBaseUrl": relaystation.DefaultBaseURL,
					"paths":          relaystation.DefaultFormatPathMap,
				},
			},
			{
				Key:   "Alibaba Coding CN",
				Value: alibabacodingplancn.PlatformId,
				Extra: map[string]any{
					"exampleBaseUrl": alibabacodingplancn.DefaultBaseURL,
					"paths":          alibabacodingplancn.DefaultFormatPathMap,
				},
			},
			{
				Key:   "Kimi For Coding",
				Value: kimiforcoding.PlatformId,
				Extra: map[string]any{
					"exampleBaseUrl": kimiforcoding.DefaultBaseURL,
					"paths":          kimiforcoding.DefaultFormatPathMap,
				},
			},
			{
				Key:   "Tencent Coding Plan",
				Value: tencentcodingplan.PlatformId,
				Extra: map[string]any{
					"exampleBaseUrl": tencentcodingplan.DefaultBaseURL,
					"paths":          tencentcodingplan.DefaultFormatPathMap,
				},
			},
			{
				Key:   "Baidu Coding",
				Value: baiducoding.PlatformId,
				Extra: map[string]any{
					"exampleBaseUrl": baiducoding.DefaultBaseURL,
					"paths":          baiducoding.DefaultFormatPathMap,
				},
			},
			{
				Key:   "联通云Coding",
				Value: cucloudcoding.PlatformId,
				Extra: map[string]any{
					"exampleBaseUrl": cucloudcoding.DefaultBaseURL,
					"paths":          cucloudcoding.DefaultFormatPathMap,
				},
			},
			{
				Key:   "天翼云Coding",
				Value: ctyuncoding.PlatformId,
				Extra: map[string]any{
					"exampleBaseUrl": ctyuncoding.DefaultBaseURL,
					"paths":          ctyuncoding.DefaultFormatPathMap,
				},
			},
			{
				Key:   "JD Coding",
				Value: jdcoding.PlatformId,
				Extra: map[string]any{
					"exampleBaseUrl": jdcoding.DefaultBaseURL,
					"paths":          jdcoding.DefaultFormatPathMap,
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
				Key:   "小米MimoPlan",
				Value: mimocoding.PlatformId,
				Extra: map[string]any{
					"exampleBaseUrl": mimocoding.DefaultBaseURL,
					"paths":          mimocoding.DefaultFormatPathMap,
				},
			},
			{
				Key:   "Minimax CodingPlan",
				Value: minimaxcoding.PlatformId,
				Extra: map[string]any{
					"exampleBaseUrl": minimaxcoding.DefaultBaseURL,
					"paths":          minimaxcoding.DefaultFormatPathMap,
				},
			},
		},
	}
	v1.Success(c, options)
}
