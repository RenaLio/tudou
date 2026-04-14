package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/RenaLio/tudou/internal/models"
)

type ChannelListOption struct {
	Page          int
	PageSize      int
	OrderBy       string
	Keyword       string
	GroupID       int64
	Type          models.ChannelType
	Status        *int
	IDs           []int64
	OnlyAvailable bool
	PreloadGroups bool
}

type ChannelRepo interface {
	Create(ctx context.Context, channel *models.Channel) error
	BatchCreate(ctx context.Context, channels []*models.Channel) error
	GetByID(ctx context.Context, id int64) (*models.Channel, error)
	GetByIDWithGroups(ctx context.Context, id int64) (*models.Channel, error)
	GetByIDs(ctx context.Context, ids []int64) ([]*models.Channel, error)
	GetByName(ctx context.Context, name string) (*models.Channel, error)
	List(ctx context.Context, opt ChannelListOption) ([]*models.Channel, int64, error)
	Update(ctx context.Context, channel *models.Channel) error
	UpdateStatus(ctx context.Context, id int64, status int) error
	Delete(ctx context.Context, id int64) error
	ReplaceGroups(ctx context.Context, channelID int64, groupIDs []int64) error
	Exists(ctx context.Context, id int64) (bool, error)
	Save(ctx context.Context, channel *models.Channel) error
	FirstBy(ctx context.Context, condition string, args ...any) (*models.Channel, error)
}

type channelRepo struct {
	*Repository
}

func NewChannelRepo(r *Repository) ChannelRepo {
	return &channelRepo{Repository: r}
}

func (r *channelRepo) Create(ctx context.Context, channel *models.Channel) error {
	return Create[models.Channel](ctx, channel, r.DB(ctx))
}

func (r *channelRepo) BatchCreate(ctx context.Context, channels []*models.Channel) error {
	return BatchCreate[*models.Channel](ctx, channels, r.DB(ctx))
}

func (r *channelRepo) GetByID(ctx context.Context, id int64) (*models.Channel, error) {
	return GetByID[models.Channel](ctx, id, r.DB(ctx))
}

func (r *channelRepo) GetByIDWithGroups(ctx context.Context, id int64) (*models.Channel, error) {
	return GetByIDWithPreload[models.Channel](ctx, id, r.DB(ctx), "Groups", nil)
}

func (r *channelRepo) GetByIDs(ctx context.Context, ids []int64) ([]*models.Channel, error) {
	ids = uniqueInt64(ids)
	return GetByIDs[*models.Channel](ctx, ids, r.DB(ctx))
	//if len(ids) == 0 {
	//	return []*models.Channel{}, nil
	//}
	//data, err := gorm.G[*models.Channel](r.DB(ctx)).Where("id IN ?", ids).Find(ctx)
	//if err != nil {
	//	return nil, err
	//}
	//return data, nil
}

func (r *channelRepo) GetByName(ctx context.Context, name string) (*models.Channel, error) {
	return GetByKey[models.Channel](ctx, "name", name, r.DB(ctx))
}

func (r *channelRepo) List(ctx context.Context, opt ChannelListOption) ([]*models.Channel, int64, error) {
	db := r.DB(ctx).Model(&models.Channel{})
	if opt.PreloadGroups {
		db = db.Preload("Groups")
	}

	keyword := strings.TrimSpace(opt.Keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where(
			"name LIKE ? OR base_url LIKE ? OR tag LIKE ? OR model LIKE ? OR custom_model LIKE ?",
			like, like, like, like, like,
		)
	}

	if opt.GroupID > 0 {
		db = db.Joins("JOIN group_channels gc ON gc.channel_id = channels.id").
			Where("gc.group_id = ?", opt.GroupID)
	}

	if opt.Type != "" {
		db = db.Where("type = ?", opt.Type)
	}

	if opt.Status != nil {
		db = db.Where("status = ?", *opt.Status)
	}

	if opt.OnlyAvailable {
		now := time.Now()
		db = db.Where("status = 1").Where("expired_at IS NULL OR expired_at > ?", now)
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

	data := make([]*models.Channel, 0, pageSize)
	if err := db.Order(orderBy).Offset(offset).Limit(pageSize).Find(&data).Error; err != nil {
		return nil, 0, err
	}
	return data, total, nil
}

func (r *channelRepo) Update(ctx context.Context, channel *models.Channel) error {
	return Update[models.Channel](ctx, channel, channel.ID, []string{"ID", "CreatedAt", "DeletedAt", "Groups", "Stats"}, r.DB(ctx))
}

func (r *channelRepo) UpdateStatus(ctx context.Context, id int64, status int) error {
	return SetField[models.Channel](ctx, "status", status, id, r.DB(ctx))
}

func (r *channelRepo) Delete(ctx context.Context, id int64) error {
	return Delete[models.Channel](ctx, id, r.DB(ctx))
}

func (r *channelRepo) ReplaceGroups(ctx context.Context, channelID int64, groupIDs []int64) error {
	db := r.DB(ctx)
	channel := new(models.Channel)
	if err := db.Where("id = ?", channelID).First(channel).Error; err != nil {
		return err
	}

	groupIDs = uniqueInt64(groupIDs)
	if len(groupIDs) == 0 {
		return db.Model(channel).Association("Groups").Clear()
	}

	groups := make([]*models.ChannelGroup, 0, len(groupIDs))
	if err := db.Where("id IN ?", groupIDs).Find(&groups).Error; err != nil {
		return err
	}
	if len(groups) != len(groupIDs) {
		return fmt.Errorf("expected %d groups but got %d", len(groupIDs), len(groups))
	}

	return db.Model(channel).Association("Groups").Replace(groups)
}

func (r *channelRepo) Exists(ctx context.Context, id int64) (bool, error) {
	return Exists[models.Channel](ctx, id, r.DB(ctx))
}

func (r *channelRepo) Save(ctx context.Context, channel *models.Channel) error {
	return Save[models.Channel](ctx, channel, r.DB(ctx))
}

func (r *channelRepo) FirstBy(ctx context.Context, condition string, args ...any) (*models.Channel, error) {
	channel := new(models.Channel)
	if err := r.DB(ctx).Where(condition, args...).First(channel).Error; err != nil {
		return nil, err
	}
	return channel, nil
}
