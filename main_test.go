package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestBashCompletionSmoke(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "whisper-cli")
	scriptPath := filepath.Join(dir, "whisper-cli.bash")

	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Env = append(os.Environ(), "GOFLAGS=-mod=vendor")
	output, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("build binary: %v\n%s", err, output)
	}

	script, err := exec.Command(binPath, "completion", "bash").Output()
	if err != nil {
		t.Fatalf("generate bash completion: %v", err)
	}
	if err := os.WriteFile(scriptPath, script, 0o644); err != nil {
		t.Fatalf("write completion script: %v", err)
	}

	providers := completeWords(t, dir, scriptPath, []string{"whisper-cli", "--provider", "o"})
	if !slices.Equal(providers, []string{"openai"}) {
		t.Fatalf("provider completion = %v, want [openai]", providers)
	}

	models := completeWords(t, dir, scriptPath, []string{"whisper-cli", "--provider", "groq", "--model", "whisper"})
	wantModels := []string{"whisper-large-v3", "whisper-large-v3-turbo"}
	if !slices.Equal(models, wantModels) {
		t.Fatalf("model completion = %v, want %v", models, wantModels)
	}
	for _, reply := range models {
		if strings.Contains(reply, "gpt-4o") {
			t.Fatalf("unexpected openai model in groq completion: %v", models)
		}
	}

	outputs := completeWords(t, dir, scriptPath, []string{"whisper-cli", "--outputs", "timestamps,s"})
	if !slices.Equal(outputs, []string{"timestamps,srt"}) {
		t.Fatalf("outputs completion = %v, want [timestamps,srt]", outputs)
	}

	none := completeWords(t, dir, scriptPath, []string{"whisper-cli", "--outputs", "n"})
	if !slices.Equal(none, []string{"none"}) {
		t.Fatalf("none completion = %v, want [none]", none)
	}
}

func completeWords(t *testing.T, binDir string, scriptPath string, words []string) []string {
	t.Helper()

	cmd := exec.Command("bash", "-lc", fmt.Sprintf(`
PATH=%q:$PATH
_get_comp_words_by_ref() {
    if [[ $1 == "-n" ]]; then
        shift 2
    fi
    while [[ $# -gt 0 ]]; do
        case "$1" in
            cur)
                printf -v "$1" '%%s' "${COMP_WORDS[COMP_CWORD]}"
                ;;
            prev)
                local prevValue=""
                if (( COMP_CWORD > 0 )); then
                    prevValue=${COMP_WORDS[COMP_CWORD-1]}
                fi
                printf -v "$1" '%%s' "$prevValue"
                ;;
            words)
                eval "$1=(\"\${COMP_WORDS[@]}\")"
                ;;
            cword)
                printf -v "$1" '%%s' "$COMP_CWORD"
                ;;
        esac
        shift
    done
}
source %q
fn=$(complete -p whisper-cli | sed -n 's/.*-F \([^ ]*\) whisper-cli/\1/p')
COMP_WORDS=(%s)
COMP_CWORD=%d
COMP_LINE=%q
COMP_POINT=%d
COMP_TYPE=9
"$fn"
printf '%%s\n' "${COMPREPLY[@]}"
`, binDir, scriptPath, bashWords(words), len(words)-1, strings.Join(words, " "), len(strings.Join(words, " "))))
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
