package service

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/helpers"
	"github.com/RenaLio/tudou/internal/loadbalancer"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/repository"
	"github.com/RenaLio/tudou/internal/types"
	"github.com/RenaLio/tudou/pkg/httpclient"
	"github.com/RenaLio/tudou/pkg/provider"
	"github.com/RenaLio/tudou/pkg/provider/constant"
	alibabacodingplancn "github.com/RenaLio/tudou/pkg/provider/platforms/alibaba_coding_plan_cn"
	baiducoding "github.com/RenaLio/tudou/pkg/provider/platforms/baidu_coding"
	"github.com/RenaLio/tudou/pkg/provider/platforms/base"
	codingplan "github.com/RenaLio/tudou/pkg/provider/platforms/coding_plan"
	ctyuncoding "github.com/RenaLio/tudou/pkg/provider/platforms/ctyuncoding"
	cucloudcoding "github.com/RenaLio/tudou/pkg/provider/platforms/cucloud_coding"
	ecloudcoding "github.com/RenaLio/tudou/pkg/provider/platforms/ecloud_coding"
	jdcoding "github.com/RenaLio/tudou/pkg/provider/platforms/jd_coding"
	kimiforcoding "github.com/RenaLio/tudou/pkg/provider/platforms/kimi_for_coding"
	mimocoding "github.com/RenaLio/tudou/pkg/provider/platforms/mimo_coding"
	minimaxcoding "github.com/RenaLio/tudou/pkg/provider/platforms/minimax_coding"
	"github.com/RenaLio/tudou/pkg/provider/platforms/openai"
	relaystation "github.com/RenaLio/tudou/pkg/provider/platforms/relay_station"
	tencentcodingplan "github.com/RenaLio/tudou/pkg/provider/platforms/tencent_coding_plan"
	"github.com/RenaLio/tudou/pkg/provider/plog"
	ptypes "github.com/RenaLio/tudou/pkg/provider/types"
)

const maxRetry = 3

type RelayService struct {
	lb          loadbalancer.LoadBalancer
	collector   loadbalancer.MetricsCollector
	requestLog  RequestLogCreator
	modelRepo   repository.AIModelRepo
	groupRepo   repository.ChannelGroupRepo
	channelRepo repository.ChannelRepo
	*Service
}

type RequestLogCreator interface {
	CreateAsync(ctx context.Context, log *models.RequestLog) error
}

func NewRelayService(
	s *Service,
	lb loadbalancer.LoadBalancer,
	collector loadbalancer.MetricsCollector,
	modelRepo repository.AIModelRepo,
	requestLog RequestLogCreator,
	groupRepo repository.ChannelGroupRepo,
	channelRepo repository.ChannelRepo,
) *RelayService {
	return &RelayService{lb: lb, collector: collector, Service: s, modelRepo: modelRepo, requestLog: requestLog, groupRepo: groupRepo, channelRepo: channelRepo}
}

func (s *RelayService) GetTokenModels(ctx context.Context, tokenId int64, groupId int64) (*v1.RelayListResp[v1.RelayModelItemResp], error) {
	modelSet := make(map[string]struct{})
	group, err := s.groupRepo.GetByIDWithChannels(ctx, groupId)
	if err != nil {
		return nil, err
	}
	channels := group.Channels
	resp := &v1.RelayListResp[v1.RelayModelItemResp]{
		Object: "list",
		Data:   make([]v1.RelayModelItemResp, 0),
	}
	for _, channel := range channels {
		modelMap := channel.Models()
		for key := range modelMap {
			if _, ok := modelSet[key]; ok {
				continue
			}
			resp.Data = append(resp.Data, v1.RelayModelItemResp{
				Id:      key,
				Object:  "model",
				Created: channel.CreatedAt.Unix(),
				OwnedBy: string(channel.Type),
			})
			modelSet[key] = struct{}{}
		}
	}
	return resp, nil
}

func (s *RelayService) FetchModel(ctx context.Context, req *v1.FetchModelRequest) ([]string, error) {
	httpClient := httpclient.GetDefaultClient()
	prov := buildProvider(string(req.Type), req.BaseURL, req.APIKey, httpClient)
	modelList, err := prov.Models()
	if err != nil {
		return nil, err
	}
	return modelList, nil
}

func (s *RelayService) Forward(ctx context.Context, meta types.RelayMeta, body []byte, rawHeader http.Header) (*ptypes.Response, error) {
	if len(body) == 0 {
		return nil, errors.New("empty request body")
	}

	model := helpers.GetModelName(body)
	if model == "" {
		return nil, errors.New("missing model in request")
	}

	lbReq := &loadbalancer.Request{
		GroupID:  meta.GroupID,
		UserID:   meta.UserID,
		Model:    model,
		Strategy: meta.Strategy,
	}

	candidates, err := s.lb.Select(ctx, lbReq)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, loadbalancer.ErrNoAvailableChannel
	}

	var lastResp *ptypes.Response
	var lastErr error
	var retryTrace []models.RetryDetail

	baseHeader := http.Header{}

	exceptedHeaderKeys := []string{"Content-Type", "User-Agent", "X-Request-Id"}

	for _, key := range exceptedHeaderKeys {
		if val, ok := rawHeader[key]; ok {
			baseHeader[key] = val
		}
	}

	for i, candidate := range candidates {
		if i >= maxRetry {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:

		}
		if s.collector != nil {
			s.collector.IncConn(candidate.Channel.ID)
		}

		httpClient, err := httpclient.GetDefineClient(httpclient.Config{
			Timeout:      -1,
			DisableHTTP2: candidate.Channel.Settings.DisableHTTP2,
		})
		if err != nil {
			lastErr = err
			plog.Error("build http client error:", err)
			if s.collector != nil {
				s.collector.DecConn(candidate.Channel.ID)
			}
			continue
		}

		prov := buildProvider(string(candidate.Channel.Type), candidate.Channel.BaseURL, candidate.Channel.APIKey, httpClient)
		curUpstreamModel := candidate.UpstreamModel
		hasLogged := false
		body := helpers.SetModelName(body, curUpstreamModel)
		reqHeader := baseHeader.Clone()
		for key, value := range candidate.Channel.Settings.CustomHeaders {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			reqHeader.Set(key, value)
		}
		req := &ptypes.Request{
			Model:    curUpstreamModel,
			Payload:  body,
			Format:   meta.Format,
			Headers:  reqHeader,
			IsStream: helpers.GetStream(body),
		}
		var resp *ptypes.Response
		var execErr error

		resp, execErr = prov.Execute(ctx, req, func(metrics *ptypes.ResponseMetrics) {
			plog.Debug("metrics:", metrics)

			lbRecord := &loadbalancer.ResultRecord{
				Model:         model,
				UpstreamModel: curUpstreamModel,
				ChannelID:     candidate.Channel.ID,
				ChannelName:   candidate.Channel.Name,
				OutputTokens:  metrics.Usage.OutputTokens,
				TTFT:          metrics.TTFT.Milliseconds(),
				Duration:      metrics.TransferTime.Milliseconds(),
				Status:        metrics.Status,
				StatusCode:    metrics.StatusCode,
			}

			if s.collector != nil {
				collectErr := s.collector.CollectMetrics(ctx, lbRecord)
				if collectErr != nil {
					plog.Error("collect metrics error:", collectErr)
				}
			}

			// 成功 || 达到最大重试次数 || 没有后继候选
			// - 记录日志
			if metrics.Status == 1 || i >= maxRetry-1 || i == len(candidates)-1 {
				hasLogged = true
				status := models.RequestStatusSuccess
				if metrics.Status != 1 {
					status = models.RequestStatusFail
				}
				reqLog := models.RequestLog{
					ID:                        s.NextID(),
					RequestID:                 getRequestId(ctx),
					UserID:                    meta.UserID,
					TokenID:                   meta.TokenID,
					TokenName:                 meta.TokenName,
					GroupID:                   meta.GroupID,
					GroupName:                 meta.GroupName,
					ChannelID:                 candidate.Channel.ID,
					ChannelName:               candidate.Channel.Name,
					ChannelPriceRate:          candidate.Channel.PriceRate,
					Model:                     model,
					UpstreamModel:             curUpstreamModel,
					InputToken:                metrics.Usage.InputTokens,
					OutputToken:               metrics.Usage.OutputTokens,
					CachedCreationInputTokens: metrics.Usage.CachedCreationInputTokens,
					CachedReadInputTokens:     metrics.Usage.CachedReadInputTokens,
					Pricing:                   models.ModelPricing{},
					CostMicros:                0,
					Status:                    status,
					TTFT:                      metrics.TTFT.Milliseconds(),
					TransferTime:              metrics.TransferTime.Milliseconds(),
					ErrorCode:                 strconv.FormatInt(int64(metrics.StatusCode), 10),
					ErrorMsg:                  "",
					IsStream:                  metrics.IsStream,
					Extra: models.RequestExtra{
						Headers:     nil,
						IP:          meta.Extra.IP,
						UserAgent:   rawHeader.Get("User-Agent"),
						RequestPath: meta.Extra.Path,
						RetryTrace:  retryTrace,
					},
					ProviderDetail: models.ProviderDetail{
						Provider:      prov.Identifier(),
						RequestFormat: string(metrics.Format),
					},
				}
				if metrics.ProcessingError != nil {
					reqLog.ErrorMsg = metrics.ProcessingError.Error()
				}
				reqFormat := metrics.Extra[constant.RequestFormatKey]
				if reqFormatStr, ok := reqFormat.(ptypes.Format); ok {
					reqLog.ProviderDetail.TransFormat = string(reqFormatStr)
				}

				if resp != nil && (resp.StatusCode < 200 || resp.StatusCode >= 300) {
					temp := string(resp.RawData)
					if temp != "" {
						reqLog.ErrorMsg = temp
					}
				}

				headerMap := make(map[string]string)
				for key, values := range rawHeader {
					headerMap[key] = values[0]
				}
				reqLog.Extra.Headers = headerMap
				delete(reqLog.Extra.Headers, "Authorization")
				delete(reqLog.Extra.Headers, "authorization")

				cbCtx := context.Background()
				aiModel, err := s.modelRepo.GetByName(cbCtx, curUpstreamModel)
				if err != nil {
					plog.Error("get ai model error:", err)
				}
				if aiModel != nil {
					if aiModel.PricingType == models.ModelPricingTypeTokens {
						reqLog.Pricing = aiModel.Pricing
						calcInputTokens := metrics.Usage.InputTokens - metrics.Usage.CachedCreationInputTokens - metrics.Usage.CachedReadInputTokens
						reqLog.CostMicros = aiModel.CalculateByTokensWithCacheAndContextMicros(
							calcInputTokens,
							metrics.Usage.OutputTokens,
							metrics.Usage.CachedCreationInputTokens,
							metrics.Usage.CachedReadInputTokens,
							metrics.Usage.InputTokens,
						)
					} else {
						reqLog.Pricing = aiModel.Pricing
						reqLog.CostMicros = aiModel.CalculateByRequestWithContextMicros(metrics.Usage.InputTokens)
					}
					reqLog.CostMicros = int64(float64(reqLog.CostMicros) * candidate.Channel.PriceRate)
				}
				if err := s.requestLog.CreateAsync(ctx, &reqLog); err != nil {
					plog.Error("create request log error:", err)
				}

			}
		})

		if execErr != nil {
			lastErr = execErr
			plog.Error("execute error:", execErr)
			if s.collector != nil {
				s.collector.DecConn(candidate.Channel.ID)
			}
			tryDetail := models.RetryDetail{
				ChannelID:     candidate.Channel.ID,
				ChannelName:   candidate.Channel.Name,
				UpstreamModel: candidate.UpstreamModel,
				StatusCode:    -1,
				StatusBody:    execErr.Error(),
			}
			if !hasLogged && (i >= maxRetry-1 || i == len(candidates)-1) {
				if err := s.logFinalExecuteError(ctx, meta, model, candidate, curUpstreamModel, rawHeader, retryTrace, execErr, prov, req.IsStream); err != nil {
					plog.Error("create fallback request log error:", err)
				}
			}
			retryTrace = append(retryTrace, tryDetail)
			continue
		}
		lastResp = resp

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}

		tryDetail := models.RetryDetail{
			ChannelID:     candidate.Channel.ID,
			ChannelName:   candidate.Channel.Name,
			UpstreamModel: candidate.UpstreamModel,
			StatusCode:    resp.StatusCode,
			StatusBody:    "",
		}

		if !resp.IsStream {
			tryDetail.StatusBody = string(resp.RawData)
		}
		retryTrace = append(retryTrace, tryDetail)
	}

	if lastResp != nil {
		return lastResp, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, loadbalancer.ErrNoAvailableChannel
}

func (s *RelayService) logFinalExecuteError(
	ctx context.Context,
	meta types.RelayMeta,
	model string,
	candidate *loadbalancer.Result,
	upstreamModel string,
	rawHeader http.Header,
	retryTrace []models.RetryDetail,
	execErr error,
	prov provider.Provider,
	isStream bool,
) error {
	if candidate == nil {
		return nil
	}

	headerMap := make(map[string]string)
	for key, values := range rawHeader {
		if len(values) == 0 {
			continue
		}
		headerMap[key] = values[0]
	}
	delete(headerMap, "Authorization")
	delete(headerMap, "authorization")

	reqLog := models.RequestLog{
		ID:                        s.NextID(),
		RequestID:                 getRequestId(ctx),
		UserID:                    meta.UserID,
		TokenID:                   meta.TokenID,
		TokenName:                 meta.TokenName,
		GroupID:                   meta.GroupID,
		GroupName:                 meta.GroupName,
		ChannelID:                 candidate.Channel.ID,
		ChannelName:               candidate.Channel.Name,
		ChannelPriceRate:          candidate.Channel.PriceRate,
		Model:                     model,
		UpstreamModel:             upstreamModel,
		InputToken:                0,
		OutputToken:               0,
		CachedCreationInputTokens: 0,
		CachedReadInputTokens:     0,
		Pricing:                   models.ModelPricing{},
		CostMicros:                0,
		Status:                    models.RequestStatusFail,
		TTFT:                      0,
		TransferTime:              0,
		ErrorCode:                 "-1",
		ErrorMsg:                  execErr.Error(),
		IsStream:                  isStream,
		Extra: models.RequestExtra{
			Headers:     headerMap,
			IP:          meta.Extra.IP,
			UserAgent:   rawHeader.Get("User-Agent"),
			RequestPath: meta.Extra.Path,
			RetryTrace:  retryTrace,
		},
		ProviderDetail: models.ProviderDetail{
			Provider:      prov.Identifier(),
			RequestFormat: string(meta.Format),
		},
	}
	reqLog.ProviderDetail.TransFormat = string(meta.Format)
	return s.requestLog.CreateAsync(ctx, &reqLog)
}

func buildProvider(platform string, baseURL string, apiKey string, httpc *http.Client) provider.Provider {
	switch platform {
	case openai.PlatformId:
		return openai.NewClient(httpc, baseURL, apiKey)
	case ecloudcoding.PlatformId:
		return ecloudcoding.NewClient(httpc, baseURL, apiKey)
	case mimocoding.PlatformId:
		return mimocoding.NewClient(httpc, baseURL, apiKey)
	case minimaxcoding.PlatformId:
		return minimaxcoding.NewClient(httpc, baseURL, apiKey)
	case relaystation.PlatformId:
		return relaystation.NewClient(httpc, baseURL, apiKey)
	case baiducoding.PlatformId:
		return baiducoding.NewClient(httpc, baseURL, apiKey)
	case cucloudcoding.PlatformId:
		return cucloudcoding.NewClient(httpc, baseURL, apiKey)
	case ctyuncoding.PlatformId:
		return ctyuncoding.NewClient(httpc, baseURL, apiKey)
	case jdcoding.PlatformId:
		return jdcoding.NewClient(httpc, baseURL, apiKey)
	case alibabacodingplancn.PlatformId:
		return alibabacodingplancn.NewClient(httpc, baseURL, apiKey)
	case kimiforcoding.PlatformId:
		return kimiforcoding.NewClient(httpc, baseURL, apiKey)
	case tencentcodingplan.PlatformId:
		return tencentcodingplan.NewClient(httpc, baseURL, apiKey)
	case "coding-plan-adapter":
		return codingplan.NewClient(httpc, baseURL, apiKey)
	case "chat-completion":
		fallthrough
	default:
		return base.NewClient(httpc, baseURL, apiKey, platform, []ptypes.Ability{ptypes.AbilityChatCompletions}, nil)
	}
}
