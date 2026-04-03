package main

import (
	"context"
	"fmt"
	"time"

	"github.com/RenaLio/tudou/pkg/provider"
	"github.com/RenaLio/tudou/pkg/provider/types"
)

// ApplyPlugins 将洋葱皮 (plugins) 一层层套在洋葱心 (baseSDK) 上
func ApplyPlugins(baseSDK types.Invoker, plugins ...types.Plugin) types.Invoker {
	// 核心调用作为初始的 invoker
	invoker := baseSDK

	// 重点：从后往前遍历套娃，保证数组前面的插件在最外层执行
	for i := len(plugins) - 1; i >= 0; i-- {
		invoker = plugins[i](invoker)
	}

	return invoker
}

func PromptInjector(systemPrompt string) types.Plugin {
	return func(next types.Invoker) types.Invoker {
		// 返回被包装后的新函数
		return func(ctx context.Context, req *types.Request, cb types.MetricsCallback) (*types.Response, error) {

			newCb := func(metrics *types.ResponseMetrics) {
				if metrics.Extra == nil {
					metrics.Extra = make(map[string]any)
				}
				metrics.Extra["input_format"] = req.Format
				cb(metrics)
			}

			req.Headers.Set("X-Timestamp", "1694502400000")
			//fmt.Println(2222)

			// 没有任何后置逻辑，直接 return next() 的结果
			return next(ctx, req, newCb)
		}
	}
}

func PluginDurationLogger(pluginName string) types.Plugin {
	return func(next types.Invoker) types.Invoker {
		return func(ctx context.Context, req *types.Request, cb types.MetricsCallback) (*types.Response, error) {

			// 【前置逻辑】：记录开始时间
			start := time.Now()
			//fmt.Println(1111)

			// 执行下一层洋葱（可能是下一个插件，或者是底层的真实发包）
			resp, err := next(ctx, req, cb)

			_ = start
			// 【后置逻辑】：计算耗时
			//log.Printf("Plugin [%s] executed in %v", pluginName, time.Since(start))

			return resp, err
		}
	}
}

type RelayService struct {
	Client provider.Provider
}

func NewRelayService(client provider.Provider) *RelayService {
	return &RelayService{Client: client}
}

func (s *RelayService) RelayServiceFunc(ctx context.Context, req *types.Request) (*types.Response, error) {
	cb := func(metrics *types.ResponseMetrics) {
		fmt.Printf("metrics: %#v\n", metrics)
		// 处理响应
	}
	finalInvoker := ApplyPlugins(
		s.Client.Execute,
		PluginDurationLogger("PromptPlugin"),
		PromptInjector("你是专业的AI助手"),
	)
	return finalInvoker(ctx, req, cb)
}
