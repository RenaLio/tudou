package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/repository"
	"go.uber.org/zap"
)

const (
	defaultRequestLogAsyncQueueSize = 1024
	defaultRequestLogAsyncWorkers   = 1
)

type RequestLogService interface {
	Create(ctx context.Context, log *models.RequestLog) error
	CreateAsync(ctx context.Context, log *models.RequestLog) error
	BatchCreate(ctx context.Context, logs []*models.RequestLog) error
	GetByID(ctx context.Context, id int64) (*models.RequestLog, error)
	List(ctx context.Context, opt repository.RequestLogListOption) ([]*models.RequestLog, int64, error)
	Delete(ctx context.Context, id int64) error
	Exists(ctx context.Context, id int64) (bool, error)
	Flush(ctx context.Context) error
}

type asyncRequestLogTask struct {
	ctx context.Context
	log *models.RequestLog
}

type requestLogService struct {
	*Service
	repo repository.RequestLogRepo

	asyncQueue chan asyncRequestLogTask
	asyncWG    sync.WaitGroup
}

func NewRequestLogService(base *Service, repo repository.RequestLogRepo) RequestLogService {
	s := &requestLogService{
		Service:    base,
		repo:       repo,
		asyncQueue: make(chan asyncRequestLogTask, defaultRequestLogAsyncQueueSize),
	}
	s.startAsyncWorkers(defaultRequestLogAsyncWorkers)
	return s
}

func (s *requestLogService) Create(ctx context.Context, log *models.RequestLog) error {
	prepared, err := s.prepareRequestLog(log)
	if err != nil {
		return err
	}
	return s.repo.Create(ctx, prepared)
}

func (s *requestLogService) CreateAsync(ctx context.Context, log *models.RequestLog) error {
	prepared, err := s.prepareRequestLog(log)
	if err != nil {
		return err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	taskCtx := context.WithoutCancel(ctx)

	s.asyncWG.Add(1)
	select {
	case s.asyncQueue <- asyncRequestLogTask{
		ctx: taskCtx,
		log: prepared,
	}:
		return nil
	default:
		s.asyncWG.Done()
		// 队列满时回退为同步写，避免日志丢失。
		s.Log(ctx).Warn("request log async queue is full; fallback to sync create",
			zap.String("requestID", prepared.RequestID))
		return s.repo.Create(taskCtx, prepared)
	}
}

func (s *requestLogService) BatchCreate(ctx context.Context, logs []*models.RequestLog) error {
	if len(logs) == 0 {
		return nil
	}
	prepared := make([]*models.RequestLog, 0, len(logs))
	for _, item := range logs {
		logItem, err := s.prepareRequestLog(item)
		if err != nil {
			return err
		}
		prepared = append(prepared, logItem)
	}
	return s.repo.BatchCreate(ctx, prepared)
}

func (s *requestLogService) GetByID(ctx context.Context, id int64) (*models.RequestLog, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *requestLogService) List(ctx context.Context, opt repository.RequestLogListOption) ([]*models.RequestLog, int64, error) {
	return s.repo.List(ctx, opt)
}

func (s *requestLogService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *requestLogService) Exists(ctx context.Context, id int64) (bool, error) {
	return s.repo.Exists(ctx, id)
}

func (s *requestLogService) Flush(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	done := make(chan struct{})
	go func() {
		s.asyncWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *requestLogService) startAsyncWorkers(workerCount int) {
	if workerCount <= 0 {
		workerCount = 1
	}
	for i := 0; i < workerCount; i++ {
		go s.asyncCreateWorker()
	}
}

func (s *requestLogService) asyncCreateWorker() {
	for task := range s.asyncQueue {
		if err := s.repo.Create(task.ctx, task.log); err != nil {
			s.Log(task.ctx).Error("async create request log failed",
				zap.String("requestID", task.log.RequestID),
				zap.Error(err))
		}
		s.asyncWG.Done()
	}
}

func (s *requestLogService) prepareRequestLog(log *models.RequestLog) (*models.RequestLog, error) {
	if log == nil {
		return nil, errors.New("request log is nil")
	}
	logCopy := cloneRequestLog(log)
	if logCopy.ID <= 0 {
		logCopy.ID = s.NextID()
	}
	if logCopy.ID <= 0 {
		return nil, errors.New("failed to generate id by sid")
	}
	if logCopy.CreatedAt.IsZero() {
		logCopy.CreatedAt = time.Now()
	}
	return logCopy, nil
}

func cloneRequestLog(src *models.RequestLog) *models.RequestLog {
	if src == nil {
		return nil
	}
	dst := *src
	if src.Extra.Headers != nil {
		headers := make(map[string]string, len(src.Extra.Headers))
		for k, v := range src.Extra.Headers {
			headers[k] = v
		}
		dst.Extra.Headers = headers
	}
	return &dst
}
