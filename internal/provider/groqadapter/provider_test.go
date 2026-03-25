package groqadapter

import (
	"io"
	"strings"
	"testing"

	"github.com/arykalin/whisper-cli/internal/platform/fsx"
	"github.com/rs/zerolog"
)

func TestProviderPreflightRequiresKey(t *testing.T) {
	t.Parallel()

	client := New("", fsx.OS{}, zerolog.New(io.Discard))
	if err := client.Preflight(); err == nil || !strings.Contains(err.Error(), "export GROQ_API_KEY") {
		t.Fatalf("expected preflight error for missing key")
	}
}
