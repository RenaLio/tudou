package phelpers

import "github.com/RenaLio/tudou/pkg/provider/types"

var FormatAbilityMap = map[types.Format]types.Ability{
	types.FormatChatCompletion:         types.AbilityChatCompletions,
	types.FormatOpenAIResponses:        types.AbilityResponses,
	types.FormatClaudeMessages:         types.AbilityClaudeMessages,
	types.FormatOpenAIEmbeddings:       types.AbilityEmbeddings,
	types.FormatOpenAIResponsesCompact: types.AbilityResponsesCompact,
}

var AbilityFormatMap = map[types.Ability]types.Format{
	types.AbilityChatCompletions:  types.FormatChatCompletion,
	types.AbilityResponses:        types.FormatOpenAIResponses,
	types.AbilityClaudeMessages:   types.FormatClaudeMessages,
	types.AbilityEmbeddings:       types.FormatOpenAIEmbeddings,
	types.AbilityResponsesCompact: types.FormatOpenAIResponsesCompact,
}

func FormatToAbility(format types.Format) (types.Ability, bool) {
	ability, ok := FormatAbilityMap[format]
	return ability, ok
}

func AbilityToFormat(ability types.Ability) (types.Format, bool) {
	format, ok := AbilityFormatMap[ability]
	return format, ok
}
