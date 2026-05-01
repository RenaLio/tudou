package types

// Ability 能力类型，对外暴露的能力类型
type Ability string

const (
	// AbilityChatCompletions 原生支持的chat completions api
	AbilityChatCompletions = "chat.chatcompletions"
	AbilityResponses       = "responses"
	AbilityClaudeMessages  = "claude.messages"

	AbilityEmbeddings = "embeddings"
	AbilityRerank     = "rerank"
)
