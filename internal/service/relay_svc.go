package service

import (
	"context"
	"errors"
	"net/http"
	"time"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/helpers"
	"github.com/RenaLio/tudou/internal/loadbalancer"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/repository"
	"github.com/RenaLio/tudou/pkg/httpclient"
	"github.com/RenaLio/tudou/pkg/provider"
	"github.com/RenaLio/tudou/pkg/provider/constant"
	"github.com/RenaLio/tudou/pkg/provider/platforms/base"
	"github.com/RenaLio/tudou/pkg/provider/plog"
	"github.com/RenaLio/tudou/pkg/provider/types"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
)

const maxRetry = 3

type RelayService interface {
	FetchModel(ctx context.Context, req *v1.FetchModelRequest) ([]string, error)
	Forward(ctx context.Context, meta RelayMeta, body []byte, header http.Header) (*types.Response, error)
}

type RelayMeta struct {
	Format    types.Format
	TokenID   int64
	TokenName string
	UserID    int64
	GroupID   int64
	GroupName string
}

type RelayServiceImpl struct {
	lb         loadbalancer.LoadBalancer
	collector  loadbalancer.MetricsCollector
	requestLog RequestLogCreator
	modelRepo  repository.AIModelRepo
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
) RelayService {
	return &RelayServiceImpl{lb: lb, collector: collector, Service: s, modelRepo: modelRepo, requestLog: requestLog}
}

func (s *RelayServiceImpl) FetchModel(ctx context.Context, req *v1.FetchModelRequest) ([]string, error) {
	// todo
	return []string{"gpt-4o", "gpt-3.5-turbo", "deepseek/deepseek-v3.2"}, nil
}

func (s *RelayServiceImpl) Forward(ctx context.Context, meta RelayMeta, body []byte, header http.Header) (*types.Response, error) {
	if len(body) == 0 {
		return nil, errors.New("empty request body")
	}

	model := helpers.GetModelName(body)
	if model == "" {
		return nil, errors.New("missing model in request")
	}

	lbReq := &loadbalancer.Request{
		GroupID: meta.GroupID,
		UserID:  meta.UserID,
		Model:   model,
	}

	candidates, err := s.lb.Select(ctx, lbReq)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, loadbalancer.ErrNoAvailableChannel
	}

	var lastResp *types.Response
	var lastErr error
	var retryTrace []models.RetryDetail

	httpClient, err := httpclient.GetDefineClient(httpclient.Config{
		Timeout: -1,
	})
	if err != nil {
		return nil, err
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
		req := &types.Request{
			Model:    curUpstreamModel,
			Payload:  body,
			Format:   meta.Format,
			Headers:  header,
			IsStream: helpers.GetStream(body),
		}

		resp, execErr := prov.Execute(ctx, req, func(metrics *types.ResponseMetrics) {
			plog.Debug("metrics:", metrics)
			if s.collector != nil {
				s.collector.DecConn(candidate.Channel.ID)
			}

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
				plog.Info("request success:", metrics)
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
					ErrorCode:                 "",
					ErrorMsg:                  "",
					IsStream:                  metrics.IsStream,
					Extra: models.RequestExtra{
						RetryTrace: retryTrace,
					},
					ProviderDetail: models.ProviderDetail{
						Provider:      prov.Identifier(),
						RequestFormat: string(metrics.Format),
					},
				}
				reqFormat := metrics.Extra[constant.RequestFormatKey]
				if reqFormatStr, ok := reqFormat.(string); ok {
					reqLog.ProviderDetail.TransFormat = reqFormatStr
				}
				reqLogData, err := json.Marshal(reqLog)
				if err != nil {
					plog.Error("marshal request log error:", err)
				}
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
				}
				if err := s.requestLog.CreateAsync(ctx, &reqLog); err != nil {
					plog.Error("create request log error:", err)
				}

				plog.Debug("request log:", string(reqLogData))
			}
		})

		if execErr != nil {
			lastErr = execErr
			plog.Error("execute error:", execErr)
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}

		tryDetail := models.RetryDetail{
			ChannelID:     candidate.Channel.ID,
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

func (s *RelayServiceImpl) createLog(ctx context.Context, meta RelayMeta, model string, candidate *loadbalancer.Result, resp *types.Response, startTime time.Time, body []byte) error {
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
	return base.NewClient(httpc, baseURL, apiKey, platform, []types.Ability{types.AbilityChat, types.AbilityChatCompletions})
}
