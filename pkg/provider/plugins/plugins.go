package plugins

import "github.com/RenaLio/tudou/pkg/provider/types"

// ApplyPlugins wraps the base invoker with plugins from back to front so the
// first plugin in the list becomes the outermost layer.
func ApplyPlugins(base types.Invoker, plugins ...types.Plugin) types.Invoker {
	invoker := base
	for i := len(plugins) - 1; i >= 0; i-- {
		invoker = plugins[i](invoker)
	}
	return invoker
}
