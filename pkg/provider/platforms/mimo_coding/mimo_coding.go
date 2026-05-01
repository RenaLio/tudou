package mimocoding

import (
	"net/http"

	"github.com/RenaLio/tudou/pkg/provider/platforms/base"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

const PlatformId = "mimo_coding"

const DefaultBaseURL = "https://token-plan-cn.xiaomimimo.com"

var DefaultFormatPathMap = map[types.Format]string{
	types.FormatClaudeMessages: "/anthropic/v1/messages",
	types.FormatChatCompletion: "/v1/chat/completions",
}

var ModelList = []string{
	"mimo-v2.5-pro",
	"mimo-v2-pro",
	"mimo-v2.5",
	"mimo-v2-omni",
	"mimo-v2-flash",
	"mimo-v2.5-tts",
	"mimo-v2.5-tts-voiceclone",
	"mimo-v2.5-tts-voicedesign",
	"mimo-v2-tts",
}

type Client struct {
	*base.Client
}

func NewClient(httpC *http.Client, baseURL string, apiKey string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		Client: base.NewClient(
			httpC,
			baseURL,
			apiKey,
			PlatformId,
			[]types.Ability{types.AbilityClaudeMessages, types.AbilityChatCompletions},
			DefaultFormatPathMap,
		),
	}
}

func (c *Client) Models() ([]string, error) {
	return ModelList, nil
}
