package openairesponses

import (
	"github.com/RenaLio/tudou/pkg/provider/translator"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

func init() {
	// 注册请求转换：Claude -> OpenAI Responses
	translator.DefaultRegistry().RegisterRequest(types.FormatClaudeMessages, types.FormatOpenAIResponses, ConvertClaudeRequestToResponses)
	// 注册响应转换：OpenAI Responses -> Claude
	translator.DefaultRegistry().RegisterResponse(types.FormatOpenAIResponses, types.FormatClaudeMessages, ConvertResponsesToClaudeResponse)
}
