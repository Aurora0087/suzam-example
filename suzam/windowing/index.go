package windowing

import "math"



func GenerateHannWindow(size int) []float32 {
	window := make([]float32, size)
	for i := 0; i < size; i++ {
		//hann formula: 0.5 * (1 - cos(2 * PI * i / (size - 1)))
		val := 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(size-1)))
		window[i] = float32(val)
	}
	return window
}

func ApplyWindowToFramesSafe(frames [][]float32, window []float32) [][]float32 {
	newFrames := make([][]float32, len(frames))
	for f := 0; f < len(frames); f++ {
		newFrames[f] = make([]float32, len(frames[f]))
		for i := 0; i < len(frames[f]); i++ {
			newFrames[f][i] = frames[f][i] * window[i]
		}
	}
	return newFrames
}