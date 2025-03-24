package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/arykalin/whisper-cli/whisper"
	"github.com/openai/openai-go"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type Config struct {
	InputFile string `yaml:"input_file"`
	Language  string `yaml:"language,omitempty"`   // Optional language field
	OutPutDir string `yaml:"output_dir,omitempty"` // Optional output directory
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
	outputDir := fmt.Sprintf("%s/%s", config.OutPutDir, strings.TrimSuffix(inputFile, filepath.Ext(inputFile)))
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

	ctx := context.Background()
	client := openai.NewClient()
	allSegments := makeAllSegmentsParallel(
		ctx,
		outputPattern,
		config.Language,
		client,
	)

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
	outputFile := strings.TrimSuffix(inputFile, filepath.Ext(inputFile)) + ".txt"
	outputPath := filepath.Join(outputDir, outputFile)
	if err := os.WriteFile(outputPath, []byte(result.String()), 0644); err != nil {
		log.Printf("Ошибка при записи файла: %v\n", err)
		return
	}
}

func makeAllSegmentsParallel(
	ctx context.Context,
	outputPattern string,
	language string,
	client *openai.Client,
) []whisper.Segment {
	var allSegments []whisper.Segment
	var mu sync.Mutex
	var wg sync.WaitGroup

	// First, collect all files and their offsets
	type chunkInfo struct {
		path   string
		offset float64
	}

	var chunks []chunkInfo
	offset := 0.0

	for i := 0; ; i++ {
		chunkFile := fmt.Sprintf(outputPattern, i)
		if _, err := os.Stat(chunkFile); os.IsNotExist(err) {
			break
		}

		dur, err := whisper.GetDuration(chunkFile)
		if err != nil {
			log.Error().Err(err).Str("file", chunkFile).Msg("Failed to get duration")
			break
		}

		chunks = append(chunks, chunkInfo{
			path:   chunkFile,
			offset: offset,
		})

		offset += dur
	}

	// Now process them in parallel with a worker pool
	workerCount := runtime.NumCPU()
	if workerCount > len(chunks) {
		workerCount = len(chunks)
	}

	jobs := make(chan chunkInfo, len(chunks))

	// Start workers
	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for chunk := range jobs {
				segments, err := makeSegments(ctx, chunk.path, client, chunk.offset, language)
				if err != nil {
					log.Error().Err(err).Str("file", chunk.path).Msg("Error transcribing file")
					continue
				}

				mu.Lock()
				allSegments = append(allSegments, segments...)
				mu.Unlock()
			}
		}()
	}

	// Send jobs
	for _, chunk := range chunks {
		jobs <- chunk
	}
	close(jobs)

	// Wait for completion
	wg.Wait()

	// Sort segments by start time
	sort.Slice(allSegments, func(i, j int) bool {
		return allSegments[i].Start < allSegments[j].Start
	})

	return allSegments
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

	if config.OutPutDir == "" {
		config.OutPutDir = "output"
	}

	return &config, nil
}

func makeSegments(
	ctx context.Context,
	chunkFile string,
	client *openai.Client,
	offset float64,
	language string,
) ([]whisper.Segment, error) {
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
	segments, err := whisper.TranscribeAudio(ctx, client, chunkFile, offset, language)
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
}
