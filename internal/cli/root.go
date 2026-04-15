package cli

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"slices"
	"strings"

	"github.com/arykalin/whisper-cli/internal/app"
	"github.com/arykalin/whisper-cli/internal/config"
	"github.com/arykalin/whisper-cli/internal/domain"
	"github.com/arykalin/whisper-cli/internal/provider"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	overrides config.Overrides
}

func newRootOptions() rootOptions {
	var opts rootOptions
	opts.overrides.Provider.Value = string(config.DefaultProvider)
	opts.overrides.OutputDir.Value = "output"
	opts.overrides.Language.Value = "ru"
	opts.overrides.Outputs.Value = "timestamps"
	opts.overrides.ChunkSeconds.Value = 600
	opts.overrides.Concurrency.Value = runtime.NumCPU()
	return opts
}

func Run(ctx context.Context, application *app.Application, args []string, stdout io.Writer, stderr io.Writer) error {
	cmd := NewRootCommand(application)
	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	err := cmd.ExecuteContext(ctx)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
	}
	return err
}

func NewRootCommand(application *app.Application) *cobra.Command {
	opts := newRootOptions()

	root := &cobra.Command{
		Use:           "whisper-cli",
		Short:         "Transcribe local media files with OpenAI-compatible speech providers",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Resolve(opts.overrides, envSource(application))
			if err != nil {
				return err
			}
			return application.Run(cmd.Context(), cfg)
		},
	}

	flags := root.Flags()
	flags.SortFlags = false
	flags.Var(&opts.overrides.Provider, "provider", "Provider: openai, groq, openrouter")
	flags.Var(&opts.overrides.Model, "model", "Model name")
	flags.Var(&opts.overrides.Input, "input", "Input media file or directory")
	flags.Var(&opts.overrides.OutputDir, "output-dir", "Output directory root")
	flags.Var(&opts.overrides.Language, "language", "Language code")
	flags.Var(&opts.overrides.Outputs, "outputs", "Optional artifacts: timestamps,srt,vtt,diarized,raw or none")
	flags.Var(&opts.overrides.ChunkSeconds, "chunk-seconds", "Chunk size in seconds")
	flags.Var(&opts.overrides.Concurrency, "concurrency", "Number of worker goroutines")
	flags.Var(&opts.overrides.Prompt, "prompt", "Prompt for supported models")

	must(root.RegisterFlagCompletionFunc("provider", completeProviders(application.Registry)))
	must(root.RegisterFlagCompletionFunc("model", completeModels(application.Registry, &opts)))
	must(root.RegisterFlagCompletionFunc("outputs", completeOutputs))
	must(root.RegisterFlagCompletionFunc("input", completeInputPaths))
	must(root.MarkFlagDirname("output-dir"))

	root.AddCommand(newCompletionCommand(root))
	return root
}

func newCompletionCommand(root *cobra.Command) *cobra.Command {
	completion := &cobra.Command{
		Use:   "completion",
		Short: "Generate shell completion scripts",
		Args:  cobra.NoArgs,
	}

	completion.AddCommand(&cobra.Command{
		Use:                   "bash",
		Short:                 "Print Bash completion script to stdout",
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return root.GenBashCompletionV2(cmd.OutOrStdout(), true)
		},
	})

	return completion
}

func completeProviders(registry provider.Registry) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		var matches []string
		for _, item := range visibleProviders(registry) {
			if strings.HasPrefix(item, toComplete) {
				matches = append(matches, item)
			}
		}
		return matches, cobra.ShellCompDirectiveNoFileComp
	}
}

func completeModels(registry provider.Registry, opts *rootOptions) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		providerName := strings.TrimSpace(opts.overrides.Provider.Value)
		if providerName == "" {
			providerName = string(config.DefaultProvider)
		}

		var matches []string
		for _, model := range modelsForProvider(registry, domain.Provider(providerName)) {
			if strings.HasPrefix(model, toComplete) {
				matches = append(matches, model)
			}
		}
		return matches, cobra.ShellCompDirectiveNoFileComp
	}
}

func completeOutputs(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	base := ""
	fragment := toComplete
	if before, after, ok := strings.Cut(toComplete, ","); ok {
		base = before
		fragment = after
		if strings.Contains(after, ",") {
			base = toComplete[:strings.LastIndex(toComplete, ",")]
			fragment = toComplete[strings.LastIndex(toComplete, ",")+1:]
		}
	}

	used := splitOutputs(base)
	if slices.Contains(used, "none") {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var matches []string
	if base == "" && strings.HasPrefix("none", fragment) {
		matches = append(matches, "none")
	}
	for _, artifact := range domain.KnownArtifacts() {
		name := string(artifact)
		if slices.Contains(used, name) || !strings.HasPrefix(name, fragment) {
			continue
		}
		if base == "" {
			matches = append(matches, name)
			continue
		}
		matches = append(matches, base+","+name)
	}

	return matches, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
}

func completeInputPaths(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveDefault
}

func visibleProviders(registry provider.Registry) []string {
	var items []string
	for _, client := range registry.Clients() {
		if len(client.SupportedModels()) == 0 {
			continue
		}
		items = append(items, string(client.Name()))
	}
	return items
}

func modelsForProvider(registry provider.Registry, providerName domain.Provider) []string {
	client, err := registry.Provider(providerName)
	if err != nil {
		return nil
	}

	models := append([]string(nil), client.SupportedModels()...)
	slices.Sort(models)
	return models
}

func splitOutputs(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		items = append(items, trimmed)
	}
	return items
}

func envSource(application *app.Application) config.EnvSource {
	if application != nil && application.Env != nil {
		return application.Env
	}
	return config.OSEnv{}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
