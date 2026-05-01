package service

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/helpers"
	"github.com/RenaLio/tudou/internal/loadbalancer"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/repository"
	"github.com/RenaLio/tudou/internal/types"
	"github.com/RenaLio/tudou/pkg/httpclient"
	"github.com/RenaLio/tudou/pkg/provider"
	"github.com/RenaLio/tudou/pkg/provider/constant"
	"github.com/RenaLio/tudou/pkg/provider/platforms/base"
	"github.com/RenaLio/tudou/pkg/provider/plog"
	ptypes "github.com/RenaLio/tudou/pkg/provider/types"
	"github.com/google/uuid"
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

	httpClient, err := httpclient.GetDefineClient(httpclient.Config{
		Timeout: -1,
	})
	if err != nil {
		return nil, err
	}

	header := http.Header{}

	exceptedHeaderKeys := []string{"Content-Type", "User-Agent", "X-Request-Id"}

	for _, key := range exceptedHeaderKeys {
		if val, ok := rawHeader[key]; ok {
			header[key] = val
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
		plog.Debug("i", i, "nums", len(candidates))

		prov := buildProvider(string(candidate.Channel.Type), candidate.Channel.BaseURL, candidate.Channel.APIKey, httpClient)
		curUpstreamModel := candidate.UpstreamModel
		body := helpers.SetModelName(body, curUpstreamModel)
		req := &ptypes.Request{
			Model:    curUpstreamModel,
			Payload:  body,
			Format:   meta.Format,
			Headers:  header,
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

			// 成功 || 达到最大重试次数
			// - 记录日志
			if metrics.Status == 1 || i >= maxRetry-1 {
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
				reqFormat := metrics.Extra[constant.RequestFormatKey]
				if reqFormatStr, ok := reqFormat.(ptypes.Format); ok {
					reqLog.ProviderDetail.TransFormat = string(reqFormatStr)
				}

				if resp != nil && !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
					reqLog.ErrorMsg = string(resp.RawData)
				}

				headerMap := make(map[string]string)
				for key, values := range rawHeader {
					headerMap[key] = values[0]
				}
				reqLog.Extra.Headers = headerMap

				cbCtx := context.Background()
				aiModel, err := s.modelRepo.GetByName(cbCtx, curUpstreamModel)
				if err != nil {
					plog.Error("get ai model error:", err)
				}
				if aiModel != nil {
					if aiModel.PricingType == models.ModelPricingTypeTokens {
						reqLog.Pricing = aiModel.Pricing
						reqLog.CostMicros = aiModel.CalculateByTokensWithCacheMicros(metrics.Usage.InputTokens, metrics.Usage.OutputTokens, metrics.Usage.CachedCreationInputTokens, metrics.Usage.CachedReadInputTokens)
					} else {
						reqLog.Pricing = aiModel.Pricing
						reqLog.CostMicros = aiModel.CalculateByRequestMicros()
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
			retryTrace = append(retryTrace, models.RetryDetail{
				ChannelID:     candidate.Channel.ID,
				ChannelName:   candidate.Channel.Name,
				UpstreamModel: candidate.UpstreamModel,
				StatusCode:    -1,
				StatusBody:    execErr.Error(),
			})
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

func (s *RelayService) createLog(ctx context.Context, meta types.RelayMeta, model string, candidate *loadbalancer.Result, resp *ptypes.Response, startTime time.Time, body []byte) error {
	log := &models.RequestLog{
		RequestID:     uuid.New().String(),
		UserID:        meta.UserID,
		TokenID:       meta.TokenID,
		GroupID:       meta.GroupID,
		ChannelID:     candidate.Channel.ID,
		ChannelName:   candidate.Channel.Name,
		Model:         model,
		UpstreamModel: candidate.UpstreamModel,
		Status:        models.RequestStatusSuccess,
		CreatedAt:     startTime,
		IsStream:      helpers.GetStream(body),
	}
	if resp != nil {
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			log.Status = models.RequestStatusSuccess
		} else {
			log.Status = models.RequestStatusFail
		}
		log.TransferTime = time.Since(startTime).Milliseconds()
	}
	return s.requestLog.CreateAsync(ctx, log)
}

func buildProvider(platform string, baseURL string, apiKey string, httpc *http.Client) provider.Provider {
	return base.NewClient(httpc, baseURL, apiKey, platform, []ptypes.Ability{ptypes.AbilityChat, ptypes.AbilityChatCompletions}, nil)
}
