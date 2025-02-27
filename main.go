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
	"strconv"
	"strings"

	"github.com/openai/openai-go"
	"github.com/rs/zerolog/log"
)

type Segment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

// Get the duration of the file using ffprobe
func getDuration(filePath string) (float64, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "csv=p=0",
		filePath,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0, err
	}
	str := strings.TrimSpace(out.String())
	return strconv.ParseFloat(str, 64)
}

// Split the audio file into parts using ffmpeg
func splitAudioFile(inputPath string, outputPattern string) error {
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-f", "segment",
		"-segment_time", "600", "-c", "copy", outputPattern)
	return cmd.Run()
}

// Send the file to Whisper and return JSON with segments
func transcribeAudio(ctx context.Context, client *openai.Client, filePath string, offset float64, language string) ([]Segment, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer f.Close()

	resp, err := client.Audio.Transcriptions.New(ctx, openai.AudioTranscriptionNewParams{
		File:           openai.F[io.Reader](f),
		Model:          openai.F(openai.AudioModelWhisper1),
		Language:       openai.F(language),
		ResponseFormat: openai.F(openai.AudioResponseFormatVerboseJSON),
		TimestampGranularities: openai.F([]openai.AudioTranscriptionNewParamsTimestampGranularity{
			openai.AudioTranscriptionNewParamsTimestampGranularitySegment,
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}

	// Parse the received JSON, it should contain a structure with an array of segments
	var result struct {
		Segments []Segment `json:"segments"`
	}
	err = json.Unmarshal([]byte(resp.Text), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	// Shift the timestamps
	for i := range result.Segments {
		result.Segments[i].Start += offset
		result.Segments[i].End += offset
	}
	return result.Segments, nil
}

func main() {
	inputFile := flag.String("input", "./whisper/audio/2025_02_15_диспут_Чегон_нельзя_делать_с_Мифом.m4a", "Path to the input audio file")
	outputPattern := flag.String("output", "./whisper/audio/chunk/chunk_%03d.m4a", "Pattern for the output audio chunks")
	language := flag.String("language", "ru", "Language for transcription")
	flag.Parse()

	fmt.Println("Splitting the audio file into parts...")
	err := splitAudioFile(*inputFile, *outputPattern)
	if err != nil {
		fmt.Printf("Error splitting file: %v\n", err)
		return
	}

	// Initialize the client based on the environment variable
	client := openai.NewClient()

	var allSegments []Segment
	offset := 0.0

	for i := 0; ; i++ {
		chunkFile := fmt.Sprintf(*outputPattern, i)
		if _, err := os.Stat(chunkFile); os.IsNotExist(err) {
			break
		}

		// Get the duration of the current chunk
		dur, err := getDuration(chunkFile)
		if err != nil {
			fmt.Printf("Failed to get duration of file %s: %v\n", chunkFile, err)
			break
		}

		ctx := context.Background()
		segments, err := transcribeAudio(ctx, client, chunkFile, offset, *language)
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
	if err := ioutil.WriteFile(`combined.json`, data, 0644); err != nil {
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
	err = ioutil.WriteFile(`transcription.txt`, []byte(textResult.String()), 0644)
	if err != nil {
		fmt.Printf("Error saving file transcription.txt: %v\n", err)
		return
	}
	fmt.Println("Overall text saved to file `transcription.txt`")
}
