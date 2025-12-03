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

type chunkInfo struct {
	number int
	path   string
	offset float64
}

var supportedAudioExt = map[string]struct{}{
	".m4a": {},
}

type Config struct {
	InputFile string `yaml:"input_file"`
	Language  string `yaml:"language,omitempty"`   // Optional language field
	OutPutDir string `yaml:"output_dir,omitempty"` // Optional output directory
	UserGPT4  bool   `yaml:"usergpt4,omitempty"`   // Optional format field
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
	config.UserGPT4 = true

	// If the input file is not provided in the config, check the command line flag
	inputPath := strings.TrimSpace(config.InputFile)
	if *inputFlag != "" {
		inputPath = *inputFlag
	}

	if inputPath == "" {
		log.Fatal().Msg("No input file specified. Use -input flag or set input_file in config.yaml.")
	}

	inputPath = filepath.Clean(inputPath)
	inputPath, err = filepath.Abs(inputPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to resolve input path")
	}

	info, err := os.Stat(inputPath)
	if err != nil {
		log.Fatal().Err(err).Str("path", inputPath).Msg("Cannot stat input path")
	}

	ctx := context.Background()
	client := openai.NewClient()

	if info.IsDir() {
		files, err := collectAudioFiles(inputPath)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to scan input directory")
		}
		if len(files) == 0 {
			log.Fatal().Msg("Input directory does not contain .m4a audio files")
		}

		for _, file := range files {
			if err := processFile(ctx, client, config, file); err != nil {
				log.Error().Err(err).Str("file", file).Msg("Failed to transcribe file")
			}
		}
		return
	}

	if err := processFile(ctx, client, config, inputPath); err != nil {
		log.Fatal().Err(err).Msg("Failed to transcribe input file")
	}
}

func processFile(
	ctx context.Context,
	client openai.Client,
	config *Config,
	inputFile string,
) error {
	inputFile = filepath.Clean(inputFile)
	absInput, err := filepath.Abs(inputFile)
	if err != nil {
		return fmt.Errorf("failed to resolve path %s: %w", inputFile, err)
	}

	if !isSupportedAudio(absInput) {
		return fmt.Errorf("unsupported file format (only .m4a is allowed): %s", absInput)
	}

	// Extract just the base filename without extension
	baseName := filepath.Base(strings.TrimSuffix(absInput, filepath.Ext(absInput)))
	outputRoot, err := filepath.Abs(filepath.Clean(config.OutPutDir))
	if err != nil {
		return fmt.Errorf("failed to resolve output directory %s: %w", config.OutPutDir, err)
	}

	outputDir := filepath.Join(outputRoot, baseName)
	if err = os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return fmt.Errorf("error creating output directory %s: %w", outputDir, err)
	}

	// Update the outputPattern to save chunks in the created folder
	outputPattern := filepath.Join(outputDir, "chunk_%03d.m4a")

	fmt.Printf("Splitting the audio file into chunks for %s...\n", absInput)
	if err = whisper.SplitAudioFile(absInput, outputPattern); err != nil {
		return fmt.Errorf("error splitting file %s: %w", absInput, err)
	}

	allSegments := makeAllParallel(
		ctx,
		outputPattern,
		config.Language,
		client,
		config.UserGPT4,
	)

	// Serialize all segments into a single JSON
	data, err := json.MarshalIndent(allSegments, "", "  ")
	if err != nil {
		return fmt.Errorf("error serializing JSON: %w", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, `transcription.json`), data, 0644); err != nil {
		return fmt.Errorf("error saving file transcription.json: %w", err)
	}
	fmt.Println("Combined JSON saved to file `transcription.json`")

	// Additionally, collect the overall text
	var textResult strings.Builder
	for _, seg := range allSegments {
		textResult.WriteString(seg.Text)
		textResult.WriteString("\n")
	}
	if err = os.WriteFile(filepath.Join(outputDir, "transcription.txt"), []byte(textResult.String()), 0644); err != nil {
		return fmt.Errorf("error saving file transcription.txt: %w", err)
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
	outputFile := baseName + ".txt"
	outputPath := filepath.Join(outputDir, outputFile)
	if err := os.WriteFile(outputPath, []byte(result.String()), 0644); err != nil {
		return fmt.Errorf("error writing result file: %w", err)
	}

	return nil
}

func makeAllParallel(
	ctx context.Context,
	outputPattern string,
	language string,
	client openai.Client,
	useGPT4 bool,
) []whisper.Segment {
	var allSegments []whisper.Segment
	var mu sync.Mutex
	var wg sync.WaitGroup

	// First, collect all files and their offsets
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
			number: i,
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
				segments, err := makeSegments(ctx, chunk, client, language, useGPT4)
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

func collectAudioFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if isSupportedAudio(entry.Name()) {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	sort.Strings(files)
	return files, nil
}

func isSupportedAudio(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	_, ok := supportedAudioExt[ext]
	return ok
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
	chunk chunkInfo,
	client openai.Client,
	language string,
	useGpt4 bool,
) ([]whisper.Segment, error) {
	// Check if intermediate file exists
	intermediateFile := strings.TrimSuffix(chunk.path, filepath.Ext(chunk.path)) + "_transcription.json"
	if data, err := os.ReadFile(intermediateFile); err == nil {
		// File exists, try to unmarshal
		var segments []whisper.Segment
		if err := json.Unmarshal(data, &segments); err == nil {
			log.Printf("Loading existing transcription for %s\n", chunk.path)
			return segments, nil
		}
		// If unmarshalling fails, continue with transcription
	}

	// Transcribe and save result
	var (
		segments []whisper.Segment
		err      error
	)
	if useGpt4 {
		result, err := whisper.TranscribeAudioText(ctx, client, chunk.path, language)
		if err != nil {
			return nil, fmt.Errorf("error transcribing file: %v", err)
		}
		segments = []whisper.Segment{
			{
				Start: float64(chunk.number),
				Text:  result,
			},
		}
	} else {
		if segments, err = whisper.TranscribeAudioSRT(ctx, client, chunk.path, chunk.offset, language); err != nil {
			return nil, fmt.Errorf("error transcribing file: %v", err)
		}
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
