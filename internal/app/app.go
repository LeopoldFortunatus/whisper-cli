package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/arykalin/whisper-cli/internal/audio"
	"github.com/arykalin/whisper-cli/internal/config"
	"github.com/arykalin/whisper-cli/internal/domain"
	"github.com/arykalin/whisper-cli/internal/output"
	"github.com/arykalin/whisper-cli/internal/platform/execx"
	"github.com/arykalin/whisper-cli/internal/platform/fsx"
	"github.com/arykalin/whisper-cli/internal/provider"
	"github.com/arykalin/whisper-cli/internal/provider/groqadapter"
	"github.com/arykalin/whisper-cli/internal/provider/openaiadapter"
	"github.com/rs/zerolog"
)

type Application struct {
	FS       fsx.FS
	Audio    audio.Pipeline
	Registry provider.Registry
	Logger   zerolog.Logger
	Env      config.EnvSource
}

func NewDefault() *Application {
	logger := initLogger()
	filesystem := fsx.OS{}
	audioService := audio.Service{
		FS:     filesystem,
		Runner: execx.OS{},
	}

	return &Application{
		FS:    filesystem,
		Audio: audioService,
		Registry: provider.NewRegistry(
			openaiadapter.New(strings.TrimSpace(os.Getenv("OPENAI_API_KEY")), filesystem, logger),
			groqadapter.New(strings.TrimSpace(os.Getenv("GROQ_API_KEY")), filesystem, logger),
			provider.NewBlockedClient(domain.ProviderOpenRouter, provider.ErrOpenRouterPlanned),
		),
		Logger: logger,
		Env:    config.OSEnv{},
	}
}

func (a *Application) Run(ctx context.Context, args []string) error {
	flags, err := config.ParseFlags(args)
	if err != nil {
		return err
	}

	cfg, warnings, err := config.Resolve(flags, a.Env, a.FS)
	if err != nil {
		return err
	}
	for _, warning := range warnings {
		a.Logger.Warn().Msg(warning)
	}

	if err := a.Audio.EnsureBinaries(); err != nil {
		return err
	}

	client, err := a.Registry.Provider(cfg.Provider)
	if err != nil {
		return err
	}
	if err := client.Preflight(); err != nil {
		return err
	}
	cfg, err = normalizeConfigAgainstCapabilities(client, cfg, a.Logger)
	if err != nil {
		return err
	}

	inputPath, err := a.FS.Abs(filepath.Clean(cfg.Input))
	if err != nil {
		return fmt.Errorf("resolve input path: %w", err)
	}
	outputRoot, err := a.FS.Abs(filepath.Clean(cfg.OutputDir))
	if err != nil {
		return fmt.Errorf("resolve output dir: %w", err)
	}

	info, err := a.FS.Stat(inputPath)
	if err != nil {
		return fmt.Errorf("stat input path %s: %w", inputPath, err)
	}

	if info.IsDir() {
		files, err := a.Audio.CollectMediaFiles(inputPath)
		if err != nil {
			return fmt.Errorf("scan input directory: %w", err)
		}
		if len(files) == 0 {
			return errors.New("input directory does not contain supported media files")
		}

		var firstErr error
		for _, file := range files {
			if err := a.processFile(ctx, client, cfg, file, outputRoot); err != nil {
				a.Logger.Error().Err(err).Str("file", file).Msg("failed to transcribe file")
				if firstErr == nil {
					firstErr = err
				}
			}
		}
		return firstErr
	}

	return a.processFile(ctx, client, cfg, inputPath, outputRoot)
}

func normalizeConfigAgainstCapabilities(client provider.Client, cfg config.Config, logger zerolog.Logger) (config.Config, error) {
	caps, ok := client.Capabilities(cfg.Model)
	if !ok {
		return config.Config{}, fmt.Errorf("model %s is not supported by provider %s", cfg.Model, cfg.Provider)
	}

	if cfg.Prompt != "" && !caps.SupportsPrompt {
		return config.Config{}, fmt.Errorf("model %s does not support prompt", cfg.Model)
	}
	if cfg.Outputs.Enabled(domain.ArtifactTimestamps) && !caps.SupportsSegmentTimestamps {
		delete(cfg.Outputs, domain.ArtifactTimestamps)
		logger.Warn().
			Str("provider", string(cfg.Provider)).
			Str("model", cfg.Model).
			Msg("timestamps output is not supported by model; disabling timestamps artifact")
	}
	if cfg.Outputs.Enabled(domain.ArtifactSRT) && !caps.SupportsSRT {
		return config.Config{}, fmt.Errorf("model %s does not support srt artifacts", cfg.Model)
	}
	if cfg.Outputs.Enabled(domain.ArtifactVTT) && !caps.SupportsVTT {
		return config.Config{}, fmt.Errorf("model %s does not support vtt artifacts", cfg.Model)
	}
	if cfg.Outputs.Enabled(domain.ArtifactDiarized) && !caps.SupportsDiarization {
		return config.Config{}, fmt.Errorf("model %s does not support diarization", cfg.Model)
	}

	return cfg, nil
}

func (a *Application) processFile(
	ctx context.Context,
	client provider.Client,
	cfg config.Config,
	inputPath string,
	outputRoot string,
) error {
	baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	fileOutputDir := filepath.Join(outputRoot, baseName)
	fileWorkDir := filepath.Join(fileOutputDir, "_work")

	a.Logger.Info().
		Str("input", inputPath).
		Str("output_dir", fileOutputDir).
		Str("work_dir", fileWorkDir).
		Msg("processing input file")

	if err := a.FS.MkdirAll(fileOutputDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	prepared, err := a.Audio.PrepareInput(ctx, inputPath, fileWorkDir)
	if err != nil {
		return err
	}
	logEvent := a.Logger.Info().
		Str("input", prepared.OriginalPath).
		Str("chunk_source", prepared.ChunkSourcePath).
		Str("work_dir", fileWorkDir)
	if prepared.Converted {
		logEvent.Msg("converted input to m4a for preprocessing")
	} else {
		logEvent.Msg("input already m4a; skipping conversion")
	}

	a.Logger.Info().
		Str("input", prepared.OriginalPath).
		Str("chunk_source", prepared.ChunkSourcePath).
		Str("work_dir", fileWorkDir).
		Int("chunk_seconds", cfg.ChunkSeconds).
		Msg("preparing chunks")

	chunks, err := a.Audio.PrepareChunks(ctx, prepared.ChunkSourcePath, fileWorkDir, cfg.ChunkSeconds)
	if err != nil {
		return err
	}
	a.Logger.Info().
		Str("input", prepared.OriginalPath).
		Str("work_dir", fileWorkDir).
		Int("chunks", len(chunks)).
		Msg("prepared chunks")

	transcript, rawArtifacts, err := a.transcribeChunks(ctx, client, cfg, prepared.OriginalPath, chunks)
	if err != nil {
		return err
	}

	if err := output.WriteArtifacts(a.FS, fileOutputDir, transcript, cfg.Outputs, rawArtifacts); err != nil {
		return err
	}
	a.Logger.Info().
		Str("input", prepared.OriginalPath).
		Str("output_dir", fileOutputDir).
		Msg("wrote transcript artifacts")
	return nil
}

type chunkResult struct {
	chunk    audio.Chunk
	response provider.Response
	err      error
}

func (a *Application) transcribeChunks(
	ctx context.Context,
	client provider.Client,
	cfg config.Config,
	inputPath string,
	chunks []audio.Chunk,
) (domain.Transcript, [][]byte, error) {
	workerCount := cfg.Concurrency
	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
	}
	if workerCount > len(chunks) {
		workerCount = len(chunks)
	}

	jobs := make(chan audio.Chunk, len(chunks))
	results := make(chan chunkResult, len(chunks))

	var wg sync.WaitGroup
	for idx := 0; idx < workerCount; idx++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for chunk := range jobs {
				a.Logger.Info().
					Str("file", chunk.Path).
					Int("chunk", chunk.Number).
					Str("provider", string(cfg.Provider)).
					Str("model", cfg.Model).
					Msg("transcribing chunk")

				response, err := client.Transcribe(ctx, provider.Request{
					FilePath:        chunk.Path,
					Model:           cfg.Model,
					Language:        cfg.Language,
					Prompt:          cfg.Prompt,
					WantDiarization: cfg.Outputs.Enabled(domain.ArtifactDiarized),
					WantRaw:         cfg.Outputs.Enabled(domain.ArtifactRaw),
				})
				results <- chunkResult{chunk: chunk, response: response, err: err}
			}
		}()
	}

	for _, chunk := range chunks {
		jobs <- chunk
	}
	close(jobs)

	wg.Wait()
	close(results)

	collected := make([]chunkResult, 0, len(chunks))
	for result := range results {
		if result.err != nil {
			return domain.Transcript{}, nil, fmt.Errorf("chunk %d: %w", result.chunk.Number, result.err)
		}
		collected = append(collected, result)
	}

	sort.Slice(collected, func(i, j int) bool {
		return collected[i].chunk.Number < collected[j].chunk.Number
	})

	var (
		combined domain.Transcript
		rawItems [][]byte
		texts    []string
	)

	combined.Provider = cfg.Provider
	combined.Model = cfg.Model
	combined.Source = inputPath
	combined.Language = cfg.Language

	for _, item := range collected {
		piece := item.response.Transcript
		if strings.TrimSpace(piece.Language) != "" {
			combined.Language = strings.TrimSpace(piece.Language)
		}
		if text := strings.TrimSpace(piece.PlainText()); text != "" {
			texts = append(texts, text)
		}

		combined.Segments = append(combined.Segments, domain.ShiftSegments(piece.Segments, item.chunk.Offset)...)
		combined.SpeakerSegments = append(combined.SpeakerSegments, domain.ShiftSpeakerSegments(piece.SpeakerSegments, item.chunk.Offset)...)

		if cfg.Outputs.Enabled(domain.ArtifactRaw) && len(item.response.Raw) > 0 {
			rawItems = append(rawItems, item.response.Raw)
		}
	}

	combined.Text = strings.Join(texts, "\n")

	if cfg.Outputs.Enabled(domain.ArtifactTimestamps) || cfg.Outputs.Enabled(domain.ArtifactSRT) || cfg.Outputs.Enabled(domain.ArtifactVTT) {
		if len(combined.Segments) == 0 {
			return domain.Transcript{}, nil, errors.New("requested timestamp-based artifacts but provider returned no segments")
		}
	}
	if cfg.Outputs.Enabled(domain.ArtifactDiarized) && len(combined.SpeakerSegments) == 0 {
		return domain.Transcript{}, nil, errors.New("requested diarized artifact but provider returned no speaker segments")
	}

	return combined, rawItems, nil
}
