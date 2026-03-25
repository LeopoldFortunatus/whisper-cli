package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arykalin/whisper-cli/internal/audio"
	"github.com/arykalin/whisper-cli/internal/config"
	"github.com/arykalin/whisper-cli/internal/domain"
	"github.com/arykalin/whisper-cli/internal/platform/fsx"
	"github.com/arykalin/whisper-cli/internal/provider"
	"github.com/rs/zerolog"
)

type fakeAudioPipeline struct {
	chunks []audio.Chunk
}

func (f fakeAudioPipeline) EnsureBinaries() error {
	return nil
}

func (f fakeAudioPipeline) CollectAudioFiles(dir string) ([]string, error) {
	return nil, errors.New("not implemented")
}

func (f fakeAudioPipeline) PrepareChunks(context.Context, string, string, int) ([]audio.Chunk, error) {
	return f.chunks, nil
}

type fakeProvider struct {
	name         domain.Provider
	capabilities map[string]domain.Capabilities
	responses    map[string]provider.Response
}

func (f fakeProvider) Name() domain.Provider {
	return f.name
}

func (f fakeProvider) Preflight() error {
	return nil
}

func (f fakeProvider) Capabilities(model string) (domain.Capabilities, bool) {
	caps, ok := f.capabilities[model]
	return caps, ok
}

func (f fakeProvider) Transcribe(ctx context.Context, req provider.Request) (provider.Response, error) {
	response, ok := f.responses[req.FilePath]
	if !ok {
		return provider.Response{}, errors.New("missing fake response")
	}
	return response, nil
}

type staticEnv map[string]string

func (s staticEnv) LookupEnv(key string) (string, bool) {
	value, ok := s[key]
	return value, ok
}

func TestApplicationRunWritesMergedArtifactsInChunkOrder(t *testing.T) {
	t.Parallel()

	logger := zerolog.New(io.Discard)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.m4a")
	if err := os.WriteFile(input, []byte("x"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	audioPipeline := fakeAudioPipeline{
		chunks: []audio.Chunk{
			{Number: 1, Path: "chunk-1", Offset: 5},
			{Number: 0, Path: "chunk-0", Offset: 0},
		},
	}

	client := fakeProvider{
		name: domain.ProviderOpenAI,
		capabilities: map[string]domain.Capabilities{
			"whisper-1": {
				SupportsPrompt:            true,
				SupportsSegmentTimestamps: true,
				SupportsSRT:               true,
				SupportsVTT:               true,
			},
		},
		responses: map[string]provider.Response{
			"chunk-0": {
				Transcript: domain.Transcript{
					Text: "first",
					Segments: []domain.Segment{
						{Start: 0, End: 1, Text: "first"},
					},
				},
				Raw: []byte(`{"chunk":0}`),
			},
			"chunk-1": {
				Transcript: domain.Transcript{
					Text: "second",
					Segments: []domain.Segment{
						{Start: 0, End: 1, Text: "second"},
					},
				},
				Raw: []byte(`{"chunk":1}`),
			},
		},
	}

	app := &Application{
		FS:       fsx.OS{},
		Audio:    audioPipeline,
		Registry: provider.NewRegistry(client),
		Logger:   logger,
		Env:      staticEnv{},
	}

	outputRoot := filepath.Join(dir, "out")
	err := app.Run(context.Background(), []string{
		"-input", input,
		"-output-dir", outputRoot,
		"-provider", "openai",
		"-model", "whisper-1",
		"-outputs", "timestamps,raw",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	outDir := filepath.Join(outputRoot, "input")
	text, err := os.ReadFile(filepath.Join(outDir, "transcript.txt"))
	if err != nil {
		t.Fatalf("read transcript.txt: %v", err)
	}
	if strings.TrimSpace(string(text)) != "first\nsecond" {
		t.Fatalf("unexpected transcript.txt: %q", string(text))
	}

	timestamps, err := os.ReadFile(filepath.Join(outDir, "timestamps.txt"))
	if err != nil {
		t.Fatalf("read timestamps.txt: %v", err)
	}
	if !strings.Contains(string(timestamps), "[00:00:05 - 00:00:06] second") {
		t.Fatalf("timestamps did not contain shifted second segment: %s", string(timestamps))
	}

	raw, err := os.ReadFile(filepath.Join(outDir, "raw.json"))
	if err != nil {
		t.Fatalf("read raw.json: %v", err)
	}
	var rawItems []json.RawMessage
	if err := json.Unmarshal(raw, &rawItems); err != nil {
		t.Fatalf("unmarshal raw.json: %v", err)
	}
	if len(rawItems) != 2 {
		t.Fatalf("raw item count = %d", len(rawItems))
	}
}

func TestApplicationRunDisablesUnsupportedTimestampArtifacts(t *testing.T) {
	t.Parallel()

	logger := zerolog.New(io.Discard)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.m4a")
	if err := os.WriteFile(input, []byte("x"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	app := &Application{
		FS: fsx.OS{},
		Audio: fakeAudioPipeline{
			chunks: []audio.Chunk{
				{Number: 0, Path: "chunk-0", Offset: 0},
			},
		},
		Registry: provider.NewRegistry(fakeProvider{
			name: domain.ProviderOpenAI,
			capabilities: map[string]domain.Capabilities{
				"gpt-4o-transcribe": {},
			},
			responses: map[string]provider.Response{
				"chunk-0": {
					Transcript: domain.Transcript{
						Text: "plain transcript",
					},
				},
			},
		}),
		Logger: logger,
		Env:    config.OSEnv{},
	}

	outputRoot := filepath.Join(dir, "out")
	err := app.Run(context.Background(), []string{
		"-input", input,
		"-output-dir", outputRoot,
		"-provider", "openai",
		"-model", "gpt-4o-transcribe",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outDir := filepath.Join(outputRoot, "input")
	if _, err := os.Stat(filepath.Join(outDir, "timestamps.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected timestamps.txt to be skipped, stat err = %v", err)
	}

	text, err := os.ReadFile(filepath.Join(outDir, "transcript.txt"))
	if err != nil {
		t.Fatalf("read transcript.txt: %v", err)
	}
	if strings.TrimSpace(string(text)) != "plain transcript" {
		t.Fatalf("unexpected transcript.txt: %q", string(text))
	}
}

func TestApplicationRunRejectsUnsupportedSRTArtifacts(t *testing.T) {
	t.Parallel()

	logger := zerolog.New(io.Discard)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.m4a")
	if err := os.WriteFile(input, []byte("x"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	app := &Application{
		FS:    fsx.OS{},
		Audio: fakeAudioPipeline{},
		Registry: provider.NewRegistry(fakeProvider{
			name: domain.ProviderOpenAI,
			capabilities: map[string]domain.Capabilities{
				"gpt-4o-transcribe": {},
			},
		}),
		Logger: logger,
		Env:    config.OSEnv{},
	}

	err := app.Run(context.Background(), []string{
		"-input", input,
		"-provider", "openai",
		"-model", "gpt-4o-transcribe",
		"-outputs", "srt",
	})
	if err == nil {
		t.Fatalf("expected error for unsupported srt artifacts")
	}
	if !strings.Contains(err.Error(), "does not support srt artifacts") {
		t.Fatalf("unexpected error: %v", err)
	}
}
