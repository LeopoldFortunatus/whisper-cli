package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arykalin/whisper-cli/internal/domain"
	"github.com/arykalin/whisper-cli/internal/platform/fsx"
)

type mapEnv map[string]string

func (m mapEnv) LookupEnv(key string) (string, bool) {
	value, ok := m[key]
	return value, ok
}

func TestResolvePrecedenceFlagsEnvFileDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := []byte(`
provider: openai
model: whisper-1
input: yaml-input.m4a
output_dir: yaml-output
language: en
outputs: srt
chunk_seconds: 111
concurrency: 2
prompt: yaml prompt
`)
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	flags := Flags{}
	flags.ConfigPath.Value = configPath
	flags.ConfigPath.Provided = true
	flags.Input.SetValue("flag-input.mp3")
	flags.OutputDir.SetValue("flag-output")
	flags.ChunkSeconds.SetValue(333)

	env := mapEnv{
		"WHISPER_CLI_PROVIDER":    "groq",
		"WHISPER_CLI_MODEL":       "whisper-large-v3",
		"WHISPER_CLI_LANGUAGE":    "de",
		"WHISPER_CLI_OUTPUTS":     "vtt,raw",
		"WHISPER_CLI_CONCURRENCY": "7",
		"WHISPER_CLI_PROMPT":      "env prompt",
	}

	cfg, warnings, err := Resolve(flags, env, fsx.OS{})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
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

func TestResolveLegacyConfigFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := []byte(`
input_file: legacy-input.m4a
usergpt4: true
`)
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	flags := Flags{}
	flags.ConfigPath.Value = configPath
	flags.ConfigPath.Provided = true

	cfg, warnings, err := Resolve(flags, mapEnv{}, fsx.OS{})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if cfg.Input != "legacy-input.m4a" {
		t.Fatalf("input = %s", cfg.Input)
	}
	if cfg.Provider != domain.ProviderOpenAI {
		t.Fatalf("provider = %s", cfg.Provider)
	}
	if cfg.Model != "gpt-4o-transcribe" {
		t.Fatalf("model = %s", cfg.Model)
	}
	if len(warnings) != 2 {
		t.Fatalf("warnings = %v, want 2 warnings", warnings)
	}
}
