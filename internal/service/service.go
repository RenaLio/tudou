package service

import (
	"context"
	"errors"

	"github.com/RenaLio/tudou/internal/pkg/jwt"
	"github.com/RenaLio/tudou/internal/pkg/log"
	"github.com/RenaLio/tudou/internal/pkg/sid"
	"github.com/RenaLio/tudou/internal/repository"
	"github.com/RenaLio/tudou/pkg/cache"
)

type Service struct {
	logger *log.Logger
	sid    *sid.Sid
	cache  *cache.JsonCache
	jwt    *jwt.JWT
	tm     repository.Transaction
}

func NewService(logger *log.Logger, sid *sid.Sid, cache *cache.JsonCache, jwt *jwt.JWT, tm repository.Transaction) *Service {
	return &Service{
		logger: logger,
		sid:    sid,
		cache:  cache,
		jwt:    jwt,
		tm:     tm,
	}
}

func (s *Service) Log(ctx context.Context) *log.Logger {
	return s.logger.FromContext(ctx)
}

func (s *Service) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if fn == nil {
		return errors.New("transaction function is nil")
	}
	if s.tm == nil {
		return errors.New("transaction manager is nil")
	}
	return s.tm.Transaction(ctx, fn)
}

func (s *Service) NextID() int64 {
	if s.sid == nil {
		return 0
	}
	return s.sid.GenInt64()
}

func (s *Service) Cache() *cache.JsonCache {
	return s.cache
}

func (s *Service) JWT() *jwt.JWT {
	return s.jwt
}
