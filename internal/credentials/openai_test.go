package credentials

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
)

type staticEnv map[string]string

func (s staticEnv) LookupEnv(key string) (string, bool) {
	value, ok := s[key]
	return value, ok
}

type fakeRunner struct {
	output []byte
	err    error
	calls  []runnerCall
}

type runnerCall struct {
	name string
	args []string
}

func (f *fakeRunner) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, runnerCall{
		name: name,
		args: append([]string(nil), args...),
	})
	if f.err != nil {
		return nil, f.err
	}
	return f.output, nil
}

func TestOpenAIKeyResolverReturnsSecretServiceValueBeforeEnv(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{output: []byte(" secret-service-key\n")}
	resolver := OpenAIKeyResolver{
		Env:    staticEnv{openAIEnvKey: "env-key"},
		Runner: runner,
	}

	got, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if got != "secret-service-key" {
		t.Fatalf("Resolve = %q, want Secret Service value", got)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("runner calls = %d, want 1", len(runner.calls))
	}
	if runner.calls[0].name != secretToolCommand {
		t.Fatalf("command = %q, want %q", runner.calls[0].name, secretToolCommand)
	}
	if !slices.Equal(runner.calls[0].args, openAISecretToolLookupArgs) {
		t.Fatalf("args = %v, want %v", runner.calls[0].args, openAISecretToolLookupArgs)
	}
	for _, arg := range runner.calls[0].args {
		if strings.Contains(arg, "secret-service-key") || strings.Contains(arg, "env-key") {
			t.Fatalf("lookup args contain secret material: %v", runner.calls[0].args)
		}
	}
}

func TestOpenAIKeyResolverFallsBackToEnvWhenSecretServiceIsEmpty(t *testing.T) {
	t.Parallel()

	resolver := OpenAIKeyResolver{
		Env:    staticEnv{openAIEnvKey: " env-key\n"},
		Runner: &fakeRunner{output: []byte("\n")},
	}

	got, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if got != "env-key" {
		t.Fatalf("Resolve = %q, want env fallback", got)
	}
}

func TestOpenAIKeyResolverFallsBackToEnvWhenSecretServiceFails(t *testing.T) {
	t.Parallel()

	resolver := OpenAIKeyResolver{
		Env: staticEnv{openAIEnvKey: "env-key"},
		Runner: &fakeRunner{
			err: errors.New("secret service unavailable"),
		},
	}

	got, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if got != "env-key" {
		t.Fatalf("Resolve = %q, want env fallback", got)
	}
}

func TestOpenAIKeyResolverReportsActionableErrorWithoutSecretValue(t *testing.T) {
	t.Parallel()

	resolver := OpenAIKeyResolver{
		Env:    staticEnv{openAIEnvKey: "   "},
		Runner: &fakeRunner{err: errors.New("secret-value must not leak")},
	}

	_, err := resolver.Resolve(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	message := err.Error()
	if !strings.Contains(message, "secret-tool store") {
		t.Fatalf("error does not mention secret-tool store: %v", err)
	}
	if !strings.Contains(message, openAIEnvKey) {
		t.Fatalf("error does not mention %s: %v", openAIEnvKey, err)
	}
	if strings.Contains(message, "secret-value") {
		t.Fatalf("error leaked secret material: %v", err)
	}
}
