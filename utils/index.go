package utils

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
)


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


func ExtractSpotifyID(url string) string {
	parts := strings.Split(url, "track/")
	if len(parts) < 2 {
		return ""
	}
	id := strings.Split(parts[1], "?")[0]
	return id
}