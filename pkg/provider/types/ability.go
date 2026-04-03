package types

// Ability 能力类型，对外暴露的能力类型
type Ability string

const (
	// AbilityChat 所有chat能力的基础，说明原生至少实现了一个chat相关的api
	AbilityChat Ability = "chat"
	// AbilityChatCompletions 原生支持的chat completions api
	AbilityChatCompletions = "chat.chatcompletions"
	AbilityResponses       = "responses"
	AbilityClaudeMessages  = "claude.messages"

	AbilityEmbeddings = "embeddings"
	AbilityRerank     = "rerank"
)
