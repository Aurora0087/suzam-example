package spectrogram

import (
	"fmt"
	"suzam-example/suzam/fft"
	"suzam-example/suzam/windowing"
)

func CreateSpectrogramFromSample(samples []float32, frameSize, overlap int) ([][]float64, error) {

	frames, err := SliceIntoFrames(samples, frameSize, overlap)

	if err != nil {
		return nil, err
	}

	hannWindow := windowing.GenerateHannWindow(frameSize)

	windowedFrames := windowing.ApplyWindowToFramesSafe(frames, hannWindow)

	spectrogram := make([][]float64, len(windowedFrames))

	// running FFT on every frame
	for i := 0; i < len(windowedFrames); i++ {
		magnitudes := fft.ProcessWindowedFrame(windowedFrames[i])
		spectrogram[i] = magnitudes
	}

	return spectrogram, nil
}

func SliceIntoFrames(samples []float32, frameSize, overlap int) ([][]float32, error) {

	if overlap >= frameSize {
		return nil, fmt.Errorf("Overlap must be smaller than FrameSize")
	}

	hopSize := frameSize - overlap

	numFrames := (len(samples) - overlap) / hopSize

	frames := make([][]float32, numFrames)

	for i := 0; i < numFrames; i++ {
		start := i * hopSize
		end := start + frameSize

		frame := make([]float32, frameSize)

		copy(frame, samples[start:end])

		frames[i] = frame
	}
	return frames, nil
}
