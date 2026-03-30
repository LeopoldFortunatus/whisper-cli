package completion

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/arykalin/whisper-cli/internal/config"
	"github.com/arykalin/whisper-cli/internal/domain"
	"github.com/arykalin/whisper-cli/internal/platform/fsx"
	"github.com/arykalin/whisper-cli/internal/provider"
	"github.com/arykalin/whisper-cli/internal/provider/groqadapter"
	"github.com/arykalin/whisper-cli/internal/provider/openaiadapter"
	"github.com/rs/zerolog"
)

type providerMetadata struct {
	Name   string
	Models []string
}

type metadata struct {
	Flags           []config.FlagSpec
	Providers       []providerMetadata
	Outputs         []string
	DefaultProvider string
}

func BashScript() string {
	return renderBash(buildMetadata(defaultClients()))
}

func defaultClients() []provider.Client {
	filesystem := fsx.OS{}
	logger := zerolog.New(io.Discard)
	return []provider.Client{
		openaiadapter.New("", filesystem, logger),
		groqadapter.New("", filesystem, logger),
		provider.NewBlockedClient(domain.ProviderOpenRouter, provider.ErrOpenRouterPlanned),
	}
}

func buildMetadata(clients []provider.Client) metadata {
	return metadata{
		Flags:           config.CLIFlagSpecs(),
		Providers:       providerMetadataFromClients(clients),
		Outputs:         artifactNames(domain.KnownArtifacts()),
		DefaultProvider: string(config.DefaultProvider),
	}
}

func providerMetadataFromClients(clients []provider.Client) []providerMetadata {
	items := make([]providerMetadata, 0, len(clients))
	for _, client := range clients {
		models := client.SupportedModels()
		if len(models) == 0 {
			continue
		}
		items = append(items, providerMetadata{
			Name:   string(client.Name()),
			Models: append([]string(nil), models...),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return items
}

func artifactNames(artifacts []domain.ArtifactKind) []string {
	names := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		names = append(names, string(artifact))
	}
	return names
}

func flagNames(specs []config.FlagSpec) []string {
	names := make([]string, 0, len(specs))
	for _, spec := range specs {
		names = append(names, "-"+spec.Name)
	}
	sort.Strings(names)
	return names
}

func flagNamesByType(specs []config.FlagSpec, valueType config.FlagValueType) []string {
	var names []string
	for _, spec := range specs {
		if spec.ValueType == valueType {
			names = append(names, "-"+spec.Name)
		}
	}
	sort.Strings(names)
	return names
}

func providerNames(providers []providerMetadata) []string {
	names := make([]string, 0, len(providers))
	for _, provider := range providers {
		names = append(names, provider.Name)
	}
	return names
}

func casePattern(names []string) string {
	return strings.Join(names, "|")
}

func bashArrayLiteral(values []string) string {
	if len(values) == 0 {
		return ""
	}

	items := make([]string, 0, len(values))
	for _, value := range values {
		items = append(items, fmt.Sprintf("%q", value))
	}
	return strings.Join(items, " ")
}

func renderBash(md metadata) string {
	allFlags := flagNames(md.Flags)
	topLevel := append(append([]string(nil), allFlags...), "completion")
	sort.Strings(topLevel)

	pathFlags := append(
		flagNamesByType(md.Flags, config.FlagValueFilePath),
		flagNamesByType(md.Flags, config.FlagValueInputPath)...,
	)
	sort.Strings(pathFlags)

	dirFlags := flagNamesByType(md.Flags, config.FlagValueDirectoryPath)
	providerFlags := flagNamesByType(md.Flags, config.FlagValueProvider)
	modelFlags := flagNamesByType(md.Flags, config.FlagValueModel)
	outputFlags := flagNamesByType(md.Flags, config.FlagValueOutputs)
	plainFlags := flagNamesByType(md.Flags, config.FlagValuePlain)

	var builder strings.Builder
	builder.WriteString("# bash completion for whisper-cli\n")
	fmt.Fprintf(&builder, "_whisper_cli_default_provider=%q\n", md.DefaultProvider)
	fmt.Fprintf(&builder, "_whisper_cli_top_level=(%s)\n", bashArrayLiteral(topLevel))
	fmt.Fprintf(&builder, "_whisper_cli_providers=(%s)\n", bashArrayLiteral(providerNames(md.Providers)))
	fmt.Fprintf(&builder, "_whisper_cli_outputs=(%s)\n\n", bashArrayLiteral(md.Outputs))

	builder.WriteString("_whisper_cli_contains() {\n")
	builder.WriteString("  local needle=\"$1\"\n")
	builder.WriteString("  shift\n")
	builder.WriteString("  local item\n")
	builder.WriteString("  for item in \"$@\"; do\n")
	builder.WriteString("    if [[ \"$item\" == \"$needle\" ]]; then\n")
	builder.WriteString("      return 0\n")
	builder.WriteString("    fi\n")
	builder.WriteString("  done\n")
	builder.WriteString("  return 1\n")
	builder.WriteString("}\n\n")

	builder.WriteString("_whisper_cli_complete_files() {\n")
	builder.WriteString("  local cur=\"$1\"\n")
	builder.WriteString("  local item\n")
	builder.WriteString("  COMPREPLY=()\n")
	builder.WriteString("  compopt -o filenames 2>/dev/null\n")
	builder.WriteString("  while IFS= read -r item; do\n")
	builder.WriteString("    COMPREPLY+=(\"$item\")\n")
	builder.WriteString("  done < <(compgen -f -- \"$cur\")\n")
	builder.WriteString("}\n\n")

	builder.WriteString("_whisper_cli_complete_directories() {\n")
	builder.WriteString("  local cur=\"$1\"\n")
	builder.WriteString("  local item\n")
	builder.WriteString("  COMPREPLY=()\n")
	builder.WriteString("  compopt -o filenames 2>/dev/null\n")
	builder.WriteString("  while IFS= read -r item; do\n")
	builder.WriteString("    COMPREPLY+=(\"$item\")\n")
	builder.WriteString("  done < <(compgen -d -- \"$cur\")\n")
	builder.WriteString("}\n\n")

	builder.WriteString("_whisper_cli_selected_provider() {\n")
	builder.WriteString("  local provider=\"$_whisper_cli_default_provider\"\n")
	builder.WriteString("  local idx next\n")
	builder.WriteString("  for ((idx = 1; idx < COMP_CWORD; idx++)); do\n")
	builder.WriteString("    if [[ \"${COMP_WORDS[idx]}\" == \"-provider\" ]]; then\n")
	builder.WriteString("      next=$((idx + 1))\n")
	builder.WriteString("      if (( next < ${#COMP_WORDS[@]} )); then\n")
	builder.WriteString("        provider=\"${COMP_WORDS[next]}\"\n")
	builder.WriteString("      fi\n")
	builder.WriteString("    fi\n")
	builder.WriteString("  done\n")
	builder.WriteString("  printf '%s\\n' \"$provider\"\n")
	builder.WriteString("}\n\n")

	builder.WriteString("_whisper_cli_models_for_provider() {\n")
	builder.WriteString("  case \"$1\" in\n")
	for _, provider := range md.Providers {
		fmt.Fprintf(&builder, "    %q)\n", provider.Name)
		fmt.Fprintf(&builder, "      printf '%%s\\n' %s\n", bashArrayLiteral(provider.Models))
		builder.WriteString("      ;;\n")
	}
	builder.WriteString("    *)\n")
	builder.WriteString("      return 0\n")
	builder.WriteString("      ;;\n")
	builder.WriteString("  esac\n")
	builder.WriteString("}\n\n")

	builder.WriteString("_whisper_cli_complete_outputs() {\n")
	builder.WriteString("  local cur=\"$1\"\n")
	builder.WriteString("  local base=\"\"\n")
	builder.WriteString("  local fragment=\"$cur\"\n")
	builder.WriteString("  local candidate\n")
	builder.WriteString("  local used=()\n")
	builder.WriteString("  if [[ \"$cur\" == *,* ]]; then\n")
	builder.WriteString("    base=\"${cur%,*}\"\n")
	builder.WriteString("    fragment=\"${cur##*,}\"\n")
	builder.WriteString("  fi\n")
	builder.WriteString("  if [[ -n \"$base\" ]]; then\n")
	builder.WriteString("    IFS=',' read -r -a used <<< \"$base\"\n")
	builder.WriteString("  fi\n")
	builder.WriteString("  for candidate in \"${used[@]}\"; do\n")
	builder.WriteString("    if [[ \"$candidate\" == \"none\" ]]; then\n")
	builder.WriteString("      COMPREPLY=()\n")
	builder.WriteString("      return 0\n")
	builder.WriteString("    fi\n")
	builder.WriteString("  done\n")
	builder.WriteString("  COMPREPLY=()\n")
	builder.WriteString("  compopt -o nospace 2>/dev/null\n")
	builder.WriteString("  if [[ -z \"$base\" && \"none\" == \"$fragment\"* ]]; then\n")
	builder.WriteString("    COMPREPLY+=(\"none\")\n")
	builder.WriteString("  fi\n")
	builder.WriteString("  for candidate in \"${_whisper_cli_outputs[@]}\"; do\n")
	builder.WriteString("    if _whisper_cli_contains \"$candidate\" \"${used[@]}\"; then\n")
	builder.WriteString("      continue\n")
	builder.WriteString("    fi\n")
	builder.WriteString("    if [[ \"$candidate\" == \"$fragment\"* ]]; then\n")
	builder.WriteString("      if [[ -n \"$base\" ]]; then\n")
	builder.WriteString("        COMPREPLY+=(\"${base},${candidate}\")\n")
	builder.WriteString("      else\n")
	builder.WriteString("        COMPREPLY+=(\"$candidate\")\n")
	builder.WriteString("      fi\n")
	builder.WriteString("    fi\n")
	builder.WriteString("  done\n")
	builder.WriteString("}\n\n")

	builder.WriteString("_whisper_cli() {\n")
	builder.WriteString("  local cur prev provider models\n")
	builder.WriteString("  COMPREPLY=()\n")
	builder.WriteString("  cur=\"${COMP_WORDS[COMP_CWORD]}\"\n")
	builder.WriteString("  prev=\"\"\n")
	builder.WriteString("  if (( COMP_CWORD > 0 )); then\n")
	builder.WriteString("    prev=\"${COMP_WORDS[COMP_CWORD-1]}\"\n")
	builder.WriteString("  fi\n")
	builder.WriteString("  if (( COMP_CWORD == 1 )); then\n")
	builder.WriteString("    COMPREPLY=( $(compgen -W \"${_whisper_cli_top_level[*]}\" -- \"$cur\") )\n")
	builder.WriteString("    return 0\n")
	builder.WriteString("  fi\n")
	builder.WriteString("  if [[ \"${COMP_WORDS[1]}\" == \"completion\" ]]; then\n")
	builder.WriteString("    if (( COMP_CWORD == 2 )); then\n")
	builder.WriteString("      COMPREPLY=( $(compgen -W \"bash\" -- \"$cur\") )\n")
	builder.WriteString("    fi\n")
	builder.WriteString("    return 0\n")
	builder.WriteString("  fi\n")
	builder.WriteString("  case \"$prev\" in\n")
	if pattern := casePattern(providerFlags); pattern != "" {
		fmt.Fprintf(&builder, "    %s)\n", pattern)
		builder.WriteString("      COMPREPLY=( $(compgen -W \"${_whisper_cli_providers[*]}\" -- \"$cur\") )\n")
		builder.WriteString("      return 0\n")
		builder.WriteString("      ;;\n")
	}
	if pattern := casePattern(modelFlags); pattern != "" {
		fmt.Fprintf(&builder, "    %s)\n", pattern)
		builder.WriteString("      provider=\"$(_whisper_cli_selected_provider)\"\n")
		builder.WriteString("      models=\"$(_whisper_cli_models_for_provider \"$provider\")\"\n")
		builder.WriteString("      COMPREPLY=( $(compgen -W \"$models\" -- \"$cur\") )\n")
		builder.WriteString("      return 0\n")
		builder.WriteString("      ;;\n")
	}
	if pattern := casePattern(outputFlags); pattern != "" {
		fmt.Fprintf(&builder, "    %s)\n", pattern)
		builder.WriteString("      _whisper_cli_complete_outputs \"$cur\"\n")
		builder.WriteString("      return 0\n")
		builder.WriteString("      ;;\n")
	}
	if pattern := casePattern(pathFlags); pattern != "" {
		fmt.Fprintf(&builder, "    %s)\n", pattern)
		builder.WriteString("      _whisper_cli_complete_files \"$cur\"\n")
		builder.WriteString("      return 0\n")
		builder.WriteString("      ;;\n")
	}
	if pattern := casePattern(dirFlags); pattern != "" {
		fmt.Fprintf(&builder, "    %s)\n", pattern)
		builder.WriteString("      _whisper_cli_complete_directories \"$cur\"\n")
		builder.WriteString("      return 0\n")
		builder.WriteString("      ;;\n")
	}
	if pattern := casePattern(plainFlags); pattern != "" {
		fmt.Fprintf(&builder, "    %s)\n", pattern)
		builder.WriteString("      return 0\n")
		builder.WriteString("      ;;\n")
	}
	builder.WriteString("  esac\n")
	builder.WriteString("  if [[ \"$cur\" == -* ]]; then\n")
	builder.WriteString("    COMPREPLY=( $(compgen -W \"${_whisper_cli_top_level[*]}\" -- \"$cur\") )\n")
	builder.WriteString("  fi\n")
	builder.WriteString("}\n\n")
	builder.WriteString("complete -F _whisper_cli whisper-cli\n")

	return builder.String()
}
