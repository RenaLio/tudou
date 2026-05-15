package service

import (
	"net/http"
	"testing"

	ctyuncoding "github.com/RenaLio/tudou/pkg/provider/platforms/ctyuncoding"
	cucloudcoding "github.com/RenaLio/tudou/pkg/provider/platforms/cucloud_coding"
	siliconflow "github.com/RenaLio/tudou/pkg/provider/platforms/siliconflow"
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

func TestBuildProvider_CTYunCoding(t *testing.T) {
	prov := buildProvider(ctyuncoding.PlatformId, "", "test-key", http.DefaultClient)

	client, ok := prov.(*ctyuncoding.Client)
	if !ok {
		t.Fatalf("unexpected provider type: %T", prov)
	}
	if client.Identifier() != ctyuncoding.PlatformId {
		t.Fatalf("unexpected platform id: got=%q want=%q", client.Identifier(), ctyuncoding.PlatformId)
	}
	if client.BaseURL != ctyuncoding.DefaultBaseURL {
		t.Fatalf("unexpected base url: got=%q want=%q", client.BaseURL, ctyuncoding.DefaultBaseURL)
	}
	if !client.HasAbility(ptypes.AbilityChatCompletions) {
		t.Fatalf("chat completions ability should be enabled")
	}
	if !client.HasAbility(ptypes.AbilityClaudeMessages) {
		t.Fatalf("claude messages ability should be enabled")
	}
}

func TestBuildProvider_CUCloudCoding(t *testing.T) {
	prov := buildProvider(cucloudcoding.PlatformId, "", "test-key", http.DefaultClient)

	client, ok := prov.(*cucloudcoding.Client)
	if !ok {
		t.Fatalf("unexpected provider type: %T", prov)
	}
	if client.Identifier() != cucloudcoding.PlatformId {
		t.Fatalf("unexpected platform id: got=%q want=%q", client.Identifier(), cucloudcoding.PlatformId)
	}
	if client.BaseURL != cucloudcoding.DefaultBaseURL {
		t.Fatalf("unexpected base url: got=%q want=%q", client.BaseURL, cucloudcoding.DefaultBaseURL)
	}
	if !client.HasAbility(ptypes.AbilityChatCompletions) {
		t.Fatalf("chat completions ability should be enabled")
	}
	if !client.HasAbility(ptypes.AbilityClaudeMessages) {
		t.Fatalf("claude messages ability should be enabled")
	}
}

func TestBuildProvider_SiliconFlow(t *testing.T) {
	prov := buildProvider(siliconflow.PlatformId, "", "test-key", http.DefaultClient)

	client, ok := prov.(*siliconflow.Client)
	if !ok {
		t.Fatalf("unexpected provider type: %T", prov)
	}
	if client.Identifier() != siliconflow.PlatformId {
		t.Fatalf("unexpected platform id: got=%q want=%q", client.Identifier(), siliconflow.PlatformId)
	}
	if client.BaseURL != siliconflow.DefaultBaseURL {
		t.Fatalf("unexpected base url: got=%q want=%q", client.BaseURL, siliconflow.DefaultBaseURL)
	}
	if !client.HasAbility(ptypes.AbilityChatCompletions) {
		t.Fatalf("chat completions ability should be enabled")
	}
	if !client.HasAbility(ptypes.AbilityClaudeMessages) {
		t.Fatalf("claude messages ability should be enabled")
	}
	if !client.HasAbility(ptypes.AbilityEmbeddings) {
		t.Fatalf("embeddings ability should be enabled")
	}
}
