package base

import "github.com/RenaLio/tudou/pkg/provider/types"

var defaultFormatPathMap = map[types.Format]string{
	types.FormatChatCompletion:   "/v1/chat/completions",
	types.FormatOpenAIResponses:  "/v1/responses",
	types.FormatClaudeMessages:   "/v1/messages",
	types.FormatOpenAIEmbeddings: "/v1/embeddings",
}
