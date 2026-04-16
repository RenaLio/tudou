package service

import (
	"context"

	v1 "github.com/RenaLio/tudou/api/v1"
)

type RelayService interface {
	FetchModel(ctx context.Context, req *v1.FetchModelRequest) ([]string, error)
}

type RelayServiceImpl struct {
	*Service
}

func NewRelayService(s *Service) RelayService {
	return &RelayServiceImpl{s}
}

func (s *RelayServiceImpl) FetchModel(ctx context.Context, req *v1.FetchModelRequest) ([]string, error) {
	// todo
	return []string{"gpt-4o", "gpt-3.5-turbo", "deepseek/deepseek-v3.2"}, nil
}
