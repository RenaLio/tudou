package service

import (
	"net/http"
	"testing"

	tencentcodingplan "github.com/RenaLio/tudou/pkg/provider/platforms/tencent_coding_plan"
	ptypes "github.com/RenaLio/tudou/pkg/provider/types"
)

func TestBuildProvider_TencentCodingPlan(t *testing.T) {
	prov := buildProvider(tencentcodingplan.PlatformId, "", "test-key", http.DefaultClient)

	client, ok := prov.(*tencentcodingplan.Client)
	if !ok {
		t.Fatalf("unexpected provider type: %T", prov)
	}
	if client.Identifier() != tencentcodingplan.PlatformId {
		t.Fatalf("unexpected platform id: got=%q want=%q", client.Identifier(), tencentcodingplan.PlatformId)
	}
	if client.BaseURL != tencentcodingplan.DefaultBaseURL {
		t.Fatalf("unexpected base url: got=%q want=%q", client.BaseURL, tencentcodingplan.DefaultBaseURL)
	}
	if !client.HasAbility(ptypes.AbilityChatCompletions) {
		t.Fatalf("chat completions ability should be enabled")
	}
	if !client.HasAbility(ptypes.AbilityClaudeMessages) {
		t.Fatalf("claude messages ability should be enabled")
	}
}
