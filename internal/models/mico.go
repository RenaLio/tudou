package models

import "math"

const (
	// moneyMicrosPerUnit 表示 1 个货币单位 = 1_000_000_000 micros。
	// 实际采用了十亿级精度 (Nano级别: 1,000,000,000)，为保持命名习惯仍称为 Micros
	moneyMicrosPerUnit int64 = 1_000_000_000
	// pricingTokenUnit 表示定价字段按每 1M tokens 计价。
	pricingTokenUnit int64 = 1_000_000
)

func moneyFloatToMicros(amount float64) int64 {
	return int64(math.Round(amount * float64(moneyMicrosPerUnit)))
}

func moneyMicrosToFloat(amountMicros int64) float64 {
	return float64(amountMicros) / float64(moneyMicrosPerUnit)
}

func pricingPerMillionToMicros(price float64) int64 {
	return moneyFloatToMicros(price)
}
