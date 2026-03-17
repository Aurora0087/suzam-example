package main

import (
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"math/cmplx"
	"os"
	"os/exec"
	"path/filepath"
	"suzam-example/db"

	"github.com/google/uuid"
)

func main() {

	sqllitePath := "./suzam.db"
	database, err := db.InitDB(sqllitePath)
	if err != nil {
		panic(err)
	}
	defer database.Close()

	inputFilePath_1 := "./musics/Levitating.wav"
	inputFilePath_2 := "./musics/HymnForTheWeekend.wav"
	outputFolder := "./output-data"

	MakefingarprintFromSong(outputFolder, inputFilePath_1, "Levitating", database)

	MakefingarprintFromSong(outputFolder, inputFilePath_2, "HymnForTheWeekend", database)

	clip := "./musics/Levitating-clip.wav"

	FindSongFromClip(outputFolder, clip, database)
}

func MakefingarprintFromSong(outputFolder, inputFilePath, songTitle string, database *sql.DB) {

	newUUID := uuid.New() // song id

	outputFolderPath := filepath.Join(outputFolder, newUUID.String())

	// convert wav to raw
	fmt.Println("Converting .WAV to .RAW ...")

	rawOutput, err := WavToRaw(inputFilePath, outputFolderPath, newUUID.String())

	if err != nil {
		panic(err)
	}

	// convert raw data to flote32[]

	fmt.Println("Converting .RAW to 32-bit Float Array...")
	samples := RawAudioFileToArray(rawOutput)

	fmt.Println("Total Sample Size : ", len(samples))

	// CreateSpectrogram

	frameSize := 1024

	fmt.Println("Createing Spectrogram where frameSize : ", frameSize)

	spectrogramFrames, err := CreateSpectrogramFromSample(samples, frameSize, frameSize/2)

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("W : ", len(spectrogramFrames), ",H : ", len(spectrogramFrames[0]))

	var minDB float64 = -80
	var maxDB float64 = 0

	err = SaveSpectrogramImage(spectrogramFrames, outputFolderPath+"/"+"spectrogram.png", minDB, maxDB, frameSize)
	if err != nil {
		panic(err)
	}

	p := ExtractPeaksGridOptimized(spectrogramFrames, 21, minDB, maxDB, frameSize)

	fmt.Println("Peaks : ", len(p))

	DrawConstellationMap(p, len(spectrogramFrames), len(spectrogramFrames[0])/2, outputFolderPath+"/"+"peaks.png")

	hashes := GenerateHashes(p)

	fmt.Println("Hashes : ", len(hashes))

	err = db.StoreSong(database, songTitle, hashes)
	if err != nil {
		fmt.Println("Error storing song:", err)
	} else {
		fmt.Println("Successfully indexed song!")
	}
}

func FindSongFromClip(outputFolder, inputFilePath string, database *sql.DB) {

	newUUID := uuid.New() // song id

	outputFolderPath := filepath.Join(outputFolder, newUUID.String())

	// convert wav to raw
	fmt.Println("Converting .WAV to .RAW ...")

	rawOutput, err := WavToRaw(inputFilePath, outputFolderPath, newUUID.String())

	if err != nil {
		panic(err)
	}

	// convert raw data to flote32[]

	fmt.Println("Converting .RAW to 32-bit Float Array...")
	samples := RawAudioFileToArray(rawOutput)

	fmt.Println("Total Sample Size : ", len(samples))

	// CreateSpectrogram

	frameSize := 1024

	fmt.Println("Createing Spectrogram where frameSize : ", frameSize)

	spectrogramFrames, err := CreateSpectrogramFromSample(samples, frameSize, frameSize/2)

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("W : ", len(spectrogramFrames), ",H : ", len(spectrogramFrames[0]))

	var minDB float64 = -80
	var maxDB float64 = 0

	err = SaveSpectrogramImage(spectrogramFrames, outputFolderPath+"/"+"spectrogram.png", minDB, maxDB, frameSize)
	if err != nil {
		panic(err)
	}

	p := ExtractPeaksGridOptimized(spectrogramFrames, 21, minDB, maxDB, frameSize)

	fmt.Println("Peaks : ", len(p))

	DrawConstellationMap(p, len(spectrogramFrames), len(spectrogramFrames[0])/2, outputFolderPath+"/"+"peaks.png")

	hashes := GenerateHashes(p)

	fmt.Println("Hashes : ", len(hashes))

	posibleSongTitle, score, err := db.FindMatch(database, hashes)

	if err != nil {
		fmt.Println("Error Finding song:", err)
	} else {
		fmt.Println("Successfully found song. SongTitle : ", posibleSongTitle, ", score : ", score)
	}
}

func WavToRaw(inputFile, outputFolder, songId string) (string, error) {

	outputPath := filepath.Join(outputFolder, "song.raw")

	err := os.MkdirAll(outputFolder, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("failed to create directory: %v", err)
	}
	cmd := exec.Command("ffmpeg",
		"-i", inputFile,
		"-f", "f32le",
		"-acodec", "pcm_f32le",
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg error: %v\nOutput: %s", err, string(output))
	}

	fmt.Printf("Successfully converted %s to %s\n", inputFile, outputPath)
	return outputPath, nil
}

func RawAudioFileToArray(inputFile string) []float32 {
	file, err := os.Open(inputFile)
	if err != nil {
		panic(err)
	}

	defer file.Close()

	info, _ := file.Stat()

	sampleCount := info.Size() / 4

	samples := make([]float32, sampleCount)

	buf := make([]byte, 4)
	for i := 0; i < int(sampleCount); i++ {
		_, err := io.ReadFull(file, buf)
		if err != nil {
			break
		}
		bits := binary.LittleEndian.Uint32(buf)
		samples[i] = math.Float32frombits(bits)
	}

	return samples

}

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

func CreateFloting32SampleJSON(samples []float32, outPath string) (string, error) {

	f, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("Failed to create file: %w", err)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	err = encoder.Encode(samples)
	if err != nil {
		return "", fmt.Errorf("Failed to encode samples to JSON: %w", err)
	}

	return outPath, nil
}

func CreateSpectrogramFromSample(samples []float32, frameSize, overlap int) ([][]float64, error) {
	// 1. Slice into frames
	// (Note: Ensure your SliceIntoFrames function is updated to return an error if sizes are wrong)
	frames, err := SliceIntoFrames(samples, frameSize, overlap)

	if err != nil {
		return nil, err
	}

	// 2. Generate the Hann Window once
	hannWindow := GenerateHannWindow(frameSize)

	// 3. Apply the window
	// Using the "Safe" version returns a new copy
	windowedFrames := ApplyWindowToFramesSafe(frames, hannWindow)

	// 4. Create the spectrogram storage
	// Height will be frameSize/2, Width will be number of frames
	spectrogram := make([][]float64, len(windowedFrames))

	// 5. Run FFT on every frame
	for i := 0; i < len(windowedFrames); i++ {
		// ProcessFrame returns []float64 (the magnitudes)
		magnitudes := ProcessFrame(windowedFrames[i])
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

func FFT(a []complex128) []complex128 {
	n := len(a)
	if n <= 1 {
		return a
	}

	// spliting into even and odd
	even := make([]complex128, n/2)
	odd := make([]complex128, n/2)

	for i := 0; i < n/2; i++ {
		even[i] = a[2*i]
		odd[i] = a[2*i+1]
	}

	// d.c.
	evenRes := FFT(even)
	oddRes := FFT(odd)

	// combine using twiddle factors
	result := make([]complex128, n)
	for k := 0; k < n/2; k++ {
		//twiddle factor: e^(-2*PI*i*k / N)
		angle := -2 * math.Pi * float64(k) / float64(n)
		twiddle := cmplx.Exp(complex(0, angle)) * oddRes[k]

		result[k] = evenRes[k] + twiddle
		result[k+n/2] = evenRes[k] - twiddle
	}
	return result
}

func ProcessFrame(frame []float32) []float64 {
	n := len(frame)

	input := make([]complex128, n)
	for i, val := range frame {
		input[i] = complex(float64(val), 0)
	}

	fftResult := FFT(input)

	magnitudes := make([]float64, n/2)
	for i := 0; i < n/2; i++ {
		//magnitude = sqrt(real^2 + imag^2)
		magnitudes[i] = cmplx.Abs(fftResult[i])

	}

	return magnitudes
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

	// 1. Find the Min and Max values in Decibels
	// This helps us normalize the colors so the image isn't too dark or too bright
	dbSpectrogram := make([][]float64, w)
	maxDB := -math.MaxFloat64
	minDB := math.MaxFloat64

	for x := 0; x < w; x++ {
		dbSpectrogram[x] = make([]float64, h)
		for y := 0; y < h; y++ {
			// Convert to Decibels: 20 * log10(mag)
			db := 20 * math.Log10(spectrogram[x][y]+1e-1)
			dbSpectrogram[x][y] = db

			if db > maxDB {
				maxDB = db
			}
			if db < minDB {
				minDB = db
			}
			// 2. Normalize intensity between 0.0 and 1.0
			intensity := (db - minDB) / (maxDB - minDB)

			// 3. Create a "Inferno/Heatmap" color
			// High intensity = Yellow, Medium = Red, Low = Black/Blue
			var c color.RGBA
			if intensity > 0.5 {
				// Transition from Red to Yellow
				c = color.RGBA{255, uint8((intensity - 0.5) * 2 * 255), 0, 255}
			} else {
				// Transition from Black to Red
				c = color.RGBA{uint8(intensity * 2 * 255), 0, 0, 255}
			}

			// 4. Set pixel (Flipped Y)
			img.Set(x, displayHeight-1-y, c)
		}
	}

	// 4. Encode to PNG
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

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

func DrawConstellationMap(peaks []Peak, w, h int, outPath string) error {

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

func GenerateHashes(peaks []Peak) []db.Fingerprint {
	fingerprints := []db.Fingerprint{}

	const (
		TargetZoneTimeStart = 5
		TargetZoneTimeEnd   = 30
	)

	for i := 0; i < len(peaks); i++ {
		anchor := peaks[i]

		for j := i + 1; j < len(peaks); j++ {
			target := peaks[j]

			timeDiff := target.Frame - anchor.Frame

			if timeDiff < TargetZoneTimeStart {
				continue
			}
			if timeDiff > TargetZoneTimeEnd {
				break
			}

			// Anchor Freq (9 bits), Target Freq (9 bits), Time Diff (14 bits)
			hash := uint32(anchor.Bin&0x1FF)<<23 |
				uint32(target.Bin&0x1FF)<<14 |
				uint32(timeDiff&0x3FFF)

			fingerprints = append(fingerprints, db.Fingerprint{
				Hash:       hash,
				AnchorTime: anchor.Frame,
			})
		}
	}
	return fingerprints
}
