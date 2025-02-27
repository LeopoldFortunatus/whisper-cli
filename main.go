package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/openai/openai-go"
	"github.com/rs/zerolog/log"
	"whisper"
)

type Segment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

func main() {
	inputFile := flag.String("input", "./whisper/audio/2025_02_15_диспут_Чегон_нельзя_делать_с_Мифом.m4a", "Path to the input audio file")
	language := flag.String("language", "ru", "Language for transcription")
	flag.Parse()

	// Extract the base name of the input file without extension
	baseName := strings.TrimSuffix(filepath.Base(*inputFile), filepath.Ext(*inputFile))

	// Create a directory with the base name
	outputDir := filepath.Join(filepath.Dir(*inputFile), baseName)
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		return
	}

	// Set outputPattern to save chunks in the newly created directory
	outputPattern := filepath.Join(outputDir, "chunk_%03d.m4a")

	fmt.Println("Splitting the audio file into parts...")
	err := whisper.splitAudioFile(*inputFile, outputPattern)
	if err != nil {
		fmt.Printf("Error splitting file: %v\n", err)
		return
	}

	// Initialize the client based on the environment variable
	client := openai.NewClient()

	var allSegments []Segment
	offset := 0.0

	for i := 0; ; i++ {
		chunkFile := fmt.Sprintf(outputPattern, i)
		if _, err := os.Stat(chunkFile); os.IsNotExist(err) {
			break
		}

		// Get the duration of the current chunk
		dur, err := whisper.getDuration(chunkFile)
		if err != nil {
			fmt.Printf("Failed to get duration of file %s: %v\n", chunkFile, err)
			break
		}

		ctx := context.Background()
		segments, err := whisper.transcribeAudio(ctx, client, chunkFile, offset, *language)
		if err != nil {
			log.Fatal().Err(err).Msgf("Error during transcription: %s", chunkFile)
		}
		allSegments = append(allSegments, segments...)

		// Increase the offset by the duration of the k-th chunk
		offset += dur
	}

	// Serialize all segments into a single JSON
	data, err := json.MarshalIndent(allSegments, "", "  ")
	if err != nil {
		fmt.Printf("Error serializing JSON: %v\n", err)
		return
	}
	if err := ioutil.WriteFile(filepath.Join(outputDir, `combined.json`), data, 0644); err != nil {
		fmt.Printf("Error saving file combined.json: %v\n", err)
		return
	}
	fmt.Println("Combined JSON saved to file `combined.json`")

	// Additionally, collect the overall text
	var textResult strings.Builder
	for _, seg := range allSegments {
		textResult.WriteString(seg.Text)
		textResult.WriteString("\n")
	}
	err = ioutil.WriteFile(filepath.Join(outputDir, `transcription.txt`), []byte(textResult.String()), 0644)
	if err != nil {
		fmt.Printf("Error saving file transcription.txt: %v\n", err)
		return
	}
	fmt.Println("Overall text saved to file `transcription.txt`")
}
