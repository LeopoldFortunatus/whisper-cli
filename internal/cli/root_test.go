package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/arykalin/whisper-cli/internal/app"
	"github.com/arykalin/whisper-cli/internal/domain"
	"github.com/arykalin/whisper-cli/internal/provider"
)

type fakeClient struct {
	name   domain.Provider
	models []string
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

func (f fakeClient) Transcribe(context.Context, provider.Request) (provider.Response, error) {
	return provider.Response{}, nil
}

func testApplication() *app.Application {
	return &app.Application{
		Registry: provider.NewRegistry(
			fakeClient{name: domain.ProviderOpenAI, models: []string{"gpt-4o-transcribe", "whisper-1"}},
			fakeClient{name: domain.ProviderGroq, models: []string{"whisper-large-v3-turbo"}},
			provider.NewBlockedClient(domain.ProviderOpenRouter, provider.ErrOpenRouterPlanned),
		),
	}
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
