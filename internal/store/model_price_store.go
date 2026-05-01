package store

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"
)

type ModelPriceStore struct {
	Models          []string
	RawData         []byte
	LatestFetchTime time.Time
	rwMu            sync.RWMutex
}

func NewModelPriceStore() *ModelPriceStore {
	s := &ModelPriceStore{}
	s.TryRefresh()
	return s
}

func (s *ModelPriceStore) Refresh() error {
	s.rwMu.Lock()
	defer s.rwMu.Unlock()
	reqUrl := "https://models.dev/api.json"
	resp, err := http.Get(reqUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	s.RawData, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	s.LatestFetchTime = time.Now()

	var models []string
	result := gjson.ParseBytes(s.RawData)
	result.ForEach(func(providerKey, providerValue gjson.Result) bool {
		providerID := providerKey.String()
		providerValue.Get("models").ForEach(func(modelKey, _ gjson.Result) bool {
			models = append(models, providerID+"#"+modelKey.String())
			return true
		})
		return true
	})
	s.Models = models
	return nil
}

func (s *ModelPriceStore) TryRefresh() {
	if len(s.RawData) == 0 {
		if err := s.Refresh(); err != nil {
			fmt.Println("refresh error:", err)
		}
		return
	}
	if time.Since(s.LatestFetchTime) > time.Hour*12 {
		if err := s.Refresh(); err != nil {
			fmt.Println("refresh error:", err)
		}
		return
	}
	return
}

func (s *ModelPriceStore) GetInputPrice(path string) float64 {
	return s.getFloat(path, "cost.input")
}

func (s *ModelPriceStore) GetOutputPrice(path string) float64 {
	return s.getFloat(path, "cost.output")
}

func (s *ModelPriceStore) GetCacheReadPrice(path string) float64 {
	return s.getFloat(path, "cost.cache_read")
}

func (s *ModelPriceStore) GetCacheCreatePrice(path string) float64 {
	return s.getFloat(path, "cost.cache_write")
}

func (s *ModelPriceStore) GetOver200KInputPrice(path string) float64 {
	return s.getFloat(path, "cost.context_over_200k.input")
}
func (s *ModelPriceStore) GetOver200KOutputPrice(path string) float64 {
	return s.getFloat(path, "cost.context_over_200k.output")
}
func (s *ModelPriceStore) GetOver200KCacheReadPrice(path string) float64 {
	return s.getFloat(path, "cost.context_over_200k.cache_read")
}

func (s *ModelPriceStore) GetOver200KCacheWritePrice(path string) float64 {
	return s.getFloat(path, "cost.context_over_200k.cache_write")
}

func (s *ModelPriceStore) HasPath(path string) bool {
	s.TryRefresh()
	s.rwMu.RLock()
	defer s.rwMu.RUnlock()
	return gjson.GetBytes(s.RawData, buildPath(path, "")).Exists()
}

func (s *ModelPriceStore) getFloat(path string, suffix string) float64 {
	s.TryRefresh()
	s.rwMu.RLock()
	defer s.rwMu.RUnlock()
	return gjson.GetBytes(s.RawData, buildPath(path, suffix)).Float()
}

func buildPath(path string, suffix string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	parts := strings.SplitN(path, ".models.", 2)
	if len(parts) == 2 {
		modelID := strings.ReplaceAll(parts[1], ".", `\.`)
		path = parts[0] + ".models." + modelID
	}

	if suffix == "" {
		return path
	}
	return path + "." + suffix
}

func (s *ModelPriceStore) FindSimilarPath(platform, model string) string {
	s.TryRefresh()
	s.rwMu.RLock()
	defer s.rwMu.RUnlock()

	if len(s.Models) == 0 {
		return ""
	}

	input := platform + "#" + model
	bestScore := 0
	best := ""
	bestPricedScore := 0
	bestPriced := ""

	for _, m := range s.Models {
		score := scoreMatch(input, platform, model, m)
		if score > bestScore {
			bestScore = score
			best = m
		}
		// 在分数比较的基础上，单独记录“有价格”的最优候选。
		if score > bestPricedScore && candidateHasPrice(s.RawData, m) {
			bestPricedScore = score
			bestPriced = m
		}
	}

	// 优先返回“有价格”且分数达标的路径；否则回退到原始最佳匹配。
	if bestPriced != "" && bestPricedScore >= 30 {
		best = bestPriced
		bestScore = bestPricedScore
	}

	if bestScore < 30 {
		return ""
	}
	// provider#modelId -> provider.models.modelId
	best = strings.Replace(best, "#", ".models.", 1)
	return best
}

func scoreMatch(input, platform, model, candidate string) int {
	// exact match
	if input == candidate {
		return 100
	}

	cPlatform, cModel, ok := splitCandidate(candidate)
	if !ok {
		return 0
	}

	score := 0

	// model matching prep (used by both platform fallback and model scoring)
	lModel := strings.ToLower(model)
	lcModel := strings.ToLower(cModel)

	// platform matching (case-insensitive)
	lPlatform := strings.ToLower(platform)
	lcPlatform := strings.ToLower(cPlatform)
	if lPlatform == lcPlatform {
		score += 40
	} else if strings.Contains(lPlatform, lcPlatform) || strings.Contains(lcPlatform, lPlatform) {
		score += 20
	} else if normalize(platform) == normalize(cPlatform) {
		score += 25
	} else if segmentOverlap(lPlatform, lcPlatform) {
		score += 10
	} else if lModel == lcModel || lModel == stripPrefix(lcModel) || stripPrefix(lModel) == stripPrefix(lcModel) {
		// platform 不匹配，但模型完全一致时仍给低分候选
		score += 5
	} else {
		return 0
	}

	// exact
	if lModel == lcModel {
		return score + 60
	}

	// strip provider prefix: "openai/gpt-5.2-codex" -> "gpt-5.2-codex"
	strippedModel := stripPrefix(lcModel)
	if lModel == strippedModel {
		return score + 55
	}

	// contains
	if strings.Contains(lModel, lcModel) || strings.Contains(lcModel, lModel) {
		score += 40
		return score
	}

	// strip prefix and contains
	strippedInput := stripPrefix(lModel)
	if strippedInput == strippedModel {
		return score + 50
	}
	if strings.Contains(strippedInput, strippedModel) || strings.Contains(strippedModel, strippedInput) {
		score += 35
		return score
	}

	// normalized comparison
	normInput := normalize(model)
	normCand := normalize(cModel)
	if normInput == normCand {
		return score + 45
	}
	if strings.Contains(normInput, normCand) || strings.Contains(normCand, normInput) {
		score += 30
		return score
	}

	return 0
}

func splitCandidate(s string) (platform, model string, ok bool) {
	idx := strings.Index(s, "#")
	if idx < 0 {
		return "", "", false
	}
	return s[:idx], s[idx+1:], true
}

func stripPrefix(model string) string {
	if idx := strings.LastIndex(model, "/"); idx >= 0 {
		return model[idx+1:]
	}
	return model
}

func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.NewReplacer("-", "", "_", "", " ", "").Replace(s)
	return s
}

func segmentOverlap(a, b string) bool {
	segA := splitSegments(a)
	segB := splitSegments(b)
	for _, sa := range segA {
		for _, sb := range segB {
			if sa == sb {
				return true
			}
		}
	}
	return false
}

func splitSegments(s string) []string {
	s = strings.NewReplacer("-", " ", "_", " ", "/", " ").Replace(s)
	return strings.Fields(s)
}

func candidateHasPrice(raw []byte, candidate string) bool {
	// candidate: provider#modelId，转换为 gjson 路径 provider.models.modelId
	path := strings.Replace(candidate, "#", ".models.", 1)
	// 简化策略：仅以 input 价格判断是否“有价格”。
	return gjson.GetBytes(raw, buildPath(path, "cost.input")).Float() > 0
}
