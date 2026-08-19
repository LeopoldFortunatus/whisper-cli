package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
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
	prepareChunksFunc  func(inputFile string, workDir string) ([]audio.Chunk, error)
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
	if f.prepareChunksFunc != nil {
		return f.prepareChunksFunc(inputFile, workDir)
	}
	return f.chunks, nil
}

type fakeProvider struct {
	name           domain.Provider
	capabilities   map[string]domain.Capabilities
	responses      map[string]provider.Response
	transcribeFunc func(ctx context.Context, req provider.Request) (provider.Response, error)
	preflightErr   error
	onPreflight    func()
}

func (f fakeProvider) Name() domain.Provider {
	return f.name
}

func (f fakeProvider) Preflight() error {
	if f.onPreflight != nil {
		f.onPreflight()
	}
	return f.preflightErr
}

func (f fakeProvider) Capabilities(model string) (domain.Capabilities, bool) {
	caps, ok := f.capabilities[model]
	return caps, ok
}

func (f fakeProvider) SupportedModels() []string {
	models := make([]string, 0, len(f.capabilities))
	for model := range f.capabilities {
		models = append(models, model)
	}
	sort.Strings(models)
	return models
}

func (f fakeProvider) Transcribe(ctx context.Context, req provider.Request) (provider.Response, error) {
	if f.transcribeFunc != nil {
		return f.transcribeFunc(ctx, req)
	}
	response, ok := f.responses[req.FilePath]
	if !ok {
		return provider.Response{}, errors.New("missing fake response")
	}
	return response, nil
}

func TestApplicationRunWritesDefaultOutputBesideInputAndResetsCache(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mediaDir := filepath.Join(dir, "media")
	if err := os.Mkdir(mediaDir, 0o755); err != nil {
		t.Fatalf("mkdir media dir: %v", err)
	}
	input := filepath.Join(mediaDir, "lecture.m4a")
	if err := os.WriteFile(input, []byte("audio"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}
	publicTranscript := filepath.Join(mediaDir, "lecture.txt")
	if err := os.WriteFile(publicTranscript, []byte("old transcript\n"), 0o644); err != nil {
		t.Fatalf("write old transcript: %v", err)
	}

	cacheRoot := filepath.Join(dir, "cache")
	cacheDir := filepath.Join(cacheRoot, cacheKey(input))
	workDir := filepath.Join(cacheDir, "work")
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		t.Fatalf("mkdir stale work dir: %v", err)
	}
	staleChunk := filepath.Join(workDir, "chunk_999.m4a")
	if err := os.WriteFile(staleChunk, []byte("stale"), 0o600); err != nil {
		t.Fatalf("write stale chunk: %v", err)
	}

	audioPipeline := &fakeAudioPipeline{
		chunks: []audio.Chunk{{Number: 0, Path: "chunk-0"}},
	}
	client := fakeProvider{
		name: domain.ProviderOpenAI,
		capabilities: map[string]domain.Capabilities{
			"gpt-4o-transcribe": {},
		},
		responses: map[string]provider.Response{
			"chunk-0": {Transcript: domain.Transcript{Text: "new transcript"}},
		},
	}
	application := &Application{
		FS:        fsx.OS{},
		Audio:     audioPipeline,
		Registry:  provider.NewRegistry(client),
		Logger:    zerolog.New(io.Discard),
		Env:       staticEnv{},
		CacheRoot: cacheRoot,
	}

	err := application.Run(context.Background(), config.Config{
		Inputs:       []string{input},
		Provider:     domain.ProviderOpenAI,
		Model:        "gpt-4o-transcribe",
		Outputs:      domain.DefaultArtifacts(),
		ChunkSeconds: 600,
		Concurrency:  1,
		Language:     "ru",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	text, err := os.ReadFile(publicTranscript)
	if err != nil {
		t.Fatalf("read public transcript: %v", err)
	}
	if string(text) != "new transcript\n" {
		t.Fatalf("public transcript = %q", string(text))
	}
	if _, err := os.Stat(filepath.Join(mediaDir, "transcript.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no public transcript.json, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "transcript.json")); err != nil {
		t.Fatalf("expected cached transcript.json: %v", err)
	}
	if _, err := os.Stat(staleChunk); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected stale chunk to be removed, stat err = %v", err)
	}
}

func TestApplicationRunProcessesInputsSequentiallyAndContinuesAfterError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	firstInput := filepath.Join(dir, "first.m4a")
	secondInput := filepath.Join(dir, "second.m4a")
	for _, input := range []string{firstInput, secondInput} {
		if err := os.WriteFile(input, []byte("audio"), 0o644); err != nil {
			t.Fatalf("write input %s: %v", input, err)
		}
	}

	audioPipeline := &fakeAudioPipeline{
		prepareChunksFunc: func(inputFile string, _ string) ([]audio.Chunk, error) {
			return []audio.Chunk{{Number: 0, Path: inputFile}}, nil
		},
	}
	transcribed := []string{}
	client := fakeProvider{
		name: domain.ProviderOpenAI,
		capabilities: map[string]domain.Capabilities{
			"gpt-4o-transcribe": {},
		},
		transcribeFunc: func(_ context.Context, req provider.Request) (provider.Response, error) {
			transcribed = append(transcribed, req.FilePath)
			if req.FilePath == firstInput {
				return provider.Response{}, errors.New("first input failed")
			}
			return provider.Response{Transcript: domain.Transcript{Text: "second transcript"}}, nil
		},
	}
	application := &Application{
		FS:        fsx.OS{},
		Audio:     audioPipeline,
		Registry:  provider.NewRegistry(client),
		Logger:    zerolog.New(io.Discard),
		Env:       staticEnv{},
		CacheRoot: filepath.Join(dir, "cache"),
	}

	err := application.Run(context.Background(), config.Config{
		Inputs:       []string{firstInput, secondInput},
		Provider:     domain.ProviderOpenAI,
		Model:        "gpt-4o-transcribe",
		Outputs:      domain.DefaultArtifacts(),
		ChunkSeconds: 600,
		Concurrency:  1,
		Language:     "ru",
	})
	if err == nil || !strings.Contains(err.Error(), "first input failed") {
		t.Fatalf("Run error = %v, want first input failure", err)
	}
	if strings.Join(transcribed, ",") != strings.Join([]string{firstInput, secondInput}, ",") {
		t.Fatalf("transcribed = %v, want ordered inputs", transcribed)
	}
	if _, err := os.Stat(filepath.Join(dir, "first.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no transcript for failed input, stat err = %v", err)
	}
	text, err := os.ReadFile(filepath.Join(dir, "second.txt"))
	if err != nil {
		t.Fatalf("read second transcript: %v", err)
	}
	if string(text) != "second transcript\n" {
		t.Fatalf("second transcript = %q", string(text))
	}
}

func TestApplicationRunRejectsPublicOutputCollisionBeforeTranscription(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	firstInput := filepath.Join(dir, "first", "lecture.m4a")
	secondInput := filepath.Join(dir, "second", "lecture.m4a")
	for _, input := range []string{firstInput, secondInput} {
		if err := os.MkdirAll(filepath.Dir(input), 0o755); err != nil {
			t.Fatalf("mkdir input dir: %v", err)
		}
		if err := os.WriteFile(input, []byte("audio"), 0o644); err != nil {
			t.Fatalf("write input: %v", err)
		}
	}

	transcribeCalls := 0
	audioPipeline := &fakeAudioPipeline{
		chunks: []audio.Chunk{{Number: 0, Path: "chunk-0"}},
	}
	client := fakeProvider{
		name: domain.ProviderOpenAI,
		capabilities: map[string]domain.Capabilities{
			"gpt-4o-transcribe": {},
		},
		transcribeFunc: func(context.Context, provider.Request) (provider.Response, error) {
			transcribeCalls++
			return provider.Response{Transcript: domain.Transcript{Text: "unexpected"}}, nil
		},
	}
	application := &Application{
		FS:        fsx.OS{},
		Audio:     audioPipeline,
		Registry:  provider.NewRegistry(client),
		Logger:    zerolog.New(io.Discard),
		Env:       staticEnv{},
		CacheRoot: filepath.Join(dir, "cache"),
	}

	err := application.Run(context.Background(), config.Config{
		Inputs:       []string{firstInput, secondInput},
		OutputDir:    filepath.Join(dir, "out"),
		Provider:     domain.ProviderOpenAI,
		Model:        "gpt-4o-transcribe",
		Outputs:      domain.DefaultArtifacts(),
		ChunkSeconds: 600,
		Concurrency:  1,
		Language:     "ru",
	})
	if err == nil || !strings.Contains(err.Error(), "public output path collision") {
		t.Fatalf("Run error = %v, want collision error", err)
	}
	if transcribeCalls != 0 {
		t.Fatalf("transcribe calls = %d, want 0", transcribeCalls)
	}
	if len(audioPipeline.prepareInputCalls) != 0 {
		t.Fatalf("prepare input calls = %v, want none", audioPipeline.prepareInputCalls)
	}
}

func TestApplicationPlanBatchDeduplicatesInputsAndSeparatesSameStems(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	firstInput := filepath.Join(dir, "first", "lecture.m4a")
	secondInput := filepath.Join(dir, "second", "lecture.m4a")
	for _, input := range []string{firstInput, secondInput} {
		if err := os.MkdirAll(filepath.Dir(input), 0o755); err != nil {
			t.Fatalf("mkdir input dir: %v", err)
		}
		if err := os.WriteFile(input, []byte("audio"), 0o644); err != nil {
			t.Fatalf("write input: %v", err)
		}
	}

	application := &Application{
		FS:        fsx.OS{},
		Audio:     &fakeAudioPipeline{},
		CacheRoot: filepath.Join(dir, "cache"),
	}
	items, err := application.planBatch(config.Config{
		Inputs:  []string{firstInput, firstInput, secondInput},
		Outputs: domain.DefaultArtifacts(),
	})
	if err != nil {
		t.Fatalf("planBatch returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %v, want two deduplicated inputs", items)
	}
	if items[0].inputPath != firstInput || items[1].inputPath != secondInput {
		t.Fatalf("input order = [%s %s]", items[0].inputPath, items[1].inputPath)
	}
	if items[0].publicDir == items[1].publicDir {
		t.Fatalf("public dirs must stay source-local: %s", items[0].publicDir)
	}
	if items[0].cacheDir == items[1].cacheDir {
		t.Fatalf("cache dirs must differ for distinct absolute paths: %s", items[0].cacheDir)
	}
}

type staticEnv map[string]string

func (s staticEnv) LookupEnv(key string) (string, bool) {
	value, ok := s[key]
	return value, ok
}

type fakeKeyResolver struct {
	key       string
	err       error
	onResolve func()
}

func (f fakeKeyResolver) Resolve(context.Context) (string, error) {
	if f.onResolve != nil {
		f.onResolve()
	}
	return f.key, f.err
}

func TestApplicationRunResolvesOpenAIKeyBeforeProviderPreflight(t *testing.T) {
	t.Parallel()

	var order []string
	app := &Application{
		FS:       fsx.OS{},
		Audio:    &fakeAudioPipeline{},
		Registry: provider.NewRegistry(fakeProvider{name: domain.ProviderOpenAI}),
		Logger:   zerolog.New(io.Discard),
		Env:      staticEnv{},
		OpenAIKeyResolver: fakeKeyResolver{
			key: "resolved-key",
			onResolve: func() {
				order = append(order, "resolve")
			},
		},
		OpenAIProviderFactory: func(apiKey string) provider.Client {
			if apiKey != "resolved-key" {
				t.Fatalf("apiKey = %q, want resolved-key", apiKey)
			}
			order = append(order, "factory")
			return fakeProvider{
				name:         domain.ProviderOpenAI,
				preflightErr: errors.New("stop after preflight"),
				onPreflight: func() {
					order = append(order, "preflight")
				},
			}
		},
	}

	err := app.Run(context.Background(), config.Config{
		Provider: domain.ProviderOpenAI,
		Model:    "whisper-1",
	})
	if err == nil {
		t.Fatal("expected preflight error")
	}
	if strings.Join(order, ",") != "resolve,factory,preflight" {
		t.Fatalf("order = %v, want resolve,factory,preflight", order)
	}
}

func TestApplicationRunKeepsGroqOnRegisteredProviderPath(t *testing.T) {
	t.Parallel()

	var resolverCalls int
	app := &Application{
		FS:    fsx.OS{},
		Audio: &fakeAudioPipeline{},
		Registry: provider.NewRegistry(fakeProvider{
			name:         domain.ProviderGroq,
			preflightErr: errors.New("groq preflight"),
		}),
		Logger: zerolog.New(io.Discard),
		Env:    staticEnv{},
		OpenAIKeyResolver: fakeKeyResolver{
			key: "openai-key",
			onResolve: func() {
				resolverCalls++
			},
		},
		OpenAIProviderFactory: func(apiKey string) provider.Client {
			t.Fatalf("unexpected OpenAI provider factory call with %q", apiKey)
			return nil
		},
	}

	err := app.Run(context.Background(), config.Config{
		Provider: domain.ProviderGroq,
		Model:    "whisper-large-v3-turbo",
	})
	if err == nil {
		t.Fatal("expected groq preflight error")
	}
	if resolverCalls != 0 {
		t.Fatalf("resolver calls = %d, want 0", resolverCalls)
	}
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
	cacheRoot := filepath.Join(dir, "cache")
	cacheDir := filepath.Join(cacheRoot, cacheKey(input))
	workDir := filepath.Join(cacheDir, "work")

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
		FS:        fsx.OS{},
		Audio:     audioPipeline,
		Registry:  provider.NewRegistry(client),
		Logger:    logger,
		Env:       staticEnv{},
		CacheRoot: cacheRoot,
	}

	err := app.Run(context.Background(), config.Config{
		Inputs:       []string{input},
		OutputDir:    outputRoot,
		Provider:     domain.ProviderOpenAI,
		Model:        "whisper-1",
		Outputs:      domain.ArtifactSet{domain.ArtifactTimestamps: true, domain.ArtifactRaw: true},
		ChunkSeconds: 600,
		Concurrency:  1,
		Language:     "ru",
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

	text, err := os.ReadFile(filepath.Join(outputRoot, "input.txt"))
	if err != nil {
		t.Fatalf("read transcript.txt: %v", err)
	}
	if strings.TrimSpace(string(text)) != "first\nsecond" {
		t.Fatalf("unexpected transcript.txt: %q", string(text))
	}

	timestamps, err := os.ReadFile(filepath.Join(outputRoot, "input.timestamps.txt"))
	if err != nil {
		t.Fatalf("read timestamps.txt: %v", err)
	}
	if !strings.Contains(string(timestamps), "[00:00:05 - 00:00:06] second") {
		t.Fatalf("timestamps did not contain shifted second segment: %s", string(timestamps))
	}

	raw, err := os.ReadFile(filepath.Join(outputRoot, "input.raw.json"))
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

	transcriptJSON, err := os.ReadFile(filepath.Join(cacheDir, "transcript.json"))
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
	if _, err := os.Stat(filepath.Join(outputRoot, "transcript.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no public transcript.json, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(workDir, "transcript.txt")); !errors.Is(err, os.ErrNotExist) {
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
		Logger:    logger,
		Env:       config.OSEnv{},
		CacheRoot: filepath.Join(dir, "cache"),
	}

	outputRoot := filepath.Join(dir, "out")
	err := app.Run(context.Background(), config.Config{
		Inputs:       []string{input},
		OutputDir:    outputRoot,
		Provider:     domain.ProviderOpenAI,
		Model:        "gpt-4o-transcribe",
		Outputs:      domain.ArtifactSet{domain.ArtifactTimestamps: true},
		ChunkSeconds: 600,
		Concurrency:  1,
		Language:     "ru",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outputRoot, "input.timestamps.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected timestamps.txt to be skipped, stat err = %v", err)
	}

	text, err := os.ReadFile(filepath.Join(outputRoot, "input.txt"))
	if err != nil {
		t.Fatalf("read transcript.txt: %v", err)
	}
	if strings.TrimSpace(string(text)) != "plain transcript" {
		t.Fatalf("unexpected transcript.txt: %q", string(text))
	}
}

func TestApplicationRunDoesNotPublishDiarizedArtifactByDefault(t *testing.T) {
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
				"gpt-4o-transcribe-diarize": {
					SupportsDiarization: true,
				},
			},
			responses: map[string]provider.Response{
				"chunk-0": {
					Transcript: domain.Transcript{
						Text: "speaker transcript",
						SpeakerSegments: []domain.SpeakerSegment{
							{Start: 0, End: 1, Speaker: "speaker_0", Text: "speaker transcript"},
						},
					},
				},
			},
		}),
		Logger:    logger,
		Env:       config.OSEnv{},
		CacheRoot: filepath.Join(dir, "cache"),
	}

	outputRoot := filepath.Join(dir, "out")
	err := app.Run(context.Background(), config.Config{
		Inputs:       []string{input},
		OutputDir:    outputRoot,
		Provider:     domain.ProviderOpenAI,
		Model:        "gpt-4o-transcribe-diarize",
		Outputs:      domain.DefaultArtifacts(),
		ChunkSeconds: 600,
		Concurrency:  1,
		Language:     "ru",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outputRoot, "input.timestamps.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected timestamps.txt to be skipped, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputRoot, "input.diarized.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no default diarized.json, stat err = %v", err)
	}
	text, err := os.ReadFile(filepath.Join(outputRoot, "input.txt"))
	if err != nil {
		t.Fatalf("read input.txt: %v", err)
	}
	if strings.TrimSpace(string(text)) != "speaker transcript" {
		t.Fatalf("unexpected input.txt: %q", string(text))
	}
}

func TestNormalizeConfigRespectsExplicitOutputsForDiarizeModel(t *testing.T) {
	t.Parallel()

	logger := zerolog.New(io.Discard)
	client := fakeProvider{
		name: domain.ProviderOpenAI,
		capabilities: map[string]domain.Capabilities{
			"gpt-4o-transcribe-diarize": {
				SupportsDiarization: true,
			},
		},
	}

	tests := []struct {
		name    string
		outputs string
		want    domain.ArtifactKind
	}{
		{
			name:    "env none",
			outputs: "none",
		},
		{
			name:    "flag raw",
			outputs: "raw",
			want:    domain.ArtifactRaw,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			overrides := config.Overrides{}
			overrides.Input.SetValue("input.m4a")
			if tt.name == "flag raw" {
				overrides.Outputs.SetValue(tt.outputs)
			}
			env := mapEnvForApp(tt.name, tt.outputs)
			cfg, err := config.Resolve(overrides, env)
			if err != nil {
				t.Fatalf("Resolve returned error: %v", err)
			}
			cfg.Model = "gpt-4o-transcribe-diarize"

			got, err := normalizeConfigAgainstCapabilities(client, cfg, logger)
			if err != nil {
				t.Fatalf("normalizeConfigAgainstCapabilities returned error: %v", err)
			}
			if got.Outputs.Enabled(domain.ArtifactDiarized) {
				t.Fatalf("outputs = %#v, did not want auto diarized for explicit outputs", got.Outputs)
			}
			if tt.want != "" && !got.Outputs.Enabled(tt.want) {
				t.Fatalf("outputs = %#v, want %s enabled", got.Outputs, tt.want)
			}
			if tt.want == "" && len(got.Outputs.Sorted()) != 0 {
				t.Fatalf("outputs = %#v, want none", got.Outputs)
			}
		})
	}
}

func mapEnvForApp(name string, outputs string) config.EnvSource {
	if name == "env none" {
		return staticEnv{
			"WHISPER_CLI_INPUT":   "input.m4a",
			"WHISPER_CLI_OUTPUTS": outputs,
		}
	}
	return staticEnv{}
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

	err := app.Run(context.Background(), config.Config{
		Inputs:       []string{input},
		OutputDir:    "output",
		Provider:     domain.ProviderOpenAI,
		Model:        "gpt-4o-transcribe",
		Outputs:      domain.ArtifactSet{domain.ArtifactSRT: true},
		ChunkSeconds: 600,
		Concurrency:  1,
		Language:     "ru",
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

	err := app.Run(context.Background(), config.Config{
		Inputs:       []string{inputDir},
		OutputDir:    "output",
		Provider:     domain.ProviderOpenAI,
		Model:        "whisper-1",
		Outputs:      domain.DefaultArtifacts(),
		ChunkSeconds: 600,
		Concurrency:  1,
		Language:     "ru",
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
