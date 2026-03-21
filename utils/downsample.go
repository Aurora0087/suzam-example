package utils

func DownsampleForWeb(samples []float32, targetDataPoints int) []float32 {
	if len(samples) <= targetDataPoints {
		return samples
	}

	peaks := make([]float32, targetDataPoints)
	chunkSize := len(samples) / targetDataPoints

	for i := 0; i < targetDataPoints; i++ {
		start := i * chunkSize
		end := start + chunkSize

		var max float32 = 0
		for j := start; j < end && j < len(samples); j++ {
			val := samples[j]
			if val < 0 {
				val = -val
			}
			if val > max {
				max = val
			}
		}
		peaks[i] = max
	}

	return peaks
}