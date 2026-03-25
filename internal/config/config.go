package config

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/arykalin/whisper-cli/internal/domain"
	"github.com/arykalin/whisper-cli/internal/platform/fsx"
	"gopkg.in/yaml.v3"
)

const defaultConfigPath = "config.yaml"

type StringFlag struct {
	Value    string
	Provided bool
}

func (s *StringFlag) String() string {
	return s.Value
}

func (s *StringFlag) SetValue(value string) {
	s.Value = value
	s.Provided = true
}

func (s *StringFlag) Set(value string) error {
	s.SetValue(value)
	return nil
}

type IntFlag struct {
	Value    int
	Provided bool
}

func (i *IntFlag) String() string {
	return strconv.Itoa(i.Value)
}

func (i *IntFlag) SetValue(value int) {
	i.Value = value
	i.Provided = true
}

func (i *IntFlag) Set(value string) error {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	i.SetValue(parsed)
	return nil
}

type Flags struct {
	ConfigPath   StringFlag
	Provider     StringFlag
	Model        StringFlag
	Input        StringFlag
	OutputDir    StringFlag
	Language     StringFlag
	Outputs      StringFlag
	ChunkSeconds IntFlag
	Concurrency  IntFlag
	Prompt       StringFlag
}

type Config struct {
	ConfigPath   string
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

type HelpError struct {
	Usage string
}

func (e *HelpError) Error() string {
	return flag.ErrHelp.Error()
}

func (e *HelpError) Unwrap() error {
	return flag.ErrHelp
}

type EnvSource interface {
	LookupEnv(key string) (string, bool)
}

type OSEnv struct{}

func (OSEnv) LookupEnv(key string) (string, bool) {
	value, ok := os.LookupEnv(key)
	return strings.TrimSpace(value), ok
}

type fileConfig struct {
	Provider     string `yaml:"provider"`
	Model        string `yaml:"model"`
	Input        string `yaml:"input"`
	InputFile    string `yaml:"input_file"`
	OutputDir    string `yaml:"output_dir"`
	Language     string `yaml:"language"`
	Outputs      string `yaml:"outputs"`
	ChunkSeconds int    `yaml:"chunk_seconds"`
	Concurrency  int    `yaml:"concurrency"`
	Prompt       string `yaml:"prompt"`
	UseGPT4      *bool  `yaml:"usergpt4"`
}

func ParseFlags(args []string) (Flags, error) {
	flags := Flags{}
	flags.ConfigPath.Value = defaultConfigPath

	set := flag.NewFlagSet("whisper-cli", flag.ContinueOnError)
	var stderr bytes.Buffer
	set.SetOutput(&stderr)

	set.Var(&flags.ConfigPath, "config", "Path to config file")
	set.Var(&flags.Provider, "provider", "Provider: openai, groq, openrouter")
	set.Var(&flags.Model, "model", "Model name")
	set.Var(&flags.Input, "input", "Input audio file or directory")
	set.Var(&flags.OutputDir, "output-dir", "Output directory root")
	set.Var(&flags.Language, "language", "Language code")
	set.Var(&flags.Outputs, "outputs", "Optional artifacts: timestamps,srt,vtt,diarized,raw or none")
	set.Var(&flags.ChunkSeconds, "chunk-seconds", "Chunk size in seconds")
	set.Var(&flags.Concurrency, "concurrency", "Number of worker goroutines")
	set.Var(&flags.Prompt, "prompt", "Prompt for supported models")

	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return Flags{}, &HelpError{Usage: stderr.String()}
		}
		return Flags{}, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}

	return flags, nil
}

func Resolve(flags Flags, env EnvSource, fs fsx.FS) (Config, []string, error) {
	fileCfg, err := loadFileConfig(flags.ConfigPath.Value, flags.ConfigPath.Provided, fs)
	if err != nil {
		return Config{}, nil, err
	}

	var warnings []string

	input := chooseString(flags.Input, env, "WHISPER_CLI_INPUT", fileCfg.Input, "")
	if input == "" && strings.TrimSpace(fileCfg.InputFile) != "" {
		input = strings.TrimSpace(fileCfg.InputFile)
		warnings = append(warnings, "config field input_file is deprecated; use input")
	}

	providerName := chooseString(flags.Provider, env, "WHISPER_CLI_PROVIDER", fileCfg.Provider, string(domain.ProviderOpenAI))
	model := chooseString(flags.Model, env, "WHISPER_CLI_MODEL", fileCfg.Model, "")
	if model == "" && fileCfg.UseGPT4 != nil {
		model = legacyModel(*fileCfg.UseGPT4)
		warnings = append(warnings, "config field usergpt4 is deprecated; use model")
	}
	if model == "" {
		model = defaultModelForProvider(providerName)
	}

	outputDir := chooseString(flags.OutputDir, env, "WHISPER_CLI_OUTPUT_DIR", fileCfg.OutputDir, "output")
	language := chooseString(flags.Language, env, "WHISPER_CLI_LANGUAGE", fileCfg.Language, "ru")
	outputsRaw := chooseString(flags.Outputs, env, "WHISPER_CLI_OUTPUTS", fileCfg.Outputs, "timestamps")
	chunkSeconds := chooseInt(flags.ChunkSeconds, env, "WHISPER_CLI_CHUNK_SECONDS", fileCfg.ChunkSeconds, 600)
	concurrency := chooseInt(flags.Concurrency, env, "WHISPER_CLI_CONCURRENCY", fileCfg.Concurrency, runtime.NumCPU())
	prompt := chooseString(flags.Prompt, env, "WHISPER_CLI_PROMPT", fileCfg.Prompt, "")

	if input == "" {
		return Config{}, warnings, errors.New("no input specified; use -input, WHISPER_CLI_INPUT or config.yaml")
	}

	if concurrency <= 0 {
		return Config{}, warnings, errors.New("concurrency must be greater than zero")
	}
	if chunkSeconds <= 0 {
		return Config{}, warnings, errors.New("chunk_seconds must be greater than zero")
	}

	providerValue := domain.Provider(strings.ToLower(strings.TrimSpace(providerName)))
	switch providerValue {
	case domain.ProviderOpenAI, domain.ProviderGroq, domain.ProviderOpenRouter:
	default:
		return Config{}, warnings, fmt.Errorf("unsupported provider %q", providerName)
	}

	outputs, err := domain.ParseArtifactSet(outputsRaw)
	if err != nil {
		return Config{}, warnings, err
	}

	return Config{
		ConfigPath:   filepath.Clean(flags.ConfigPath.Value),
		Provider:     providerValue,
		Model:        strings.TrimSpace(model),
		Input:        strings.TrimSpace(input),
		OutputDir:    strings.TrimSpace(outputDir),
		Language:     strings.TrimSpace(language),
		Outputs:      outputs,
		ChunkSeconds: chunkSeconds,
		Concurrency:  concurrency,
		Prompt:       strings.TrimSpace(prompt),
	}, warnings, nil
}

func loadFileConfig(path string, explicit bool, fs fsx.FS) (fileConfig, error) {
	info, err := fs.Stat(path)
	if err != nil {
		if explicit {
			return fileConfig{}, fmt.Errorf("stat config %s: %w", path, err)
		}
		return fileConfig{}, nil
	}
	if info.IsDir() {
		return fileConfig{}, fmt.Errorf("config path %s is a directory", path)
	}

	data, err := fs.ReadFile(path)
	if err != nil {
		return fileConfig{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg fileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fileConfig{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}

func chooseString(flagValue StringFlag, env EnvSource, envKey string, fileValue string, fallback string) string {
	if flagValue.Provided {
		return strings.TrimSpace(flagValue.Value)
	}
	if value, ok := env.LookupEnv(envKey); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	if strings.TrimSpace(fileValue) != "" {
		return strings.TrimSpace(fileValue)
	}
	return fallback
}

func chooseInt(flagValue IntFlag, env EnvSource, envKey string, fileValue int, fallback int) int {
	if flagValue.Provided {
		return flagValue.Value
	}
	if value, ok := env.LookupEnv(envKey); ok && strings.TrimSpace(value) != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil {
			return parsed
		}
	}
	if fileValue > 0 {
		return fileValue
	}
	return fallback
}

func legacyModel(useGPT4 bool) string {
	if useGPT4 {
		return "gpt-4o-transcribe"
	}
	return "whisper-1"
}

func defaultModelForProvider(providerName string) string {
	switch domain.Provider(strings.ToLower(strings.TrimSpace(providerName))) {
	case domain.ProviderGroq:
		return "whisper-large-v3-turbo"
	default:
		return "whisper-1"
	}
}
