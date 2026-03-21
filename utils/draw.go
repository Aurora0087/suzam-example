package utils

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"suzam-example/suzam/constellation"
)



func DrawWaveform(samples []float32, width, height int, outPath string) (string, error) {

	f, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	//bg black
	draw.Draw(img, img.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Src)

	//horizontal pixel
	samplesPerPixel := len(samples) / width

	centerY := float64(height) / 2
	amplitudeScale := float64(height) / 2

	for x := 0; x < width; x++ {
		start := x * samplesPerPixel
		end := start + samplesPerPixel
		var max float32
		for i := start; i < end && i < len(samples); i++ {
			absVal := float32(math.Abs(float64(samples[i])))
			if absVal > max {
				max = absVal
			}
		}

		// Draw a vertical line for this pixel
		lineHeight := int(float64(max) * amplitudeScale)
		for y := int(centerY) - lineHeight; y <= int(centerY)+lineHeight; y++ {
			if y >= 0 && y < height {
				img.Set(x, y, color.RGBA{0, 255, 255, 255}) // Cyan
			}
		}
	}

	f, err = os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("Failed to create file: %w", err)
	}
	defer f.Close()
	png.Encode(f, img)
	return outPath, nil
}

func SaveSpectrogramImage(spectrogram [][]float64, outPath string, minAllowedDB, maxAllowedDB float64, frameSize int) error {
	width := len(spectrogram)
	height := len(spectrogram[0])

	displayHeight := height / 2

	img := image.NewRGBA(image.Rect(0, 0, width, displayHeight))

	normFactor := float64(frameSize) / 2.0

	for x := 0; x < width; x++ {
		for y := 0; y < displayHeight; y++ {

			// normalize the magnitude, raw FFT magnitude divided by 2 and bring 1.0 amplitude to 0dB
			mag := spectrogram[x][y] / normFactor

			//converting to Decibels
			db := 20 * math.Log10(mag+1e-9)

			if db < minAllowedDB {
				db = minAllowedDB
			}
			if db > maxAllowedDB {
				db = maxAllowedDB
			}

			//normalize intensity (0.0 to 1.0)
			intensity := (db - minAllowedDB) / (maxAllowedDB - minAllowedDB)

			// heatmap color mapping
			var c color.RGBA
			switch {
			case intensity < 0.2: // Black to Purple
				c = color.RGBA{uint8(intensity * 5 * 50), 0, uint8(intensity * 5 * 100), 255}
			case intensity < 0.7: // Purple to Red
				t := (intensity - 0.2) / 0.5
				c = color.RGBA{uint8(50 + t*205), 0, uint8(100 - t*100), 255}
			case intensity < 0.95: // Red to Yellow
				t := (intensity - 0.7) / 0.25
				c = color.RGBA{255, uint8(t * 255), 0, 255}
			default: // Yellow to White (The Peaks!)
				t := (intensity - 0.95) / 0.05
				c = color.RGBA{255, 255, uint8(t * 255), 255}
			}

			// Set pixel and flipping
			img.Set(x, displayHeight-1-y, c)
		}
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}


func SaveFullSpectrogramImage(spectrogram [][]float64, outPath string) error {
	w := len(spectrogram)
	h := len(spectrogram[0])

	displayHeight := h / 2

	img := image.NewRGBA(image.Rect(0, 0, w, displayHeight))

	dbSpectrogram := make([][]float64, w)
	maxDB := -math.MaxFloat64
	minDB := math.MaxFloat64

	for x := 0; x < w; x++ {
		dbSpectrogram[x] = make([]float64, h)
		for y := 0; y < h; y++ {
			//convert to Decibels: 20 * log10(mag)
			db := 20 * math.Log10(spectrogram[x][y]+1e-1)
			dbSpectrogram[x][y] = db

			if db > maxDB {
				maxDB = db
			}
			if db < minDB {
				minDB = db
			}
			//normalize
			intensity := (db - minDB) / (maxDB - minDB)

			//heatmap
			var c color.RGBA
			if intensity > 0.5 {
				// Transition from Red to Yellow
				c = color.RGBA{255, uint8((intensity - 0.5) * 2 * 255), 0, 255}
			} else {
				// Transition from Black to Red
				c = color.RGBA{uint8(intensity * 2 * 255), 0, 0, 255}
			}

			img.Set(x, displayHeight-1-y, c)
		}
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}


func DrawConstellationMap(peaks []constellation.Peak, w, h int, outPath string) error {

	img := image.NewRGBA(image.Rect(0, 0, w, h))

	//bg Black
	draw.Draw(img, img.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Src)

	// white peak
	white := color.RGBA{255, 255, 255, 255}

	//draw each peak as a white pixel
	for _, p := range peaks {
		//within the image bounds
		if p.Frame >= 0 && p.Frame < w && p.Bin >= 0 && p.Bin < h {

			img.Set(p.Frame, h-1-p.Bin, white)
		}
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}