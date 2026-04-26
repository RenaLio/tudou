package loadbalancer

const OldRate = 0.7
const NewRate = 1 - OldRate

const SuccessRate = 0.95

const CircuitBreakThreshold = 3 // 前2次失败都只是降权，只有第三次失败才会熔断 (即达到阈值 3 次失败)

func getCircuitBreakThreshold() int {
	return CircuitBreakThreshold
}

func getMaxCircuitBreakThreshold() int {
	return CircuitBreakThreshold + 5
}

func getMinBackoffInterval() int64 {
	return int64(1000 * 60 * 6) // ms
} // ms	6min

func getMaxBackoffInterval() int64 {
	return int64(1000 * 60 * 60 * 60 * 1.5) // ms
} // ms	1.5hr
