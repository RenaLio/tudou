package repository

import (
	"context"
	"strings"

	"github.com/RenaLio/tudou/internal/models"
)

type AggregationTaskRepo interface {
	Create(ctx context.Context, task *models.AggregationTask) error
	Update(ctx context.Context, task *models.AggregationTask) error
	GetByID(ctx context.Context, id int64) (*models.AggregationTask, error)
	GetLatestByTaskName(ctx context.Context, taskName string) (*models.AggregationTask, error)
	GetLatestCompletedByTaskName(ctx context.Context, taskName string) (*models.AggregationTask, error)
}

type aggregationTaskRepo struct {
	*Repository
}

func NewAggregationTaskRepo(r *Repository) AggregationTaskRepo {
	return &aggregationTaskRepo{Repository: r}
}

func (r *aggregationTaskRepo) Create(ctx context.Context, task *models.AggregationTask) error {
	return Create[models.AggregationTask](ctx, task, r.DB(ctx))
}

func (r *aggregationTaskRepo) Update(ctx context.Context, task *models.AggregationTask) error {
	return Update[models.AggregationTask](ctx, task, task.ID, []string{"ID", "CreatedAt"}, r.DB(ctx))
}

func (r *aggregationTaskRepo) GetByID(ctx context.Context, id int64) (*models.AggregationTask, error) {
	return GetByID[models.AggregationTask](ctx, id, r.DB(ctx))
}

func (r *aggregationTaskRepo) GetLatestByTaskName(ctx context.Context, taskName string) (*models.AggregationTask, error) {
	taskName = strings.TrimSpace(taskName)
	task := new(models.AggregationTask)
	if err := r.DB(ctx).
		Where("task_name = ?", taskName).
		Order("id DESC").
		First(task).Error; err != nil {
		return nil, err
	}
	return task, nil
}

func (r *aggregationTaskRepo) GetLatestCompletedByTaskName(ctx context.Context, taskName string) (*models.AggregationTask, error) {
	taskName = strings.TrimSpace(taskName)
	task := new(models.AggregationTask)
	if err := r.DB(ctx).
		Where("task_name = ?", taskName).
		Where("status = ?", models.AggregationTaskStatusDone).
		Order("id DESC").
		First(task).Error; err != nil {
		return nil, err
	}
	return task, nil
}
