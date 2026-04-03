package openai

import (
	"github.com/RenaLio/tudou/pkg/provider/translator"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

func init() {
	translator.DefaultRegistry().RegisterRequest(types.FormatOpenAIResponses, types.FormatChatCompletion, ConvertResponseRequestToChatCompletion)
	translator.DefaultRegistry().RegisterResponse(types.FormatChatCompletion, types.FormatOpenAIResponses, ConvertOpenAIResponseToResponses)
}
