package config

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/arykalin/whisper-cli/internal/domain"
)

const DefaultProvider = domain.ProviderOpenAI

type StringOverride struct {
	Value    string
	Provided bool
}

func (s *StringOverride) String() string {
	return s.Value
}

func (s *StringOverride) SetValue(value string) {
	s.Value = value
	s.Provided = true
}

func (s *StringOverride) Set(value string) error {
	s.SetValue(value)
	return nil
}

func (s *StringOverride) Type() string {
	return "string"
}

type IntOverride struct {
	Value    int
	Provided bool
}

func (i *IntOverride) String() string {
	return strconv.Itoa(i.Value)
}

func (i *IntOverride) SetValue(value int) {
	i.Value = value
	i.Provided = true
}

func (i *IntOverride) Set(value string) error {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	i.SetValue(parsed)
	return nil
}

func (i *IntOverride) Type() string {
	return "int"
}

type Overrides struct {
	Provider     StringOverride
	Model        StringOverride
	Input        StringOverride
	OutputDir    StringOverride
	Language     StringOverride
	Outputs      StringOverride
	ChunkSeconds IntOverride
	Concurrency  IntOverride
	Prompt       StringOverride
}

type Config struct {
	Provider     domain.Provider
	Model        string
	Input        string
	OutputDir    string
	Language     string
	Outputs      domain.ArtifactSet
	ChunkSeconds int
	Concurrency  int
	Prompt       string
}

type EnvSource interface {
	LookupEnv(key string) (string, bool)
}

type OSEnv struct{}

func (OSEnv) LookupEnv(key string) (string, bool) {
	value, ok := os.LookupEnv(key)
	return value, ok
}

func Resolve(overrides Overrides, env EnvSource) (Config, error) {
	if env == nil {
		env = OSEnv{}
	}

	input := chooseString(overrides.Input, env, "WHISPER_CLI_INPUT", "")
	providerName := chooseString(overrides.Provider, env, "WHISPER_CLI_PROVIDER", string(DefaultProvider))
	model := chooseString(overrides.Model, env, "WHISPER_CLI_MODEL", "")
	if model == "" {
		model = defaultModelForProvider(providerName)
	}

	outputDir := chooseString(overrides.OutputDir, env, "WHISPER_CLI_OUTPUT_DIR", "output")
	language := chooseString(overrides.Language, env, "WHISPER_CLI_LANGUAGE", "ru")
	outputsRaw := chooseString(overrides.Outputs, env, "WHISPER_CLI_OUTPUTS", "timestamps")
	chunkSeconds := chooseInt(overrides.ChunkSeconds, env, "WHISPER_CLI_CHUNK_SECONDS", 600)
	concurrency := chooseInt(overrides.Concurrency, env, "WHISPER_CLI_CONCURRENCY", runtime.NumCPU())
	prompt := chooseString(overrides.Prompt, env, "WHISPER_CLI_PROMPT", "")

	if input == "" {
		return Config{}, errors.New("no input specified; use --input or WHISPER_CLI_INPUT")
	}
	if concurrency <= 0 {
		return Config{}, errors.New("concurrency must be greater than zero")
	}
	if chunkSeconds <= 0 {
		return Config{}, errors.New("chunk-seconds must be greater than zero")
	}

	providerValue := domain.Provider(strings.ToLower(strings.TrimSpace(providerName)))
	switch providerValue {
	case domain.ProviderOpenAI, domain.ProviderGroq, domain.ProviderOpenRouter:
	default:
		return Config{}, fmt.Errorf("unsupported provider %q", providerName)
	}

	outputs, err := domain.ParseArtifactSet(outputsRaw)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Provider:     providerValue,
		Model:        strings.TrimSpace(model),
		Input:        strings.TrimSpace(input),
		OutputDir:    strings.TrimSpace(outputDir),
		Language:     strings.TrimSpace(language),
		Outputs:      outputs,
		ChunkSeconds: chunkSeconds,
		Concurrency:  concurrency,
		Prompt:       strings.TrimSpace(prompt),
	}, nil
}

func chooseString(override StringOverride, env EnvSource, envKey string, fallback string) string {
	if override.Provided {
		return strings.TrimSpace(override.Value)
	}
	if value, ok := env.LookupEnv(envKey); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func chooseInt(override IntOverride, env EnvSource, envKey string, fallback int) int {
	if override.Provided {
		return override.Value
	}
	if value, ok := env.LookupEnv(envKey); ok && strings.TrimSpace(value) != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func defaultModelForProvider(providerName string) string {
	switch domain.Provider(strings.ToLower(strings.TrimSpace(providerName))) {
	case domain.ProviderGroq:
		return "whisper-large-v3-turbo"
	default:
		return "whisper-1"
	}
}
