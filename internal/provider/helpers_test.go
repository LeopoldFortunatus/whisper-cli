package provider

import (
	"testing"

	"github.com/arykalin/whisper-cli/internal/domain"
)

func TestParseOpenAICompatibleTranscriptLanguage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "plural languages",
			raw:  `{"text":"hello","languages":[{"code":"ru"},{"code":"en"}]}`,
			want: "ru",
		},
		{
			name: "legacy singular language",
			raw:  `{"text":"hello","language":"de"}`,
			want: "de",
		},
		{
			name: "skip blank plural language",
			raw:  `{"text":"hello","languages":[{"code":" "},{"code":"en"}]}`,
			want: "en",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			transcript := ParseOpenAICompatibleTranscript(
				domain.ProviderOpenAI,
				"gpt-transcribe",
				[]byte(tt.raw),
				"fallback",
			)
			if transcript.Language != tt.want {
				t.Fatalf("language = %q, want %q", transcript.Language, tt.want)
			}
			if transcript.Text != "hello" {
				t.Fatalf("text = %q, want hello", transcript.Text)
			}
		})
	}
}
