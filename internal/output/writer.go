package output

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/arykalin/whisper-cli/internal/domain"
	"github.com/arykalin/whisper-cli/internal/platform/fsx"
	"github.com/arykalin/whisper-cli/internal/provider"
)

type Layout struct {
	CacheDir  string
	PublicDir string
	Stem      string
}

type WriteRequest struct {
	Layout       Layout
	Transcript   domain.Transcript
	Outputs      domain.ArtifactSet
	RawArtifacts [][]byte
}

func PublicArtifactPaths(layout Layout, outputs domain.ArtifactSet) []string {
	paths := []string{filepath.Join(layout.PublicDir, layout.Stem+".txt")}
	if outputs.Enabled(domain.ArtifactTimestamps) {
		paths = append(paths, filepath.Join(layout.PublicDir, layout.Stem+".timestamps.txt"))
	}
	if outputs.Enabled(domain.ArtifactSRT) {
		paths = append(paths, filepath.Join(layout.PublicDir, layout.Stem+".srt"))
	}
	if outputs.Enabled(domain.ArtifactVTT) {
		paths = append(paths, filepath.Join(layout.PublicDir, layout.Stem+".vtt"))
	}
	if outputs.Enabled(domain.ArtifactDiarized) {
		paths = append(paths, filepath.Join(layout.PublicDir, layout.Stem+".diarized.json"))
	}
	if outputs.Enabled(domain.ArtifactRaw) {
		paths = append(paths, filepath.Join(layout.PublicDir, layout.Stem+".raw.json"))
	}
	return paths
}

func WriteArtifacts(fs fsx.FS, request WriteRequest) error {
	if err := fs.MkdirAll(request.Layout.CacheDir, 0o700); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	if err := fs.MkdirAll(request.Layout.PublicDir, 0o755); err != nil {
		return fmt.Errorf("create public output dir: %w", err)
	}

	plainText := request.Transcript.PlainText()
	if plainText == "" {
		return fmt.Errorf("transcript is empty")
	}
	request.Transcript.Text = plainText

	data, err := json.MarshalIndent(request.Transcript, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal transcript.json: %w", err)
	}
	cacheTranscriptPath := filepath.Join(request.Layout.CacheDir, "transcript.json")
	if err := fs.WriteFile(cacheTranscriptPath, data, 0o600); err != nil {
		return fmt.Errorf("write transcript.json: %w", err)
	}

	publicTranscriptPath := filepath.Join(request.Layout.PublicDir, request.Layout.Stem+".txt")
	if err := fs.WriteFile(publicTranscriptPath, []byte(plainText+"\n"), 0o644); err != nil {
		return fmt.Errorf("write public transcript: %w", err)
	}

	if request.Outputs.Enabled(domain.ArtifactTimestamps) {
		content, err := timestampsText(request.Transcript.Segments)
		if err != nil {
			return err
		}
		path := filepath.Join(request.Layout.PublicDir, request.Layout.Stem+".timestamps.txt")
		if err := fs.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write timestamps.txt: %w", err)
		}
	}

	if request.Outputs.Enabled(domain.ArtifactSRT) {
		content, err := srtText(request.Transcript.Segments)
		if err != nil {
			return err
		}
		path := filepath.Join(request.Layout.PublicDir, request.Layout.Stem+".srt")
		if err := fs.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write transcript.srt: %w", err)
		}
	}

	if request.Outputs.Enabled(domain.ArtifactVTT) {
		content, err := vttText(request.Transcript.Segments)
		if err != nil {
			return err
		}
		path := filepath.Join(request.Layout.PublicDir, request.Layout.Stem+".vtt")
		if err := fs.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write transcript.vtt: %w", err)
		}
	}

	if request.Outputs.Enabled(domain.ArtifactDiarized) {
		if len(request.Transcript.SpeakerSegments) == 0 {
			return fmt.Errorf("diarized output requested but speaker segments are empty")
		}
		payload := map[string]any{
			"provider":         request.Transcript.Provider,
			"model":            request.Transcript.Model,
			"language":         request.Transcript.Language,
			"text":             request.Transcript.Text,
			"speaker_segments": request.Transcript.SpeakerSegments,
		}
		content, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal diarized.json: %w", err)
		}
		path := filepath.Join(request.Layout.PublicDir, request.Layout.Stem+".diarized.json")
		if err := fs.WriteFile(path, content, 0o644); err != nil {
			return fmt.Errorf("write diarized.json: %w", err)
		}
	}

	if request.Outputs.Enabled(domain.ArtifactRaw) {
		content, err := provider.MarshalRawArray(request.RawArtifacts)
		if err != nil {
			return err
		}
		if len(content) > 0 {
			path := filepath.Join(request.Layout.PublicDir, request.Layout.Stem+".raw.json")
			if err := fs.WriteFile(path, content, 0o644); err != nil {
				return fmt.Errorf("write raw.json: %w", err)
			}
		}
	}

	return nil
}

func timestampsText(segments []domain.Segment) (string, error) {
	if len(segments) == 0 {
		return "", fmt.Errorf("timestamps output requested but segment timestamps are unavailable")
	}

	var builder strings.Builder
	for _, segment := range segments {
		builder.WriteString("[")
		builder.WriteString(hhmmss(segment.Start))
		builder.WriteString(" - ")
		builder.WriteString(hhmmss(segment.End))
		builder.WriteString("] ")
		builder.WriteString(strings.TrimSpace(segment.Text))
		builder.WriteString("\n")
	}
	return builder.String(), nil
}

func srtText(segments []domain.Segment) (string, error) {
	if len(segments) == 0 {
		return "", fmt.Errorf("srt output requested but segment timestamps are unavailable")
	}

	var builder strings.Builder
	for index, segment := range segments {
		builder.WriteString(fmt.Sprintf("%d\n", index+1))
		builder.WriteString(srtTimestamp(segment.Start))
		builder.WriteString(" --> ")
		builder.WriteString(srtTimestamp(segment.End))
		builder.WriteString("\n")
		builder.WriteString(strings.TrimSpace(segment.Text))
		builder.WriteString("\n\n")
	}
	return builder.String(), nil
}

func vttText(segments []domain.Segment) (string, error) {
	if len(segments) == 0 {
		return "", fmt.Errorf("vtt output requested but segment timestamps are unavailable")
	}

	var builder strings.Builder
	builder.WriteString("WEBVTT\n\n")
	for _, segment := range segments {
		builder.WriteString(vttTimestamp(segment.Start))
		builder.WriteString(" --> ")
		builder.WriteString(vttTimestamp(segment.End))
		builder.WriteString("\n")
		builder.WriteString(strings.TrimSpace(segment.Text))
		builder.WriteString("\n\n")
	}
	return builder.String(), nil
}

func hhmmss(seconds float64) string {
	duration := time.Duration(seconds * float64(time.Second))
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	secs := int(duration.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

func srtTimestamp(seconds float64) string {
	duration := time.Duration(seconds * float64(time.Second))
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	secs := int(duration.Seconds()) % 60
	millis := int(duration.Milliseconds()) % 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, secs, millis)
}

func vttTimestamp(seconds float64) string {
	duration := time.Duration(seconds * float64(time.Second))
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	secs := int(duration.Seconds()) % 60
	millis := int(duration.Milliseconds()) % 1000
	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, secs, millis)
}
