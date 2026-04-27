package repository

import (
	"context"
	"strings"
	"time"

	"github.com/RenaLio/tudou/internal/models"
)

type RequestLogListOption struct {
	Page          int
	PageSize      int
	OrderBy       string
	Keyword       string
	IDGT          int64
	IDGTE         int64
	IDLT          int64
	IDLTE         int64
	RequestID     string
	UserID        int64
	TokenID       int64
	GroupID       int64
	ChannelID     int64
	Model         string
	UpstreamModel string
	Status        *models.RequestStatus
	IsStream      *bool
	DateFrom      *time.Time
	DateTo        *time.Time
	IDs           []int64
}

type RequestLogRepo interface {
	Create(ctx context.Context, log *models.RequestLog) error
	BatchCreate(ctx context.Context, logs []*models.RequestLog) error
	GetByID(ctx context.Context, id int64) (*models.RequestLog, error)
	List(ctx context.Context, opt RequestLogListOption) ([]*models.RequestLog, int64, error)
	Delete(ctx context.Context, id int64) error
	Exists(ctx context.Context, id int64) (bool, error)
}

type requestLogRepo struct {
	*Repository
}

func NewRequestLogRepo(r *Repository) RequestLogRepo {
	return &requestLogRepo{Repository: r}
}

func (r *requestLogRepo) Create(ctx context.Context, log *models.RequestLog) error {
	return Create[models.RequestLog](ctx, log, r.DB(ctx))
}

func (r *requestLogRepo) BatchCreate(ctx context.Context, logs []*models.RequestLog) error {
	return BatchCreate[*models.RequestLog](ctx, logs, r.DB(ctx))
}

func (r *requestLogRepo) GetByID(ctx context.Context, id int64) (*models.RequestLog, error) {
	return GetByID[models.RequestLog](ctx, id, r.DB(ctx))
}

func (r *requestLogRepo) List(ctx context.Context, opt RequestLogListOption) ([]*models.RequestLog, int64, error) {
	db := r.DB(ctx).Model(&models.RequestLog{})

	keyword := strings.TrimSpace(opt.Keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where(
			"request_id LIKE ? OR channel_name LIKE ? OR model LIKE ? OR upstream_model LIKE ? OR error_code LIKE ? OR error_msg LIKE ?",
			like, like, like, like, like, like,
		)
	}

	requestID := strings.TrimSpace(opt.RequestID)
	if requestID != "" {
		db = db.Where("request_id = ?", requestID)
	}
	if opt.IDGT > 0 {
		db = db.Where("id > ?", opt.IDGT)
	}
	if opt.IDGTE > 0 {
		db = db.Where("id >= ?", opt.IDGTE)
	}
	if opt.IDLT > 0 {
		db = db.Where("id < ?", opt.IDLT)
	}
	if opt.IDLTE > 0 {
		db = db.Where("id <= ?", opt.IDLTE)
	}
	if opt.UserID > 0 {
		db = db.Where("user_id = ?", opt.UserID)
	}
	if opt.TokenID > 0 {
		db = db.Where("token_id = ?", opt.TokenID)
	}
	if opt.GroupID > 0 {
		db = db.Where("group_id = ?", opt.GroupID)
	}
	if opt.ChannelID > 0 {
		db = db.Where("channel_id = ?", opt.ChannelID)
	}

	model := strings.TrimSpace(opt.Model)
	if model != "" {
		db = db.Where("model = ?", model)
	}
	upstreamModel := strings.TrimSpace(opt.UpstreamModel)
	if upstreamModel != "" {
		db = db.Where("upstream_model = ?", upstreamModel)
	}

	if opt.Status != nil {
		db = db.Where("status = ?", *opt.Status)
	}
	if opt.IsStream != nil {
		db = db.Where("is_stream = ?", *opt.IsStream)
	}
	if opt.DateFrom != nil {
		db = db.Where("created_at >= ?", *opt.DateFrom)
	}
	if opt.DateTo != nil {
		db = db.Where("created_at <= ?", *opt.DateTo)
	}
	if len(opt.IDs) > 0 {
		db = db.Where("id IN ?", uniqueInt64(opt.IDs))
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	_, pageSize, offset := normalizePagination(opt.Page, opt.PageSize)
	orderBy := sanitizeOrderBy(opt.OrderBy, "created_at DESC, id DESC")

	data := make([]*models.RequestLog, 0, pageSize)
	if err := db.Order(orderBy).Offset(offset).Limit(pageSize).Find(&data).Error; err != nil {
		return nil, 0, err
	}
	return data, total, nil
}

func (r *requestLogRepo) Delete(ctx context.Context, id int64) error {
	return Delete[models.RequestLog](ctx, id, r.DB(ctx))
}

func (r *requestLogRepo) Exists(ctx context.Context, id int64) (bool, error) {
	return Exists[models.RequestLog](ctx, id, r.DB(ctx))
}
