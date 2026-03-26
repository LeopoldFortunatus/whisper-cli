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

type prepareInputCall struct {
	inputFile string
	workDir   string
}

type prepareChunksCall struct {
	inputFile    string
	workDir      string
	chunkSeconds int
}

type fakeAudioPipeline struct {
	mediaFiles         []string
	preparedInput      audio.PreparedInput
	chunks             []audio.Chunk
	ensureErr          error
	collectErr         error
	prepareInputErr    error
	prepareChunksErr   error
	collectCalls       []string
	prepareInputCalls  []prepareInputCall
	prepareChunksCalls []prepareChunksCall
	callOrder          []string
}

func (f *fakeAudioPipeline) EnsureBinaries() error {
	f.callOrder = append(f.callOrder, "ensure")
	return f.ensureErr
}

func (f *fakeAudioPipeline) CollectMediaFiles(dir string) ([]string, error) {
	f.callOrder = append(f.callOrder, "collect")
	f.collectCalls = append(f.collectCalls, dir)
	if f.collectErr != nil {
		return nil, f.collectErr
	}
	return f.mediaFiles, nil
}

func (f *fakeAudioPipeline) PrepareInput(_ context.Context, inputFile string, workDir string) (audio.PreparedInput, error) {
	f.callOrder = append(f.callOrder, "prepare_input")
	f.prepareInputCalls = append(f.prepareInputCalls, prepareInputCall{
		inputFile: inputFile,
		workDir:   workDir,
	})
	if f.prepareInputErr != nil {
		return audio.PreparedInput{}, f.prepareInputErr
	}
	if f.preparedInput.OriginalPath == "" && f.preparedInput.ChunkSourcePath == "" && !f.preparedInput.Converted {
		return audio.PreparedInput{
			OriginalPath:    inputFile,
			ChunkSourcePath: inputFile,
		}, nil
	}
	return f.preparedInput, nil
}

func (f *fakeAudioPipeline) PrepareChunks(_ context.Context, inputFile string, workDir string, chunkSeconds int) ([]audio.Chunk, error) {
	f.callOrder = append(f.callOrder, "prepare_chunks")
	f.prepareChunksCalls = append(f.prepareChunksCalls, prepareChunksCall{
		inputFile:    inputFile,
		workDir:      workDir,
		chunkSeconds: chunkSeconds,
	})
	if f.prepareChunksErr != nil {
		return nil, f.prepareChunksErr
	}
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
	input := filepath.Join(dir, "input.mp3")
	if err := os.WriteFile(input, []byte("x"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}
	outputRoot := filepath.Join(dir, "out")
	workDir := filepath.Join(outputRoot, "input", "_work")

	audioPipeline := &fakeAudioPipeline{
		preparedInput: audio.PreparedInput{
			OriginalPath:    input,
			ChunkSourcePath: filepath.Join(workDir, "source.m4a"),
			Converted:       true,
		},
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

	if strings.Join(audioPipeline.callOrder, ",") != "ensure,prepare_input,prepare_chunks" {
		t.Fatalf("unexpected call order: %v", audioPipeline.callOrder)
	}
	if len(audioPipeline.prepareInputCalls) != 1 {
		t.Fatalf("prepare input calls = %d, want 1", len(audioPipeline.prepareInputCalls))
	}
	if audioPipeline.prepareInputCalls[0].inputFile != input {
		t.Fatalf("PrepareInput input = %s, want %s", audioPipeline.prepareInputCalls[0].inputFile, input)
	}
	if audioPipeline.prepareInputCalls[0].workDir != workDir {
		t.Fatalf("PrepareInput workDir = %s, want %s", audioPipeline.prepareInputCalls[0].workDir, workDir)
	}
	if len(audioPipeline.prepareChunksCalls) != 1 {
		t.Fatalf("prepare chunks calls = %d, want 1", len(audioPipeline.prepareChunksCalls))
	}
	if audioPipeline.prepareChunksCalls[0].inputFile != filepath.Join(workDir, "source.m4a") {
		t.Fatalf("PrepareChunks input = %s", audioPipeline.prepareChunksCalls[0].inputFile)
	}
	if audioPipeline.prepareChunksCalls[0].workDir != workDir {
		t.Fatalf("PrepareChunks workDir = %s, want %s", audioPipeline.prepareChunksCalls[0].workDir, workDir)
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

	transcriptJSON, err := os.ReadFile(filepath.Join(outDir, "transcript.json"))
	if err != nil {
		t.Fatalf("read transcript.json: %v", err)
	}
	var transcript domain.Transcript
	if err := json.Unmarshal(transcriptJSON, &transcript); err != nil {
		t.Fatalf("unmarshal transcript.json: %v", err)
	}
	if transcript.Source != input {
		t.Fatalf("transcript source = %s, want %s", transcript.Source, input)
	}
	if _, err := os.Stat(filepath.Join(outDir, "_work", "transcript.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no transcript artifacts in _work, stat err = %v", err)
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

	audioPipeline := &fakeAudioPipeline{
		chunks: []audio.Chunk{
			{Number: 0, Path: "chunk-0", Offset: 0},
		},
	}
	app := &Application{
		FS:    fsx.OS{},
		Audio: audioPipeline,
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
		Audio: &fakeAudioPipeline{},
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

func TestApplicationRunRejectsEmptyMediaDirectory(t *testing.T) {
	t.Parallel()

	logger := zerolog.New(io.Discard)
	dir := t.TempDir()
	inputDir := filepath.Join(dir, "input-dir")
	if err := os.Mkdir(inputDir, 0o755); err != nil {
		t.Fatalf("mkdir input dir: %v", err)
	}

	audioPipeline := &fakeAudioPipeline{}
	app := &Application{
		FS:    fsx.OS{},
		Audio: audioPipeline,
		Registry: provider.NewRegistry(fakeProvider{
			name: domain.ProviderOpenAI,
			capabilities: map[string]domain.Capabilities{
				"whisper-1": {
					SupportsPrompt:            true,
					SupportsSegmentTimestamps: true,
					SupportsSRT:               true,
					SupportsVTT:               true,
				},
			},
		}),
		Logger: logger,
		Env:    config.OSEnv{},
	}

	err := app.Run(context.Background(), []string{
		"-input", inputDir,
		"-provider", "openai",
		"-model", "whisper-1",
	})
	if err == nil {
		t.Fatalf("expected error for empty input directory")
	}
	if !strings.Contains(err.Error(), "supported media files") {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(audioPipeline.collectCalls) != 1 || audioPipeline.collectCalls[0] != inputDir {
		t.Fatalf("collect calls = %v, want [%s]", audioPipeline.collectCalls, inputDir)
	}
}
