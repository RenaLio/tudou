package openai

import (
	"github.com/RenaLio/tudou/pkg/provider/translator"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

func init() {
	translator.DefaultRegistry().RegisterRequest(types.FormatClaudeMessages, types.FormatChatCompletion, ConvertClaudeRequestToChatCompletionRequest)
	translator.DefaultRegistry().RegisterResponse(types.FormatChatCompletion, types.FormatClaudeMessages, ConvertChatCompletionResponseToClaudeResponse)
}
