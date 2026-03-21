package constellation

import "math"


type Peak struct {
	Frame int     // Time (X axis)
	Bin   int     // Frequency (Y axis)
	Value float64 // Amplitude (dB)
}

func ExtractPeaksGridOptimized(spectrogram [][]float64, neighborhoodSize int, minDB, maxDB float64, frameSize int) []Peak {
	w := len(spectrogram)
	h := len(spectrogram[0])

	peaks := []Peak{}

	normFactor := float64(frameSize) / 2.0

	for x := 0; x < w; x += neighborhoodSize {
		for y := 0; y < h; y += neighborhoodSize {

			blockMax := Peak{Value: -999}
			found := false

			for dx := 0; dx < neighborhoodSize; dx++ {
				for dy := 0; dy < neighborhoodSize; dy++ {
					currX := x + dx
					currY := y + dy

					if currX >= w || currY >= h {
						continue
					}

					mag := spectrogram[currX][currY] / normFactor

					db := 20 * math.Log10(mag+1e-9)

					if db < minDB || db > maxDB {
						continue
					}

					if db > blockMax.Value {
						blockMax = Peak{
							Frame: currX,
							Bin:   currY,
							Value: db,
						}
						found = true
					}
				}
			}

			if found {
				peaks = append(peaks, blockMax)
			}
		}
	}

	return peaks
}