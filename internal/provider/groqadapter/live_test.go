package groqadapter

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/arykalin/whisper-cli/internal/platform/fsx"
	"github.com/arykalin/whisper-cli/internal/provider"
	"github.com/rs/zerolog"
)

func TestLiveGroqTranscription(t *testing.T) {
	if os.Getenv("WHISPER_CLI_RUN_LIVE") == "" {
		t.Skip("live tests disabled")
	}

	audioPath := os.Getenv("WHISPER_CLI_LIVE_AUDIO")
	if audioPath == "" {
		t.Fatal("WHISPER_CLI_LIVE_AUDIO is required")
	}

	client := New(os.Getenv("GROQ_API_KEY"), fsx.OS{}, zerolog.New(io.Discard))
	if err := client.Preflight(); err != nil {
		t.Fatal(err)
	}

	response, err := client.Transcribe(context.Background(), provider.Request{
		FilePath: audioPath,
		Model:    "whisper-large-v3-turbo",
		Language: "ru",
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Transcript.PlainText() == "" {
		t.Fatal("empty transcript")
	}
}
