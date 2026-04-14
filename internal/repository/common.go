package repository

import (
	"regexp"
	"strings"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 200
)

func GetMaxPageSize() int {
	return maxPageSize
}

var orderByPattern = regexp.MustCompile(`^[a-zA-Z0-9_,.\s]+$`)

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

func sanitizeOrderBy(orderBy, fallback string) string {
	orderBy = strings.TrimSpace(orderBy)
	if orderBy == "" {
		return fallback
	}
	if !orderByPattern.MatchString(orderBy) {
		return fallback
	}
	return orderBy
}

func uniqueInt64(values []int64) []int64 {
	if len(values) <= 1 {
		return values
	}
	seen := make(map[int64]struct{}, len(values))
	out := make([]int64, 0, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
