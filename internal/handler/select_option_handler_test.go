package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	ctyuncoding "github.com/RenaLio/tudou/pkg/provider/platforms/ctyuncoding"
	cucloudcoding "github.com/RenaLio/tudou/pkg/provider/platforms/cucloud_coding"
	siliconflow "github.com/RenaLio/tudou/pkg/provider/platforms/siliconflow"
	tencentcodingplan "github.com/RenaLio/tudou/pkg/provider/platforms/tencent_coding_plan"
	"github.com/gin-gonic/gin"
)

func TestPlatformOptions_IncludesTencentCodingPlan(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	h := &Handler{}
	h.PlatformOptions(c)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got=%d want=%d", w.Code, http.StatusOK)
	}

	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"message"`
		Data struct {
			Options []struct {
				Key   string         `json:"key"`
				Value string         `json:"value"`
				Extra map[string]any `json:"extra"`
			} `json:"options"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("unexpected response code: got=%d want=0", resp.Code)
	}

	targetIdx := -1
	for i := range resp.Data.Options {
		if resp.Data.Options[i].Value == tencentcodingplan.PlatformId {
			targetIdx = i
			break
		}
	}
	if targetIdx < 0 {
		t.Fatalf("platform option %q not found", tencentcodingplan.PlatformId)
	}
	target := resp.Data.Options[targetIdx]
	if target.Key != "Tencent Coding Plan" {
		t.Fatalf("unexpected option key: got=%q want=%q", target.Key, "Tencent Coding Plan")
	}

	baseURL, ok := target.Extra["exampleBaseUrl"].(string)
	if !ok {
		t.Fatalf("exampleBaseUrl type mismatch: %T", target.Extra["exampleBaseUrl"])
	}
	if baseURL != tencentcodingplan.DefaultBaseURL {
		t.Fatalf("unexpected base url: got=%q want=%q", baseURL, tencentcodingplan.DefaultBaseURL)
	}

	rawPaths, ok := target.Extra["paths"].(map[string]any)
	if !ok {
		t.Fatalf("paths type mismatch: %T", target.Extra["paths"])
	}

	expected := map[string]string{
		"chat.completion": "/coding/v3/chat/completions",
		"claude.messages": "/coding/anthropic/v1/messages",
	}
	for k, v := range expected {
		got, ok := rawPaths[k].(string)
		if !ok {
			t.Fatalf("path value type mismatch for %q: %T", k, rawPaths[k])
		}
		if got != v {
			t.Fatalf("unexpected path for %q: got=%q want=%q", k, got, v)
		}
	}
}

func TestPlatformOptions_IncludesCTYunCoding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	h := &Handler{}
	h.PlatformOptions(c)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got=%d want=%d", w.Code, http.StatusOK)
	}

	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"message"`
		Data struct {
			Options []struct {
				Key   string         `json:"key"`
				Value string         `json:"value"`
				Extra map[string]any `json:"extra"`
			} `json:"options"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("unexpected response code: got=%d want=0", resp.Code)
	}

	targetIdx := -1
	for i := range resp.Data.Options {
		if resp.Data.Options[i].Value == ctyuncoding.PlatformId {
			targetIdx = i
			break
		}
	}
	if targetIdx < 0 {
		t.Fatalf("platform option %q not found", ctyuncoding.PlatformId)
	}
	target := resp.Data.Options[targetIdx]
	if target.Key != "天翼云Coding" {
		t.Fatalf("unexpected option key: got=%q want=%q", target.Key, "天翼云Coding")
	}

	baseURL, ok := target.Extra["exampleBaseUrl"].(string)
	if !ok {
		t.Fatalf("exampleBaseUrl type mismatch: %T", target.Extra["exampleBaseUrl"])
	}
	if baseURL != ctyuncoding.DefaultBaseURL {
		t.Fatalf("unexpected base url: got=%q want=%q", baseURL, ctyuncoding.DefaultBaseURL)
	}

	rawPaths, ok := target.Extra["paths"].(map[string]any)
	if !ok {
		t.Fatalf("paths type mismatch: %T", target.Extra["paths"])
	}

	expected := map[string]string{
		"chat.completion": "/coding/v1/chat/completions",
		"claude.messages": "/coding/v1/messages",
	}
	for k, v := range expected {
		got, ok := rawPaths[k].(string)
		if !ok {
			t.Fatalf("path value type mismatch for %q: %T", k, rawPaths[k])
		}
		if got != v {
			t.Fatalf("unexpected path for %q: got=%q want=%q", k, got, v)
		}
	}
}

func TestPlatformOptions_IncludesCUCloudCoding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	h := &Handler{}
	h.PlatformOptions(c)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got=%d want=%d", w.Code, http.StatusOK)
	}

	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"message"`
		Data struct {
			Options []struct {
				Key   string         `json:"key"`
				Value string         `json:"value"`
				Extra map[string]any `json:"extra"`
			} `json:"options"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("unexpected response code: got=%d want=0", resp.Code)
	}

	targetIdx := -1
	for i := range resp.Data.Options {
		if resp.Data.Options[i].Value == cucloudcoding.PlatformId {
			targetIdx = i
			break
		}
	}
	if targetIdx < 0 {
		t.Fatalf("platform option %q not found", cucloudcoding.PlatformId)
	}
	target := resp.Data.Options[targetIdx]
	if target.Key != "联通云Coding" {
		t.Fatalf("unexpected option key: got=%q want=%q", target.Key, "联通云Coding")
	}

	baseURL, ok := target.Extra["exampleBaseUrl"].(string)
	if !ok {
		t.Fatalf("exampleBaseUrl type mismatch: %T", target.Extra["exampleBaseUrl"])
	}
	if baseURL != cucloudcoding.DefaultBaseURL {
		t.Fatalf("unexpected base url: got=%q want=%q", baseURL, cucloudcoding.DefaultBaseURL)
	}

	rawPaths, ok := target.Extra["paths"].(map[string]any)
	if !ok {
		t.Fatalf("paths type mismatch: %T", target.Extra["paths"])
	}

	expected := map[string]string{
		"chat.completion": "/v1/chat/completions",
		"claude.messages": "/v1/messages",
	}
	for k, v := range expected {
		got, ok := rawPaths[k].(string)
		if !ok {
			t.Fatalf("path value type mismatch for %q: %T", k, rawPaths[k])
		}
		if got != v {
			t.Fatalf("unexpected path for %q: got=%q want=%q", k, got, v)
		}
	}
}

func TestPlatformOptions_IncludesSiliconFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	h := &Handler{}
	h.PlatformOptions(c)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got=%d want=%d", w.Code, http.StatusOK)
	}

	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"message"`
		Data struct {
			Options []struct {
				Key   string         `json:"key"`
				Value string         `json:"value"`
				Extra map[string]any `json:"extra"`
			} `json:"options"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("unexpected response code: got=%d want=0", resp.Code)
	}

	targetIdx := -1
	for i := range resp.Data.Options {
		if resp.Data.Options[i].Value == siliconflow.PlatformId {
			targetIdx = i
			break
		}
	}
	if targetIdx < 0 {
		t.Fatalf("platform option %q not found", siliconflow.PlatformId)
	}
	target := resp.Data.Options[targetIdx]
	if target.Key != "SiliconFlow 硅基流动" {
		t.Fatalf("unexpected option key: got=%q want=%q", target.Key, "SiliconFlow 硅基流动")
	}

	baseURL, ok := target.Extra["exampleBaseUrl"].(string)
	if !ok {
		t.Fatalf("exampleBaseUrl type mismatch: %T", target.Extra["exampleBaseUrl"])
	}
	if baseURL != siliconflow.DefaultBaseURL {
		t.Fatalf("unexpected base url: got=%q want=%q", baseURL, siliconflow.DefaultBaseURL)
	}

	rawPaths, ok := target.Extra["paths"].(map[string]any)
	if !ok {
		t.Fatalf("paths type mismatch: %T", target.Extra["paths"])
	}

	expected := map[string]string{
		"chat.completion":   "/v1/chat/completions",
		"claude.messages":   "/v1/messages",
		"openai.embeddings": "/v1/embeddings",
	}
	for k, v := range expected {
		got, ok := rawPaths[k].(string)
		if !ok {
			t.Fatalf("path value type mismatch for %q: %T", k, rawPaths[k])
		}
		if got != v {
			t.Fatalf("unexpected path for %q: got=%q want=%q", k, got, v)
		}
	}
}
