package types

type Format string

const (
	FormatChatCompletion  Format = "chat.completion"
	FormatOpenAIResponses Format = "openai.responses"
	FormatClaudeMessages  Format = "claude.messages"

	FormatOpenAIEmbeddings Format = "openai.embeddings"
)
