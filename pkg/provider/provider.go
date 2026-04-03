package provider

import (
	"context"

	"github.com/RenaLio/tudou/pkg/provider/types"
)

type Provider interface {
	Identifier() string
	Execute(ctx context.Context, req *types.Request, cb types.MetricsCallback) (*types.Response, error)
	Models() ([]string, error)
	Abilities() []types.Ability
	HasAbility(ability types.Ability) bool
}
