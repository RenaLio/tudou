package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/RenaLio/tudou/internal/models"
	"gorm.io/gorm"
)

type ChannelGroupListOption struct {
	Page            int
	PageSize        int
	OrderBy         string
	Keyword         string
	ChannelID       int64
	PermissionNumGE *int32
	IDs             []int64
	PreloadChannels bool
}

type ChannelGroupRepo interface {
	Create(ctx context.Context, group *models.ChannelGroup) error
	BatchCreate(ctx context.Context, groups []*models.ChannelGroup) error
	GetByID(ctx context.Context, id int64) (*models.ChannelGroup, error)
	GetByIDWithChannels(ctx context.Context, id int64) (*models.ChannelGroup, error)
	GetByName(ctx context.Context, name string) (*models.ChannelGroup, error)
	List(ctx context.Context, opt ChannelGroupListOption) ([]*models.ChannelGroup, int64, error)
	Update(ctx context.Context, group *models.ChannelGroup) error
	Delete(ctx context.Context, id int64) error
	ReplaceChannels(ctx context.Context, groupID int64, channelIDs []int64) error
	Exists(ctx context.Context, id int64) (bool, error)
	PreLoadRegistryData(ctx context.Context) ([]*models.ChannelGroup, error)
}

type channelGroupRepo struct {
	*Repository
}

func NewChannelGroupRepo(r *Repository) ChannelGroupRepo {
	return &channelGroupRepo{Repository: r}
}

func (r *channelGroupRepo) Create(ctx context.Context, group *models.ChannelGroup) error {
	return Create[models.ChannelGroup](ctx, group, r.DB(ctx))
}

func (r *channelGroupRepo) BatchCreate(ctx context.Context, groups []*models.ChannelGroup) error {
	return BatchCreate[*models.ChannelGroup](ctx, groups, r.DB(ctx))
}

func (r *channelGroupRepo) GetByID(ctx context.Context, id int64) (*models.ChannelGroup, error) {
	return GetByID[models.ChannelGroup](ctx, id, r.DB(ctx))
}

func (r *channelGroupRepo) GetByIDWithChannels(ctx context.Context, id int64) (*models.ChannelGroup, error) {
	return GetByIDWithPreload[models.ChannelGroup](ctx, id, r.DB(ctx), "Channels", nil)
}

func (r *channelGroupRepo) GetByName(ctx context.Context, name string) (*models.ChannelGroup, error) {
	return GetByKey[models.ChannelGroup](ctx, "name", name, r.DB(ctx))
}

func (r *channelGroupRepo) List(ctx context.Context, opt ChannelGroupListOption) ([]*models.ChannelGroup, int64, error) {
	db := r.DB(ctx).Model(&models.ChannelGroup{})
	if opt.PreloadChannels {
		db = db.Preload("Channels")
	}

	keyword := strings.TrimSpace(opt.Keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("name LIKE ? OR name_remark LIKE ? OR description LIKE ?", like, like, like)
	}

	if opt.ChannelID > 0 {
		db = db.Joins("JOIN group_channels gc ON gc.group_id = channel_groups.id").
			Where("gc.channel_id = ?", opt.ChannelID)
	}

	if opt.PermissionNumGE != nil {
		db = db.Where("permission_num >= ?", *opt.PermissionNumGE)
	}

	if len(opt.IDs) > 0 {
		db = db.Where("id IN ?", uniqueInt64(opt.IDs))
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	_, pageSize, offset := normalizePagination(opt.Page, opt.PageSize)
	orderBy := sanitizeOrderBy(opt.OrderBy, "id DESC")

	data := make([]*models.ChannelGroup, 0, pageSize)
	if err := db.Order(orderBy).Offset(offset).Limit(pageSize).Find(&data).Error; err != nil {
		return nil, 0, err
	}
	return data, total, nil
}

func (r *channelGroupRepo) Update(ctx context.Context, group *models.ChannelGroup) error {
	return Update[models.ChannelGroup](ctx, group, group.ID, []string{"ID", "CreatedAt", "DeletedAt", "Channels", "Tokens"}, r.DB(ctx))
}

func (r *channelGroupRepo) Delete(ctx context.Context, id int64) error {
	return Delete[models.ChannelGroup](ctx, id, r.DB(ctx))
}

func (r *channelGroupRepo) ReplaceChannels(ctx context.Context, groupID int64, channelIDs []int64) error {
	db := r.DB(ctx)
	group := new(models.ChannelGroup)
	if err := db.Where("id = ?", groupID).First(group).Error; err != nil {
		return err
	}

	channelIDs = uniqueInt64(channelIDs)
	if len(channelIDs) == 0 {
		return db.Model(group).Association("Channels").Clear()
	}

	channels := make([]*models.Channel, 0, len(channelIDs))
	if err := db.Where("id IN ?", channelIDs).Find(&channels).Error; err != nil {
		return err
	}
	if len(channels) != len(channelIDs) {
		return fmt.Errorf("expected %d channels but got %d", len(channelIDs), len(channels))
	}

	if err := db.Model(group).Association("Channels").Replace(channels); err != nil {
		return err
	}
	return nil
}

func (r *channelGroupRepo) Exists(ctx context.Context, id int64) (bool, error) {
	return Exists[models.ChannelGroup](ctx, id, r.DB(ctx))
}

func (r *channelGroupRepo) PreLoadRegistryData(ctx context.Context) ([]*models.ChannelGroup, error) {
	return gorm.G[[]*models.ChannelGroup](r.DB(ctx)).Preload("Channels", func(db gorm.PreloadBuilder) error {
		db.Select("id")
		return nil
	}).First(ctx)
}
