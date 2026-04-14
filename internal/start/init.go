package start

import (
	"context"

	"github.com/RenaLio/tudou/internal/server"
)

func InitApp(m *server.Migrate) error {
	ctx := context.Background()
	var err error
	err = m.Start(ctx)
	return err
}
