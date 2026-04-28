package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

func Save[T any](ctx context.Context, item *T, db *gorm.DB) error {
	//if item == nil {
	//	return errors.New("item is nil")
	//}
	return db.Save(item).Error
}

func Create[T any](ctx context.Context, inputItem *T, db *gorm.DB) error {
	//if inputItem == nil {
	//	return errors.New("input item is nil")
	//}
	return gorm.G[T](db).Create(ctx, inputItem)
}

func BatchCreate[T any](ctx context.Context, inputItems []T, db *gorm.DB) error {
	if len(inputItems) == 0 {
		return nil
	}
	return gorm.G[T](db).CreateInBatches(ctx, &inputItems, 100)
}

func GetByID[T any](ctx context.Context, id int64, db *gorm.DB) (*T, error) {
	item, err := gorm.G[T](db).Where("id = ?", id).First(ctx)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func GetByIDs[T any](ctx context.Context, ids []int64, db *gorm.DB) ([]T, error) {
	return gorm.G[T](db).Where("id IN ?", ids).Find(ctx)
}

func GetByKey[T any](ctx context.Context, filterKey string, filterVal string, db *gorm.DB) (*T, error) {
	if filterKey == "" {
		return nil, errors.New("filterKey is required")
	}
	item, err := gorm.G[T](db).Where(fmt.Sprintf("%s = ?", filterKey), filterVal).First(ctx)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func GetSomeByKey[T any](ctx context.Context, filterKey string, filterVal string, db *gorm.DB) ([]T, error) {
	if filterKey == "" {
		return nil, errors.New("filterKey is required")
	}
	return gorm.G[T](db).Where(fmt.Sprintf("%s = ?", filterKey), filterVal).Find(ctx)
}

func GetByIDWithPreload[T any](ctx context.Context, id int64, db *gorm.DB, preload string, query func(db gorm.PreloadBuilder) error) (*T, error) {
	item, err := gorm.G[T](db).Where("id = ?", id).Preload(preload, query).First(ctx)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func GetByIDWithPreloads[T any](ctx context.Context, id int64, db *gorm.DB, preloads ...string) (*T, error) {
	temp := gorm.G[T](db).Where("id = ?", id)
	for _, preload := range preloads {
		temp = temp.Preload(preload, nil)
	}
	item, err := temp.First(ctx)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func Update[T any](ctx context.Context, item *T, id int64, omits []string, db *gorm.DB) error {
	_, err := gorm.G[T](db).Where("id = ?", id).Select("*").Omit(omits...).Updates(ctx, *item)
	return err
}

func SetField[T any](ctx context.Context, field string, value any, id int64, db *gorm.DB) error {
	if field == "" {
		return errors.New("field is required")
	}
	_, err := gorm.G[T](db).Where("id = ?", id).Update(ctx, field, value)
	return err
}

func Delete[T any](ctx context.Context, id int64, db *gorm.DB) error {
	//if id <= 0 {
	//	return errors.New("invalid id")
	//}
	_, err := gorm.G[T](db).Where("id = ?", id).Delete(ctx)
	return err
}

func Exists[T any](ctx context.Context, id int64, db *gorm.DB) (bool, error) {
	//if id <= 0 {
	//	return false, errors.New("invalid id")
	//}
	count, err := gorm.G[T](db).Where("id = ?", id).Count(ctx, "id")
	if err != nil {
		return false, err
	}
	return count > 0, err
}

func IsNotFound[T any](err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
