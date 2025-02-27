package whisper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
func splitAudioFile(inputPath string, outputPattern string, outputDir string) error {
	// Create the output directory if it does not exist
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return fmt.Errorf("error creating directory: %v", err)
	}

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
