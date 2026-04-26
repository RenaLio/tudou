package service

import (
	"context"
	"strconv"

	"github.com/RenaLio/tudou/internal/constants"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 200
)

func normalizePagination(page, pageSize int) (int, int, int) {
	if page <= 0 {
		page = defaultPage
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	offset := (page - 1) * pageSize
	return page, pageSize, offset
}

func int64ToString(v int64) string {
	return strconv.FormatInt(v, 10)
}

func stringToInt64(v string) (int64, error) {
	return strconv.ParseInt(v, 10, 64)
}

func getRequestId(ctx context.Context) string {
	value := ctx.Value(constants.RequestIdKey())
	if value == nil {
		return ""
	}
	if value, ok := value.(string); ok {
		return value
	}
	return ""
}
