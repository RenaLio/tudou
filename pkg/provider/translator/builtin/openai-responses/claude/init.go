package claude

import (
	"github.com/RenaLio/tudou/pkg/provider/translator"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

func init() {
	translator.DefaultRegistry().RegisterRequest(types.FormatOpenAIResponses, types.FormatClaudeMessages, ConvertResponsesRequestToClaude)
	translator.DefaultRegistry().RegisterResponse(types.FormatClaudeMessages, types.FormatOpenAIResponses, ConvertClaudeResponseToOpenAIResponses)
}
