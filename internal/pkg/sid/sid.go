package sid

import (
	"time"

	"github.com/RenaLio/tudou/internal/config"
	"github.com/bwmarrin/snowflake"
)

type Sid struct {
	*snowflake.Node
}

func NewSid(conf *config.Config) *Sid {
	startTime := "2001-01-01"
	id64 := conf.Security.Sid.Id
	var st time.Time
	st, err := time.Parse("2006-01-02", startTime)
	if err != nil {
		panic(err)
	}
	snowflake.Epoch = st.UnixNano() / 1000000
	node, err := snowflake.NewNode(id64)
	if err != nil {
		panic(err)
	}
	return &Sid{node}
}

func (s *Sid) GenString() string {
	return s.Generate().String()
}

func (s *Sid) GenInt64() int64 {
	return s.Generate().Int64()
}
