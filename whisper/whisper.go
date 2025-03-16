package whisper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/openai/openai-go"
)

// GetDuration Get the duration of the file using ffprobe
func GetDuration(filePath string) (float64, error) {
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

// SplitAudioFile Split the audio file into parts using ffmpeg
func SplitAudioFile(
	inputPath string,
	outputPattern string,
) error {
	// Create the output directory if it doesn't exist
	outputDir := filepath.Dir(outputPattern)
	err := os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error creating output directory: %v", err)
	}

	cmd := exec.Command("ffmpeg", "-i", inputPath, "-f", "segment",
		"-segment_time", "600", "-c", "copy", outputPattern)
	return cmd.Run()
}

// TranscribeAudio Send the file to Whisper and return JSON with segments
func TranscribeAudio(
	ctx context.Context,
	client *openai.Client,
	filePath string,
	offset float64,
	language string,
) ([]Segment, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Printf("error closing file: %v", err)
		}
	}(f)

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

	var result WhisperResponse
	err = json.Unmarshal([]byte(resp.JSON.RawJSON()), &result)
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
