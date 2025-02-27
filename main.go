package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arykalin/whisper-cli/whisper"
	"github.com/openai/openai-go"
	"github.com/rs/zerolog/log"
)

func main() {
	inputFile := flag.String("input", "", "Path to the input audio file")
	language := flag.String("language", "ru", "Language for transcription")
	flag.Parse()

	// Create a folder named the same as the transcription file but without the extension
	outputDir := strings.TrimSuffix(*inputFile, filepath.Ext(*inputFile))
	err := os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		return
	}

	// Update the outputPattern to save chunks in the created folder
	outputPattern := filepath.Join(outputDir, "chunk_%03d.m4a")

	fmt.Println("Splitting the audio file into parts...")
	err = whisper.SplitAudioFile(*inputFile, outputPattern)
	if err != nil {
		fmt.Printf("Error splitting file: %v\n", err)
		return
	}

	// Initialize the client based on the environment variable
	client := openai.NewClient()

	var allSegments []whisper.Segment
	offset := 0.0

	for i := 0; ; i++ {
		chunkFile := fmt.Sprintf(outputPattern, i)
		if _, err := os.Stat(chunkFile); os.IsNotExist(err) {
			break
		}

		// Get the duration of the current chunk
		dur, err := whisper.GetDuration(chunkFile)
		if err != nil {
			fmt.Printf("Failed to get duration of file %s: %v\n", chunkFile, err)
			break
		}

		ctx := context.Background()
		segments, err := whisper.TranscribeAudio(ctx, client, chunkFile, offset, *language)
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
	if err := os.WriteFile(filepath.Join(outputDir, `combined.json`), data, 0644); err != nil {
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
	err = os.WriteFile(filepath.Join(outputDir, `transcription.txt`), []byte(textResult.String()), 0644)
	if err != nil {
		fmt.Printf("Error saving file transcription.txt: %v\n", err)
		return
	}
	fmt.Println("Overall text saved to file `transcription.txt`")
}
