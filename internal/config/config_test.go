package config

import (
	"strings"
	"testing"

	"github.com/arykalin/whisper-cli/internal/domain"
)

type mapEnv map[string]string

func (m mapEnv) LookupEnv(key string) (string, bool) {
	value, ok := m[key]
	return value, ok
}

func TestResolvePrecedenceFlagsEnvDefaults(t *testing.T) {
	t.Parallel()

	overrides := Overrides{}
	overrides.Input.SetValue("flag-input.mp3")
	overrides.OutputDir.SetValue("flag-output")
	overrides.ChunkSeconds.SetValue(333)

	env := mapEnv{
		"WHISPER_CLI_PROVIDER":    "groq",
		"WHISPER_CLI_MODEL":       "whisper-large-v3",
		"WHISPER_CLI_LANGUAGE":    "de",
		"WHISPER_CLI_OUTPUTS":     "vtt,raw",
		"WHISPER_CLI_CONCURRENCY": "7",
		"WHISPER_CLI_PROMPT":      "env prompt",
	}

	cfg, err := Resolve(overrides, env)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if cfg.Provider != domain.ProviderGroq {
		t.Fatalf("provider = %s, want %s", cfg.Provider, domain.ProviderGroq)
	}
	if cfg.Model != "whisper-large-v3" {
		t.Fatalf("model = %s", cfg.Model)
	}
	if cfg.Input != "flag-input.mp3" {
		t.Fatalf("input = %s", cfg.Input)
	}
	if cfg.OutputDir != "flag-output" {
		t.Fatalf("outputDir = %s", cfg.OutputDir)
	}
	if cfg.Language != "de" {
		t.Fatalf("language = %s", cfg.Language)
	}
	if !cfg.Outputs.Enabled(domain.ArtifactVTT) || !cfg.Outputs.Enabled(domain.ArtifactRaw) {
		t.Fatalf("outputs = %#v", cfg.Outputs)
	}
	if cfg.ChunkSeconds != 333 {
		t.Fatalf("chunkSeconds = %d", cfg.ChunkSeconds)
	}
	if cfg.Concurrency != 7 {
		t.Fatalf("concurrency = %d", cfg.Concurrency)
	}
	if cfg.Prompt != "env prompt" {
		t.Fatalf("prompt = %q", cfg.Prompt)
	}
}

func TestResolveUsesDefaultsWithoutEnv(t *testing.T) {
	t.Parallel()

	overrides := Overrides{}
	overrides.Input.SetValue("input.m4a")

	cfg, err := Resolve(overrides, mapEnv{})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if cfg.Provider != domain.ProviderOpenAI {
		t.Fatalf("provider = %s, want %s", cfg.Provider, domain.ProviderOpenAI)
	}
	if cfg.Model != "whisper-1" {
		t.Fatalf("model = %s, want whisper-1", cfg.Model)
	}
	if cfg.OutputDir != "output" {
		t.Fatalf("outputDir = %s, want output", cfg.OutputDir)
	}
	if cfg.Language != "ru" {
		t.Fatalf("language = %s, want ru", cfg.Language)
	}
	if !cfg.Outputs.Enabled(domain.ArtifactTimestamps) {
		t.Fatalf("outputs = %#v, want timestamps enabled", cfg.Outputs)
	}
}

func TestResolveUsesProviderSpecificDefaultModel(t *testing.T) {
	t.Parallel()

	cfg, err := Resolve(Overrides{}, mapEnv{
		"WHISPER_CLI_INPUT":    "input.m4a",
		"WHISPER_CLI_PROVIDER": "groq",
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if cfg.Model != "whisper-large-v3-turbo" {
		t.Fatalf("model = %s, want whisper-large-v3-turbo", cfg.Model)
	}
}

func TestResolveRejectsMissingInput(t *testing.T) {
	t.Parallel()

	_, err := Resolve(Overrides{}, mapEnv{})
	if err == nil {
		t.Fatal("expected missing input error")
	}
	if !strings.Contains(err.Error(), "--input") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveRejectsInvalidChunkSeconds(t *testing.T) {
	t.Parallel()

	overrides := Overrides{}
	overrides.Input.SetValue("input.m4a")
	overrides.ChunkSeconds.SetValue(0)

	_, err := Resolve(overrides, mapEnv{})
	if err == nil {
		t.Fatal("expected invalid chunk-seconds error")
	}
	if !strings.Contains(err.Error(), "chunk-seconds") {
		t.Fatalf("unexpected error: %v", err)
	}
}
