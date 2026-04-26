package loadbalancer

// Normalize TTFT score to a value between 0 and 1000
func normalizeTTFTScore(num int64) int {
	if num <= 0 {
		return 1000
	}
	if num >= 10000 {
		return 0
	}
	return int((10000 - num) * 1000 / 10000)
}

// Normalize TPS score to a value between 0 and 1000
func normalizeTPSScore(num int64) int {
	if num <= 0 {
		return 0
	}
	if num >= 1000 {
		return 1000
	}
	return int(num * 1000 / 1000)
}

func normalizeWeight(weight int64) int {
	if weight <= 0 {
		return 0
	}
	weight = weight * 10
	if weight >= 1000 {
		return 1000
	}
	return int(weight)
}
