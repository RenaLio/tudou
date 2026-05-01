package helpers

import (
	"github.com/RenaLio/tudou/pkg/provider/translator/models/claude"
	"github.com/RenaLio/tudou/pkg/provider/translator/models/openai"
)

// ChatUsageToResponseUsage converts Chat Completions API usage to Responses API usage
//
// Parameter:
//   - usage - The ChatCompletionUsage from Chat Completions API
//
// Returns:
//   - The ResponseUsage for Responses API
//
// Note:
//   - Maps PromptTokens to InputTokens, CompletionTokens to OutputTokens
//   - Calculates TextTokens by subtracting AudioTokens and CachedTokens from PromptTokens
//   - Calculates Output TextTokens by subtracting AudioTokens, ReasoningTokens, and prediction tokens
func ChatUsageToResponseUsage(usage *openai.ChatCompletionUsage) *openai.ResponseUsage {
	if usage == nil {
		return nil
	}

	responseUsage := &openai.ResponseUsage{
		InputTokens:  usage.PromptTokens,
		OutputTokens: usage.CompletionTokens,
		TotalTokens:  usage.TotalTokens,
	}

	if usage.PromptTokensDetails != nil {
		responseUsage.InputTokenDetails = &openai.InputTokenDetails{
			CachedTokens: usage.PromptTokensDetails.CachedTokens,
		}
	}
	if usage.CompletionTokensDetails != nil {
		responseUsage.OutputTokenDetails = &openai.OutputTokenDetails{
			ReasoningTokens: usage.CompletionTokensDetails.ReasoningTokens,
		}
	}

	return responseUsage
}

// ResponseUsageToChatUsage converts Responses API usage to Chat Completions API usage
//
// Parameter:
//   - usage - The ResponseUsage from Responses API
//
// Returns:
//   - The ChatCompletionUsage for Chat Completions API
//
// Note:
//   - Maps InputTokens to PromptTokens, OutputTokens to CompletionTokens
//   - Preserves CachedTokens and AudioTokens in PromptTokensDetails
//   - Preserves ReasoningTokens and AudioTokens in CompletionTokensDetails
//   - AcceptedPredictionTokens and RejectedPredictionTokens are not available in ResponseUsage
func ResponseUsageToChatUsage(usage *openai.ResponseUsage) *openai.ChatCompletionUsage {
	if usage == nil {
		return nil
	}

	chatUsage := &openai.ChatCompletionUsage{
		PromptTokens:     usage.InputTokens,
		CompletionTokens: usage.OutputTokens,
		TotalTokens:      usage.TotalTokens,
	}

	if usage.InputTokenDetails != nil {
		chatUsage.PromptTokensDetails = &openai.PromptTokensDetails{
			CachedTokens: usage.InputTokenDetails.CachedTokens,
		}
	}

	if usage.OutputTokenDetails != nil {
		chatUsage.CompletionTokensDetails = &openai.CompletionTokensDetails{
			ReasoningTokens: usage.OutputTokenDetails.ReasoningTokens,
		}
	}

	return chatUsage
}

// ClaudeUsageToChatUsage converts Claude API usage to Chat Completions API usage
//
// Parameter:
//   - usage - The Usage from Claude API
//
// Returns:
//   - The ChatCompletionUsage for Chat Completions API
//
// Note:
//   - Maps InputTokens to PromptTokens, OutputTokens to CompletionTokens
//   - TotalTokens is calculated as InputTokens + OutputTokens
//   - Maps CacheCreationInputTokens to CachedTokens in PromptTokensDetails
func ClaudeUsageToChatUsage(usage *claude.Usage) *openai.ChatCompletionUsage {
	if usage == nil {
		return nil
	}
	chatUsage := &openai.ChatCompletionUsage{
		PromptTokens:     usage.InputTokens,
		CompletionTokens: usage.OutputTokens,
	}
	var cachedTokens int64

	if usage.CacheReadInputTokens != nil {
		chatUsage.PromptTokens += *usage.CacheReadInputTokens
		cachedTokens += *usage.CacheReadInputTokens
	}
	if usage.CacheCreationInputTokens != nil {
		chatUsage.PromptTokens += *usage.CacheCreationInputTokens
		cachedTokens += *usage.CacheCreationInputTokens
	}
	chatUsage.PromptTokensDetails = &openai.PromptTokensDetails{
		CachedTokens: cachedTokens,
	}
	chatUsage.TotalTokens = chatUsage.PromptTokens + chatUsage.CompletionTokens
	return chatUsage
}

// ChatUsageToClaudeUsage converts Chat Completions API usage to Claude API usage
//
// Parameter:
//   - usage - The ChatCompletionUsage from Chat Completions API
//
// Returns:
//   - The Usage for Claude API
//
// Note:
//   - Maps PromptTokens to InputTokens, CompletionTokens to OutputTokens
//   - Maps CachedTokens from PromptTokensDetails to CacheCreationInputTokens
//   - CacheReadInputTokens is not available in ChatCompletionUsage
//   - CacheCreation field is not populated as it requires additional context
func ChatUsageToClaudeUsage(usage *openai.ChatCompletionUsage) *claude.Usage {
	if usage == nil {
		return nil
	}
	claudeUsage := &claude.Usage{
		InputTokens:  usage.PromptTokens,
		OutputTokens: usage.CompletionTokens,
	}
	if usage.PromptTokensDetails != nil && usage.PromptTokensDetails.CachedTokens > 0 {
		cachedTokens := usage.PromptTokensDetails.CachedTokens
		claudeUsage.CacheReadInputTokens = &cachedTokens
	}
	if claudeUsage.CacheReadInputTokens != nil {
		claudeUsage.InputTokens = claudeUsage.InputTokens - *claudeUsage.CacheReadInputTokens
	}
	return claudeUsage
}

// ResponseUsageToClaudeUsage converts Responses API usage to Claude API usage
//
// Parameter:
//   - usage - The ResponseUsage from Responses API
//
// Returns:
//   - The Usage for Claude API
//
// Note:
//   - Maps InputTokens to InputTokens, OutputTokens to OutputTokens
//   - Maps CachedTokens from InputTokenDetails to CacheCreationInputTokens
//   - CacheReadInputTokens is not available in ResponseUsage
//   - CacheCreation field is not populated as it requires additional context
func ResponseUsageToClaudeUsage(usage *openai.ResponseUsage) *claude.Usage {
	if usage == nil {
		return nil
	}
	claudeUsage := &claude.Usage{
		InputTokens:  usage.InputTokens,
		OutputTokens: usage.OutputTokens,
	}
	if usage.InputTokenDetails != nil && usage.InputTokenDetails.CachedTokens > 0 {
		cachedTokens := usage.InputTokenDetails.CachedTokens
		claudeUsage.CacheReadInputTokens = &cachedTokens
	}
	if claudeUsage.CacheReadInputTokens != nil {
		claudeUsage.InputTokens = claudeUsage.InputTokens - *claudeUsage.CacheReadInputTokens
	}
	return claudeUsage
}

// ClaudeUsageToResponseUsage converts Claude API usage to Responses API usage
//
// Parameter:
//   - usage - The Usage from Claude API
//
// Returns:
//   - The ResponseUsage for Responses API
//
// Note:
//   - Maps InputTokens to InputTokens, OutputTokens to OutputTokens
//   - TotalTokens is calculated as InputTokens + OutputTokens
//   - Maps CacheCreationInputTokens to CachedTokens in InputTokenDetails
//   - TextTokens is calculated by subtracting CachedTokens and AudioTokens from InputTokens
func ClaudeUsageToResponseUsage(usage *claude.Usage) *openai.ResponseUsage {
	if usage == nil {
		return nil
	}
	responseUsage := &openai.ResponseUsage{
		InputTokens:  usage.InputTokens,
		OutputTokens: usage.OutputTokens,
	}
	cachedTokens := int64(0)
	if usage.CacheReadInputTokens != nil {
		responseUsage.InputTokens += *usage.CacheReadInputTokens
		cachedTokens += *usage.CacheReadInputTokens
	}
	if usage.CacheCreationInputTokens != nil {
		responseUsage.InputTokens += *usage.CacheCreationInputTokens
		cachedTokens += *usage.CacheCreationInputTokens
	}
	responseUsage.InputTokenDetails = &openai.InputTokenDetails{
		CachedTokens: cachedTokens,
	}
	responseUsage.TotalTokens = responseUsage.InputTokens + responseUsage.OutputTokens
	return responseUsage
}

// MergeOpenAIUsage merges two ChatCompletionUsage by taking the maximum value of each field
//
// Parameters:
//   - usage1 - The first ChatCompletionUsage
//   - usage2 - The second ChatCompletionUsage
//
// Returns:
//   - A new ChatCompletionUsage with maximum values from both inputs
//
// Note:
//   - If both inputs are nil, returns nil
//   - If one input is nil, returns the non-nil one directly
//   - For nested details objects, recursively takes maximum values
func MergeOpenAIUsage(usage1 *openai.ChatCompletionUsage, usage2 *openai.ChatCompletionUsage) *openai.ChatCompletionUsage {
	if usage1 == nil && usage2 == nil {
		return nil
	}
	if usage1 == nil {
		return usage2
	}
	if usage2 == nil {
		return usage1
	}

	result := &openai.ChatCompletionUsage{
		CompletionTokens: max(usage1.CompletionTokens, usage2.CompletionTokens),
		PromptTokens:     max(usage1.PromptTokens, usage2.PromptTokens),
		TotalTokens:      max(usage1.TotalTokens, usage2.TotalTokens),
	}

	// Merge CompletionTokensDetails
	if usage1.CompletionTokensDetails != nil || usage2.CompletionTokensDetails != nil {
		result.CompletionTokensDetails = mergeCompletionTokensDetails(usage1.CompletionTokensDetails, usage2.CompletionTokensDetails)
	}

	// Merge PromptTokensDetails
	if usage1.PromptTokensDetails != nil || usage2.PromptTokensDetails != nil {
		result.PromptTokensDetails = mergePromptTokensDetails(usage1.PromptTokensDetails, usage2.PromptTokensDetails)
	}

	return result
}

// mergeCompletionTokensDetails merges two CompletionTokensDetails by taking maximum values
func mergeCompletionTokensDetails(details1, details2 *openai.CompletionTokensDetails) *openai.CompletionTokensDetails {
	if details1 == nil && details2 == nil {
		return nil
	}
	if details1 == nil {
		return details2
	}
	if details2 == nil {
		return details1
	}
	return &openai.CompletionTokensDetails{
		AudioTokens:              max(details1.AudioTokens, details2.AudioTokens),
		ReasoningTokens:          max(details1.ReasoningTokens, details2.ReasoningTokens),
		AcceptedPredictionTokens: max(details1.AcceptedPredictionTokens, details2.AcceptedPredictionTokens),
		RejectedPredictionTokens: max(details1.RejectedPredictionTokens, details2.RejectedPredictionTokens),
	}
}

// mergePromptTokensDetails merges two PromptTokensDetails by taking maximum values
func mergePromptTokensDetails(details1, details2 *openai.PromptTokensDetails) *openai.PromptTokensDetails {
	if details1 == nil && details2 == nil {
		return nil
	}
	if details1 == nil {
		return details2
	}
	if details2 == nil {
		return details1
	}
	return &openai.PromptTokensDetails{
		AudioTokens:  max(details1.AudioTokens, details2.AudioTokens),
		CachedTokens: max(details1.CachedTokens, details2.CachedTokens),
	}
}

// MergeClaudeUsage merges two Claude Usage by taking the maximum value of each field
//
// Parameters:
//   - usage1 - The first Usage
//   - usage2 - The second Usage
//
// Returns:
//   - A new Usage with maximum values from both inputs
//
// Note:
//   - If both inputs are nil, returns nil
//   - If one input is nil, returns the non-nil one directly
//   - For pointer fields, takes maximum of dereferenced values
func MergeClaudeUsage(usage1 *claude.Usage, usage2 *claude.Usage) *claude.Usage {
	if usage1 == nil && usage2 == nil {
		return nil
	}
	if usage1 == nil {
		return usage2
	}
	if usage2 == nil {
		return usage1
	}

	result := &claude.Usage{
		InputTokens:  max(usage1.InputTokens, usage2.InputTokens),
		OutputTokens: max(usage1.OutputTokens, usage2.OutputTokens),
	}

	// Merge CacheCreationInputTokens
	if usage1.CacheCreationInputTokens != nil || usage2.CacheCreationInputTokens != nil {
		result.CacheCreationInputTokens = ptrMax(usage1.CacheCreationInputTokens, usage2.CacheCreationInputTokens)
	}

	// Merge CacheReadInputTokens
	if usage1.CacheReadInputTokens != nil || usage2.CacheReadInputTokens != nil {
		result.CacheReadInputTokens = ptrMax(usage1.CacheReadInputTokens, usage2.CacheReadInputTokens)
	}

	return result
}

// ptrMax returns the maximum of two int64 pointers
// If one is nil, returns the other. If both are nil, returns nil.
func ptrMax(p1, p2 *int64) *int64 {
	if p1 == nil && p2 == nil {
		return nil
	}
	if p1 == nil {
		return p2
	}
	if p2 == nil {
		return p1
	}
	maxVal := max(*p1, *p2)
	return &maxVal
}

// MergeResponseUsage merges two ResponseUsage by taking the maximum value of each field
//
// Parameters:
//   - usage1 - The first ResponseUsage
//   - usage2 - The second ResponseUsage
//
// Returns:
//   - A new ResponseUsage with maximum values from both inputs
//
// Note:
//   - If both inputs are nil, returns nil
//   - If one input is nil, returns the non-nil one directly
//   - For nested details objects, recursively takes maximum values
func MergeResponseUsage(usage1 *openai.ResponseUsage, usage2 *openai.ResponseUsage) *openai.ResponseUsage {
	if usage1 == nil && usage2 == nil {
		return nil
	}
	if usage1 == nil {
		return usage2
	}
	if usage2 == nil {
		return usage1
	}

	result := &openai.ResponseUsage{
		InputTokens:  max(usage1.InputTokens, usage2.InputTokens),
		OutputTokens: max(usage1.OutputTokens, usage2.OutputTokens),
		TotalTokens:  max(usage1.TotalTokens, usage2.TotalTokens),
	}

	// Merge InputTokenDetails
	if usage1.InputTokenDetails != nil || usage2.InputTokenDetails != nil {
		result.InputTokenDetails = mergeInputTokenDetails(usage1.InputTokenDetails, usage2.InputTokenDetails)
	}

	// Merge OutputTokenDetails
	if usage1.OutputTokenDetails != nil || usage2.OutputTokenDetails != nil {
		result.OutputTokenDetails = mergeOutputTokenDetails(usage1.OutputTokenDetails, usage2.OutputTokenDetails)
	}

	return result
}

// mergeInputTokenDetails merges two InputTokenDetails by taking maximum values
func mergeInputTokenDetails(details1, details2 *openai.InputTokenDetails) *openai.InputTokenDetails {
	if details1 == nil && details2 == nil {
		return nil
	}
	if details1 == nil {
		return details2
	}
	if details2 == nil {
		return details1
	}
	return &openai.InputTokenDetails{
		CachedTokens: max(details1.CachedTokens, details2.CachedTokens),
	}
}

// mergeOutputTokenDetails merges two OutputTokenDetails by taking maximum values
func mergeOutputTokenDetails(details1, details2 *openai.OutputTokenDetails) *openai.OutputTokenDetails {
	if details1 == nil && details2 == nil {
		return nil
	}
	if details1 == nil {
		return details2
	}
	if details2 == nil {
		return details1
	}
	return &openai.OutputTokenDetails{
		ReasoningTokens: max(details1.ReasoningTokens, details2.ReasoningTokens),
	}
}
