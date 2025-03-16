package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/arykalin/whisper-cli/whisper"
	"github.com/openai/openai-go"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type Config struct {
	InputFile string `yaml:"input_file"`
	Language  string `yaml:"language,omitempty"` // Optional language field
}

func main() {
	configPath := flag.String("config", "config.yaml", "Path to the config file")
	inputFlag := flag.String("input", "", "Path to the input audio file")
	flag.Parse()

	config, err := loadConfig(*configPath)
	if err != nil {
		log.Printf("Ошибка загрузки конфигурации: %v\n", err)
		return
	}

	// If the input file is not provided in the config, check the command line flag
	inputFile := config.InputFile
	if *inputFlag != "" {
		inputFile = *inputFlag
	}

	if inputFile == "" {
		log.Fatal().Msg("No input file specified. Use -input flag or set input_file in config.yaml.")
	}

	// Create the output directory based on the input file name
	outputDir := strings.TrimSuffix(inputFile, filepath.Ext(inputFile))
	err = os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		log.Printf("Error creating output directory: %v\n", err)
		return
	}

	// Update the outputPattern to save chunks in the created folder
	outputPattern := filepath.Join(outputDir, "chunk_%03d.m4a")

	fmt.Println("Splitting the audio file into chunks...")
	err = whisper.SplitAudioFile(inputFile, outputPattern)
	if err != nil {
		log.Printf("Error splitting file: %v\n", err)
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
			log.Printf("Failed to get duration of file %s: %v\n", chunkFile, err)
			break
		}

		ctx := context.Background()
		segments, err := func() ([]whisper.Segment, error) {
			// Check if intermediate file exists
			intermediateFile := strings.TrimSuffix(chunkFile, filepath.Ext(chunkFile)) + "_transcription.json"
			if data, err := os.ReadFile(intermediateFile); err == nil {
				// File exists, try to unmarshal
				var segments []whisper.Segment
				if err := json.Unmarshal(data, &segments); err == nil {
					log.Printf("Loading existing transcription for %s\n", chunkFile)
					return segments, nil
				}
				// If unmarshalling fails, continue with transcription
			}

			// Transcribe and save result
			segments, err := whisper.TranscribeAudio(ctx, client, chunkFile, offset, config.Language)
			if err != nil {
				return nil, err
			}

			// Save intermediate result
			data, err := json.MarshalIndent(segments, "", "  ")
			if err != nil {
				log.Printf("Warning: Failed to serialize intermediate result: %v\n", err)
			} else {
				if err := os.WriteFile(intermediateFile, data, 0644); err != nil {
					log.Printf("Warning: Failed to save intermediate result: %v\n", err)
				}
			}

			return segments, nil
		}()
		allSegments = append(allSegments, segments...)

		// Increase the offset by the duration of the k-th chunk
		offset += dur
	}

	// Serialize all segments into a single JSON
	data, err := json.MarshalIndent(allSegments, "", "  ")
	if err != nil {
		log.Printf("Error serializing JSON: %v\n", err)
		return
	}
	if err := os.WriteFile(filepath.Join(outputDir, `transcription.json`), data, 0644); err != nil {
		log.Printf("Error saving file transcription.json: %v\n", err)
		return
	}
	fmt.Println("Combined JSON saved to file `transcription.json`")

	// Additionally, collect the overall text
	var textResult strings.Builder
	for _, seg := range allSegments {
		textResult.WriteString(seg.Text)
		textResult.WriteString("\n")
	}
	err = os.WriteFile(filepath.Join(outputDir, "transcription.txt"), []byte(textResult.String()), 0644)
	if err != nil {
		log.Printf("Error saving file transcription.txt: %v\n", err)
		return
	}
	fmt.Println("Overall text saved to file `transcription.txt`")

	// Create a text file with timestamps
	var result strings.Builder
	for _, segment := range allSegments {
		// Convert seconds to HH:MM:SS format
		startTime := formatTimestamp(segment.Start)
		endTime := formatTimestamp(segment.End)

		// Add timestamp and segment text
		result.WriteString(fmt.Sprintf("[%s - %s] %s\n", startTime, endTime, segment.Text))
	}

	// writing the result to a file
	outputPath := filepath.Join(outputDir, "transcription_with_timestamps.txt")
	if err := os.WriteFile(outputPath, []byte(result.String()), 0644); err != nil {
		log.Printf("Ошибка при записи файла: %v\n", err)
		return
	}
}

func formatTimestamp(seconds float64) string {
	duration := time.Duration(seconds * float64(time.Second))
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	secs := int(duration.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

func loadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	if config.Language == "" {
		config.Language = "ru"
	}

	return &config, nil
}
