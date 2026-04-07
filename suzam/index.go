package suzam

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"suzam-example/db"
	"suzam-example/ffmpeg"
	"suzam-example/suzam/constellation"
	"suzam-example/suzam/hash"
	"suzam-example/suzam/spectrogram"
	"suzam-example/utils"

	"github.com/google/uuid"
)

func MakefingarprintFromSong(id int, outputFolder, inputFilePath, songTitle, spotify_id, authors string, duration float64, database *sql.DB) {

	outputFolderPath := filepath.Join(outputFolder, strconv.Itoa(id))

	// convert wav to raw
	fmt.Println("Converting .WAV to .RAW ...")

	rawOutput, err := ffmpeg.WavToRaw(inputFilePath, outputFolderPath, strconv.Itoa(id))

	if err != nil {
		panic(err)
	}

	// deletee orginal song file
	err = os.Remove(inputFilePath)
	if err != nil {
		fmt.Println("Error fail to delete .wav file! path : ", inputFilePath)
	}

	// convert raw data to flote32[]

	fmt.Println("Converting .RAW to 32-bit Float Array...")
	samples := utils.RawAudioFileToArray(rawOutput)

	fmt.Println("Total Sample Size : ", len(samples))

	// CreateSpectrogram

	frameSize := 1024

	fmt.Println("Createing Spectrogram where frameSize : ", frameSize)

	spectrogramFrames, err := spectrogram.CreateSpectrogramFromSample(samples, frameSize, frameSize/2)

	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("W : ", len(spectrogramFrames), ",H : ", len(spectrogramFrames[0]))

	var minDB float64 = -80
	var maxDB float64 = 0

	err = utils.SaveSpectrogramImage(spectrogramFrames, outputFolderPath+"/"+"spectrogram.png", minDB, maxDB, frameSize)
	if err != nil {
		panic(err)
	}

	p := constellation.ExtractPeaksGridOptimized(spectrogramFrames, 15, minDB, maxDB, frameSize)

	fmt.Println("Peaks : ", len(p))

	utils.DrawConstellationMap(p, len(spectrogramFrames), len(spectrogramFrames[0])/2, outputFolderPath+"/"+"peaks.png")

	hashes := hash.GenerateHashes(p)

	fmt.Println("Hashes : ", len(hashes))

	songDetails := db.Song{
		SpotifyID: spotify_id,
		Title:     songTitle,
		Authors:   authors,
		Duration:  duration,
		ID:        id,
	}

	songId, err := db.StoreSong(database, songDetails, hashes)
	if err != nil {
		fmt.Println("Error storing song:", err)
	} else {
		fmt.Println("Successfully indexed song with Id!", songId)
	}

	// delete rawOutput
	err = os.Remove(rawOutput)
	if err != nil {
		fmt.Println("Error deleteing .raw file, song with Id!", songId)
	}
}

func FindSongFromClip(outputFolder, inputFilePath string, database *sql.DB) ([]db.SongWithMatchScore, error) {

	newUUID := uuid.New()

	outputFolderPath := filepath.Join(outputFolder, newUUID.String())

	// convert wav to raw
	fmt.Println("Converting .WAV to .RAW ...")

	rawOutput, err := ffmpeg.WavToRaw(inputFilePath, outputFolderPath, newUUID.String())

	if err != nil {
		panic(err)
	}

	// convert raw data to flote32[]

	fmt.Println("Converting .RAW to 32-bit Float Array...")
	samples := utils.RawAudioFileToArray(rawOutput)

	fmt.Println("Total Sample Size : ", len(samples))

	os.RemoveAll(outputFolderPath)

	// CreateSpectrogram

	frameSize := 1024

	fmt.Println("Createing Spectrogram where frameSize : ", frameSize)

	spectrogramFrames, err := spectrogram.CreateSpectrogramFromSample(samples, frameSize, frameSize/2)

	if err != nil {
		fmt.Println(err)
		return []db.SongWithMatchScore{}, err
	}

	fmt.Println("W : ", len(spectrogramFrames), ",H : ", len(spectrogramFrames[0]))

	var minDB float64 = -80
	var maxDB float64 = 0

	/*err = utils.SaveSpectrogramImage(spectrogramFrames, outputFolderPath+"/"+"spectrogram.png", minDB, maxDB, frameSize)
	if err != nil {
		panic(err)
	}*/

	p := constellation.ExtractPeaksGridOptimized(spectrogramFrames, 15, minDB, maxDB, frameSize)

	fmt.Println("Peaks : ", len(p))

	//utils.DrawConstellationMap(p, len(spectrogramFrames), len(spectrogramFrames[0])/2, outputFolderPath+"/"+"peaks.png")

	hashes := hash.GenerateHashesForClip(p)

	fmt.Println("Hashes : ", len(hashes))

	top5Matchs, err := db.FindTop5Matchs(database, hashes)

	if err != nil {
		fmt.Println("Error Finding song:", err)
		return []db.SongWithMatchScore{}, err
	}
	fmt.Println("Successfully found song. Here is top matchs : ", top5Matchs)
	return top5Matchs, nil
}
