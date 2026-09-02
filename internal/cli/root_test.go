package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arykalin/whisper-cli/internal/app"
	"github.com/arykalin/whisper-cli/internal/audio"
	"github.com/arykalin/whisper-cli/internal/domain"
	"github.com/arykalin/whisper-cli/internal/platform/fsx"
	"github.com/arykalin/whisper-cli/internal/provider"
	"github.com/rs/zerolog"
)

type fakeClient struct {
	name           domain.Provider
	models         []string
	transcribeFunc func(context.Context, provider.Request) (provider.Response, error)
}

func (f fakeClient) Name() domain.Provider {
	return f.name
}

func (f fakeClient) Preflight() error {
	return nil
}

func (f fakeClient) Capabilities(string) (domain.Capabilities, bool) {
	return domain.Capabilities{}, true
}

func (f fakeClient) SupportedModels() []string {
	return append([]string(nil), f.models...)
}

func (f fakeClient) Transcribe(ctx context.Context, req provider.Request) (provider.Response, error) {
	if f.transcribeFunc != nil {
		return f.transcribeFunc(ctx, req)
	}
	return provider.Response{}, nil
}

type emptyEnv struct{}

func (emptyEnv) LookupEnv(string) (string, bool) {
	return "", false
}

type cliAudioPipeline struct{}

func (cliAudioPipeline) EnsureBinaries() error {
	return nil
}

func (cliAudioPipeline) CollectMediaFiles(string) ([]string, error) {
	return nil, nil
}

func (cliAudioPipeline) PrepareInput(_ context.Context, inputFile string, _ string) (audio.PreparedInput, error) {
	return audio.PreparedInput{
		OriginalPath:    inputFile,
		ChunkSourcePath: inputFile,
	}, nil
}

func (cliAudioPipeline) PrepareChunks(_ context.Context, inputFile string, _ string, _ int) ([]audio.Chunk, error) {
	return []audio.Chunk{{Number: 0, Path: inputFile}}, nil
}

func testApplication() *app.Application {
	return &app.Application{
		Registry: provider.NewRegistry(
			fakeClient{name: domain.ProviderOpenAI, models: []string{"gpt-transcribe"}},
			fakeClient{name: domain.ProviderGroq, models: []string{"whisper-large-v3-turbo"}},
			provider.NewBlockedClient(domain.ProviderOpenRouter, provider.ErrOpenRouterPlanned),
		),
	}
}

type countingResolver struct {
	calls int
}

func (c *countingResolver) Resolve(context.Context) (string, error) {
	c.calls++
	return "openai-key", nil
}

func TestRunPrintsHelpWithoutError(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run(context.Background(), testApplication(), []string{"--help"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "whisper-cli") {
		t.Fatalf("help output did not contain command name: %q", output)
	}
	if !strings.Contains(output, "--input") {
		t.Fatalf("help output did not contain GNU long flags: %q", output)
	}
	if !strings.Contains(output, "completion") {
		t.Fatalf("help output did not contain completion command: %q", output)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunHelpDoesNotResolveOpenAIKey(t *testing.T) {
	t.Parallel()

	resolver := &countingResolver{}
	application := testApplication()
	application.OpenAIKeyResolver = resolver
	application.OpenAIProviderFactory = func(apiKey string) provider.Client {
		t.Fatalf("unexpected OpenAI provider factory call with %q", apiKey)
		return nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run(context.Background(), application, []string{"--help"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if resolver.calls != 0 {
		t.Fatalf("resolver calls = %d, want 0", resolver.calls)
	}
}

func TestRunPrintsBashCompletionWithoutError(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run(context.Background(), testApplication(), []string{"completion", "bash"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "__complete") {
		t.Fatalf("completion output did not contain dynamic completion hook: %q", output)
	}
	if !strings.Contains(output, "whisper-cli") {
		t.Fatalf("completion output did not contain command name: %q", output)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunBashCompletionDoesNotResolveOpenAIKey(t *testing.T) {
	t.Parallel()

	resolver := &countingResolver{}
	application := testApplication()
	application.OpenAIKeyResolver = resolver
	application.OpenAIProviderFactory = func(apiKey string) provider.Client {
		t.Fatalf("unexpected OpenAI provider factory call with %q", apiKey)
		return nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run(context.Background(), application, []string{"completion", "bash"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if resolver.calls != 0 {
		t.Fatalf("resolver calls = %d, want 0", resolver.calls)
	}
}

func TestRunRejectsUnknownFlag(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run(context.Background(), testApplication(), []string{"--unknown"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if !strings.Contains(stderr.String(), "unknown flag: --unknown") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
}

func TestRunRejectsMissingInput(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run(context.Background(), testApplication(), []string{"--provider", "openai"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing input")
	}
	if !strings.Contains(stderr.String(), "no input specified; use --input or WHISPER_CLI_INPUT") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunRejectsRemovedConfigFlag(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run(context.Background(), testApplication(), []string{"--config", "config.yaml"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for removed --config flag")
	}
	if !strings.Contains(stderr.String(), "unknown flag: --config") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunRejectsLegacySingleDashLongFlag(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run(context.Background(), testApplication(), []string{"-input", "file.m4a"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for legacy single-dash long flag")
	}
	if !strings.Contains(stderr.String(), "unknown shorthand flag") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunAcceptsShellExpandedInputsAfterInputFlag(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	firstInput := filepath.Join(dir, "first.m4a")
	secondInput := filepath.Join(dir, "second.m4a")
	for _, input := range []string{firstInput, secondInput} {
		if err := os.WriteFile(input, []byte("audio"), 0o644); err != nil {
			t.Fatalf("write input %s: %v", input, err)
		}
	}

	transcribed := []string{}
	client := fakeClient{
		name:   domain.ProviderGroq,
		models: []string{"whisper-large-v3-turbo"},
		transcribeFunc: func(_ context.Context, req provider.Request) (provider.Response, error) {
			transcribed = append(transcribed, req.FilePath)
			return provider.Response{
				Transcript: domain.Transcript{Text: filepath.Base(req.FilePath)},
			}, nil
		},
	}
	application := &app.Application{
		FS:        fsx.OS{},
		Audio:     cliAudioPipeline{},
		Registry:  provider.NewRegistry(client),
		Logger:    zerolog.New(io.Discard),
		Env:       emptyEnv{},
		CacheRoot: filepath.Join(dir, "cache"),
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run(
		context.Background(),
		application,
		[]string{
			"--provider", "groq",
			"--model", "whisper-large-v3-turbo",
			"--outputs", "none",
			"--input", firstInput,
			secondInput,
		},
		&stdout,
		&stderr,
	)
	if err != nil {
		t.Fatalf("Run returned error: %v; stderr = %q", err, stderr.String())
	}
	if strings.Join(transcribed, ",") != strings.Join([]string{firstInput, secondInput}, ",") {
		t.Fatalf("transcribed = %v, want ordered expanded inputs", transcribed)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
	for _, name := range []string{"first.txt", "second.txt"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("expected %s: %v", name, err)
		}
	}
}
