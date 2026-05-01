package types

import "github.com/RenaLio/tudou/pkg/provider/types"

type RelayMeta struct {
	Format    types.Format
	TokenID   int64
	TokenName string
	UserID    int64
	GroupID   int64
	GroupName string
	Strategy  string
	Extra     MetaExtra
}

type MetaExtra struct {
	Path string
	IP   string
}
