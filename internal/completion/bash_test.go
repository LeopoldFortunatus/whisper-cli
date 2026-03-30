package completion

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestBuildMetadataUsesSortedVisibleProviders(t *testing.T) {
	t.Parallel()

	md := buildMetadata(defaultClients())
	got := providerNames(md.Providers)
	want := []string{"groq", "openai"}

	if !slices.Equal(got, want) {
		t.Fatalf("providers = %v, want %v", got, want)
	}
	for _, provider := range got {
		if provider == "openrouter" {
			t.Fatalf("unexpected planned provider in completion metadata: %v", got)
		}
	}
}

func TestBashScriptCompletesProviders(t *testing.T) {
	t.Parallel()

	replies := completeWords(t, BashScript(), []string{"whisper-cli", "-provider", "o"})
	if !slices.Equal(replies, []string{"openai"}) {
		t.Fatalf("provider completion = %v, want [openai]", replies)
	}
}

func TestBashScriptCompletesModelsForSelectedProvider(t *testing.T) {
	t.Parallel()

	replies := completeWords(t, BashScript(), []string{"whisper-cli", "-provider", "groq", "-model", "whisper"})
	want := []string{"whisper-large-v3", "whisper-large-v3-turbo"}
	if !slices.Equal(replies, want) {
		t.Fatalf("model completion = %v, want %v", replies, want)
	}
	for _, reply := range replies {
		if strings.Contains(reply, "gpt-4o") {
			t.Fatalf("unexpected openai model in groq completion: %v", replies)
		}
	}
}

func TestBashScriptCompletesModelsForDefaultProvider(t *testing.T) {
	t.Parallel()

	replies := completeWords(t, BashScript(), []string{"whisper-cli", "-model", "gpt"})
	want := []string{"gpt-4o-mini-transcribe", "gpt-4o-transcribe", "gpt-4o-transcribe-diarize"}
	if !slices.Equal(replies, want) {
		t.Fatalf("default provider model completion = %v, want %v", replies, want)
	}
}

func TestBashScriptCompletesOutputsWithoutDuplicatesOrNoneMix(t *testing.T) {
	t.Parallel()

	replies := completeWords(t, BashScript(), []string{"whisper-cli", "-outputs", "timestamps,s"})
	if !slices.Equal(replies, []string{"timestamps,srt"}) {
		t.Fatalf("outputs completion = %v, want [timestamps,srt]", replies)
	}
	for _, reply := range replies {
		if strings.Contains(reply, "none") || strings.Contains(reply, "timestamps,timestamps") {
			t.Fatalf("unexpected outputs completion reply: %v", replies)
		}
	}
}

func TestBashScriptCompletesNoneOnlyWhenItIsStandalone(t *testing.T) {
	t.Parallel()

	replies := completeWords(t, BashScript(), []string{"whisper-cli", "-outputs", "n"})
	if !slices.Equal(replies, []string{"none"}) {
		t.Fatalf("outputs completion for none = %v, want [none]", replies)
	}
}

func completeWords(t *testing.T, script string, words []string) []string {
	t.Helper()

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "whisper-cli.bash")
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write completion script: %v", err)
	}

	cmd := exec.Command("bash", "-lc", fmt.Sprintf(`
source %q
COMP_WORDS=(%s)
COMP_CWORD=%d
_whisper_cli
printf '%%s\n' "${COMPREPLY[@]}"
`, scriptPath, bashWords(words), len(words)-1))
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("run bash completion smoke: %v", err)
	}

	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

func bashWords(words []string) string {
	quoted := make([]string, 0, len(words))
	for _, word := range words {
		quoted = append(quoted, fmt.Sprintf("%q", word))
	}
	return strings.Join(quoted, " ")
}
