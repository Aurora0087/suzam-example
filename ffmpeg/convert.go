package ffmpeg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

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


func ConvertToWav(inputPath, outputPath string) (string,error) {
	cmd := exec.Command("ffmpeg", 
		"-i", inputPath, 
		"-y", 
		"-vn", 
		"-acodec", "pcm_s16le", 
		"-ar", "44100", 
		"-ac", "2", 
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "",fmt.Errorf("ffmpeg error: %v\nOutput: %s", err, string(output))
	}

	return outputPath,nil
}