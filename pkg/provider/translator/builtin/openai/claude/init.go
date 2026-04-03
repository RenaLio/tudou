package claude

import (
	"github.com/RenaLio/tudou/pkg/provider/translator"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

func init() {
	translator.DefaultRegistry().RegisterRequest(types.FormatChatCompletion, types.FormatClaudeMessages, ConvertChatCompletionRequestToClaudeRequest)
	translator.DefaultRegistry().RegisterResponse(types.FormatClaudeMessages, types.FormatChatCompletion, ConvertClaudeResponseToChatCompletion)
}
