package audio

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/arykalin/whisper-cli/internal/platform/execx"
	"github.com/arykalin/whisper-cli/internal/platform/fsx"
)

type Chunk struct {
	Number   int
	Path     string
	Offset   float64
	Duration float64
}

type Pipeline interface {
	EnsureBinaries() error
	CollectAudioFiles(dir string) ([]string, error)
	PrepareChunks(ctx context.Context, inputFile string, workDir string, chunkSeconds int) ([]Chunk, error)
}

type Service struct {
	FS     fsx.FS
	Runner execx.Runner
}

var supportedAudioExt = map[string]struct{}{
	".flac": {},
	".m4a":  {},
	".mp3":  {},
	".mp4":  {},
	".mpeg": {},
	".mpga": {},
	".ogg":  {},
	".wav":  {},
	".webm": {},
}

func (s Service) EnsureBinaries() error {
	for _, name := range []string{"ffmpeg", "ffprobe"} {
		if _, err := s.Runner.LookPath(name); err != nil {
			return fmt.Errorf("%s not found in PATH", name)
		}
	}
	return nil
}

func (s Service) CollectAudioFiles(dir string) ([]string, error) {
	entries, err := s.FS.ReadDir(dir)
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

func (s Service) PrepareChunks(ctx context.Context, inputFile string, workDir string, chunkSeconds int) ([]Chunk, error) {
	if chunkSeconds <= 0 {
		return nil, fmt.Errorf("chunk_seconds must be greater than zero")
	}

	outputPattern := filepath.Join(workDir, "chunk_%03d.m4a")
	if err := s.FS.MkdirAll(workDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output directory: %w", err)
	}

	if err := s.split(ctx, inputFile, outputPattern, chunkSeconds); err != nil {
		return nil, err
	}

	return s.collectChunks(ctx, outputPattern)
}

func isSupportedAudio(name string) bool {
	_, ok := supportedAudioExt[strings.ToLower(filepath.Ext(name))]
	return ok
}

func (s Service) split(ctx context.Context, inputPath string, outputPattern string, chunkSeconds int) error {
	_, stderr, err := s.Runner.Run(
		ctx,
		"ffmpeg",
		"-i", inputPath,
		"-f", "segment",
		"-segment_time", strconv.Itoa(chunkSeconds),
		"-c", "copy",
		outputPattern,
	)
	if err != nil {
		return fmt.Errorf("split audio: %w: %s", err, strings.TrimSpace(string(stderr)))
	}
	return nil
}

func (s Service) collectChunks(ctx context.Context, outputPattern string) ([]Chunk, error) {
	var (
		chunks []Chunk
		offset float64
	)

	for index := 0; ; index++ {
		chunkFile := fmt.Sprintf(outputPattern, index)
		if _, err := s.FS.Stat(chunkFile); err != nil {
			if index == 0 {
				return nil, fmt.Errorf("no chunks generated for %s", outputPattern)
			}
			break
		}

		duration, err := s.duration(ctx, chunkFile)
		if err != nil {
			return nil, fmt.Errorf("duration for %s: %w", chunkFile, err)
		}

		chunks = append(chunks, Chunk{
			Number:   index,
			Path:     chunkFile,
			Offset:   offset,
			Duration: duration,
		})
		offset += duration
	}

	return chunks, nil
}

func (s Service) duration(ctx context.Context, filePath string) (float64, error) {
	stdout, stderr, err := s.Runner.Run(
		ctx,
		"ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "csv=p=0",
		filePath,
	)
	if err != nil {
		return 0, fmt.Errorf("ffprobe: %w: %s", err, strings.TrimSpace(string(stderr)))
	}

	value := strings.TrimSpace(string(stdout))
	duration, parseErr := strconv.ParseFloat(value, 64)
	if parseErr != nil {
		return 0, fmt.Errorf("parse duration %q: %w", value, parseErr)
	}
	return duration, nil
}
