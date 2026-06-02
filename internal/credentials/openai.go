package credentials

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/arykalin/whisper-cli/internal/config"
)

const (
	openAIEnvKey = "OPENAI_API_KEY"

	secretToolCommand = "secret-tool"
)

var openAISecretToolLookupArgs = []string{
	"lookup",
	"application", "whisper-cli",
	"provider", "openai",
	"kind", "api-key",
}

type CommandRunner interface {
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
}

type OSCommandRunner struct{}

func (OSCommandRunner) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

type OpenAIKeyResolver struct {
	Env    config.EnvSource
	Runner CommandRunner
}

func NewOpenAIKeyResolver(env config.EnvSource) OpenAIKeyResolver {
	return OpenAIKeyResolver{
		Env:    env,
		Runner: OSCommandRunner{},
	}
}

func (r OpenAIKeyResolver) Resolve(ctx context.Context) (string, error) {
	if r.Runner == nil {
		r.Runner = OSCommandRunner{}
	}

	if secret, err := r.lookupSecretService(ctx); err == nil && secret != "" {
		return secret, nil
	}

	if r.Env != nil {
		if value, ok := r.Env.LookupEnv(openAIEnvKey); ok {
			if key := strings.TrimSpace(value); key != "" {
				return key, nil
			}
		}
	}

	return "", errors.New("OpenAI API key is not configured; store it with `secret-tool store --label='whisper-cli OpenAI API key' application whisper-cli provider openai kind api-key` or export OPENAI_API_KEY for CI/live smoke tests")
}

func (r OpenAIKeyResolver) lookupSecretService(ctx context.Context) (string, error) {
	output, err := r.Runner.Output(ctx, secretToolCommand, openAISecretToolLookupArgs...)
	if err != nil {
		return "", fmt.Errorf("lookup OpenAI API key in Secret Service: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
