package app

import (
	"context"
	"crypto/sha256"
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
	"github.com/arykalin/whisper-cli/internal/credentials"
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
	FS        fsx.FS
	Audio     audio.Pipeline
	Registry  provider.Registry
	Logger    zerolog.Logger
	Env       config.EnvSource
	CacheRoot string

	OpenAIKeyResolver     keyResolver
	OpenAIProviderFactory func(apiKey string) provider.Client
}

type keyResolver interface {
	Resolve(ctx context.Context) (string, error)
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
			openaiadapter.New("", filesystem, logger),
			groqadapter.New(strings.TrimSpace(os.Getenv("GROQ_API_KEY")), filesystem, logger),
			provider.NewBlockedClient(domain.ProviderOpenRouter, provider.ErrOpenRouterPlanned),
		),
		Logger:            logger,
		Env:               config.OSEnv{},
		OpenAIKeyResolver: credentials.NewOpenAIKeyResolver(config.OSEnv{}),
		OpenAIProviderFactory: func(apiKey string) provider.Client {
			return openaiadapter.New(apiKey, filesystem, logger)
		},
	}
}

func (a *Application) Run(ctx context.Context, cfg config.Config) error {
	capabilityClient, err := a.Registry.Provider(cfg.Provider)
	if err != nil {
		return err
	}
	cfg, err = normalizeConfigAgainstCapabilities(capabilityClient, cfg, a.Logger)
	if err != nil {
		return err
	}

	client, err := a.providerForRun(ctx, cfg.Provider)
	if err != nil {
		return err
	}
	if err := client.Preflight(); err != nil {
		return err
	}
	if err := a.Audio.EnsureBinaries(); err != nil {
		return err
	}

	items, err := a.planBatch(cfg)
	if err != nil {
		return err
	}

	var firstErr error
	for _, item := range items {
		if err := a.processFile(ctx, client, cfg, item); err != nil {
			a.Logger.Error().Err(err).Str("file", item.inputPath).Msg("failed to transcribe file")
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

type batchItem struct {
	inputPath string
	publicDir string
	stem      string
	cacheDir  string
	workDir   string
}

func (a *Application) planBatch(cfg config.Config) ([]batchItem, error) {
	inputFiles, err := a.collectInputFiles(cfg.Inputs)
	if err != nil {
		return nil, err
	}

	cacheRoot, err := a.resolveCacheRoot()
	if err != nil {
		return nil, err
	}

	outputRoot := ""
	if cfg.OutputDir != "" {
		outputRoot, err = a.FS.Abs(filepath.Clean(cfg.OutputDir))
		if err != nil {
			return nil, fmt.Errorf("resolve output dir: %w", err)
		}
	}

	items := make([]batchItem, 0, len(inputFiles))
	for _, inputPath := range inputFiles {
		publicDir := filepath.Dir(inputPath)
		if outputRoot != "" {
			publicDir = outputRoot
		}

		stem := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
		cacheDir := filepath.Join(cacheRoot, cacheKey(inputPath))
		items = append(items, batchItem{
			inputPath: inputPath,
			publicDir: publicDir,
			stem:      stem,
			cacheDir:  cacheDir,
			workDir:   filepath.Join(cacheDir, "work"),
		})
	}

	if err := validatePublicPathCollisions(items, cfg.Outputs); err != nil {
		return nil, err
	}
	return items, nil
}

func (a *Application) collectInputFiles(inputs []string) ([]string, error) {
	if len(inputs) == 0 {
		return nil, errors.New("no input specified; use --input or WHISPER_CLI_INPUT")
	}

	files := []string{}
	seen := map[string]struct{}{}
	for _, input := range inputs {
		trimmed := strings.TrimSpace(input)
		if trimmed == "" {
			return nil, errors.New("input path must not be empty")
		}

		inputPath, err := a.FS.Abs(filepath.Clean(trimmed))
		if err != nil {
			return nil, fmt.Errorf("resolve input path %q: %w", trimmed, err)
		}
		info, err := a.FS.Stat(inputPath)
		if err != nil {
			return nil, fmt.Errorf("stat input path %s: %w", inputPath, err)
		}

		candidates := []string{inputPath}
		if info.IsDir() {
			candidates, err = a.Audio.CollectMediaFiles(inputPath)
			if err != nil {
				return nil, fmt.Errorf("scan input directory %s: %w", inputPath, err)
			}
			if len(candidates) == 0 {
				return nil, fmt.Errorf("input directory %s does not contain supported media files", inputPath)
			}
		}

		for _, candidate := range candidates {
			candidatePath, err := a.FS.Abs(filepath.Clean(candidate))
			if err != nil {
				return nil, fmt.Errorf("resolve media path %q: %w", candidate, err)
			}
			if _, ok := seen[candidatePath]; ok {
				continue
			}
			seen[candidatePath] = struct{}{}
			files = append(files, candidatePath)
		}
	}

	return files, nil
}

func (a *Application) resolveCacheRoot() (string, error) {
	cacheRoot := strings.TrimSpace(a.CacheRoot)
	if cacheRoot == "" {
		userCacheDir, err := a.FS.UserCacheDir()
		if err != nil {
			return "", fmt.Errorf("resolve user cache directory: %w", err)
		}
		cacheRoot = filepath.Join(userCacheDir, "whisper-cli")
	}

	absolutePath, err := a.FS.Abs(filepath.Clean(cacheRoot))
	if err != nil {
		return "", fmt.Errorf("resolve cache root: %w", err)
	}
	return absolutePath, nil
}

func validatePublicPathCollisions(items []batchItem, outputs domain.ArtifactSet) error {
	owners := map[string]string{}
	for _, item := range items {
		layout := output.Layout{
			CacheDir:  item.cacheDir,
			PublicDir: item.publicDir,
			Stem:      item.stem,
		}
		for _, artifactPath := range output.PublicArtifactPaths(layout, outputs) {
			if owner, ok := owners[artifactPath]; ok {
				return fmt.Errorf(
					"public output path collision: inputs %q and %q both write %s",
					owner,
					item.inputPath,
					artifactPath,
				)
			}
			owners[artifactPath] = item.inputPath
		}
	}
	return nil
}

func cacheKey(inputPath string) string {
	sum := sha256.Sum256([]byte(inputPath))
	return fmt.Sprintf("%x", sum)
}

func (a *Application) providerForRun(ctx context.Context, providerName domain.Provider) (provider.Client, error) {
	client, err := a.Registry.Provider(providerName)
	if err != nil {
		return nil, err
	}
	if providerName != domain.ProviderOpenAI {
		return client, nil
	}
	if a.OpenAIKeyResolver == nil || a.OpenAIProviderFactory == nil {
		return client, nil
	}

	apiKey, err := a.OpenAIKeyResolver.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	return a.OpenAIProviderFactory(apiKey), nil
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
	return cfg, nil
}

func (a *Application) processFile(
	ctx context.Context,
	client provider.Client,
	cfg config.Config,
	item batchItem,
) error {
	a.Logger.Info().
		Str("input", item.inputPath).
		Str("public_dir", item.publicDir).
		Str("cache_dir", item.cacheDir).
		Msg("processing input file")

	if err := a.FS.RemoveAll(item.cacheDir); err != nil {
		return fmt.Errorf("reset input cache: %w", err)
	}
	if err := a.FS.MkdirAll(item.workDir, 0o700); err != nil {
		return fmt.Errorf("create work directory: %w", err)
	}

	prepared, err := a.Audio.PrepareInput(ctx, item.inputPath, item.workDir)
	if err != nil {
		return err
	}
	logEvent := a.Logger.Info().
		Str("input", prepared.OriginalPath).
		Str("chunk_source", prepared.ChunkSourcePath).
		Str("work_dir", item.workDir)
	if prepared.Converted {
		logEvent.Msg("converted input to m4a for preprocessing")
	} else {
		logEvent.Msg("input already m4a; skipping conversion")
	}

	a.Logger.Info().
		Str("input", prepared.OriginalPath).
		Str("chunk_source", prepared.ChunkSourcePath).
		Str("work_dir", item.workDir).
		Int("chunk_seconds", cfg.ChunkSeconds).
		Msg("preparing chunks")

	chunks, err := a.Audio.PrepareChunks(ctx, prepared.ChunkSourcePath, item.workDir, cfg.ChunkSeconds)
	if err != nil {
		return err
	}
	a.Logger.Info().
		Str("input", prepared.OriginalPath).
		Str("work_dir", item.workDir).
		Int("chunks", len(chunks)).
		Msg("prepared chunks")

	transcript, rawArtifacts, err := a.transcribeChunks(ctx, client, cfg, prepared.OriginalPath, chunks)
	if err != nil {
		return err
	}

	if err := output.WriteArtifacts(a.FS, output.WriteRequest{
		Layout: output.Layout{
			CacheDir:  item.cacheDir,
			PublicDir: item.publicDir,
			Stem:      item.stem,
		},
		Transcript:   transcript,
		Outputs:      cfg.Outputs,
		RawArtifacts: rawArtifacts,
	}); err != nil {
		return err
	}
	a.Logger.Info().
		Str("input", prepared.OriginalPath).
		Str("public_file", filepath.Join(item.publicDir, item.stem+".txt")).
		Str("cache_dir", item.cacheDir).
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
					FilePath: chunk.Path,
					Model:    cfg.Model,
					Language: cfg.Language,
					Prompt:   cfg.Prompt,
					WantRaw:  cfg.Outputs.Enabled(domain.ArtifactRaw),
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
	return combined, rawItems, nil
}
