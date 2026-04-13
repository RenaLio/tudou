package loadbalancer

const OldRate = 0.7
const NewRate = 1 - OldRate

const SuccessRate = 0.95

func getCircuitBreakThreshold() int {
	return 5
}

func getMinBackoffInterval() int64 {
	return int64(1000 * 60 * 6) // ms
}

func getMaxBackoffInterval() int64 {
	return int64(1000 * 60 * 60 * 60 * 1.5) // ms
}
