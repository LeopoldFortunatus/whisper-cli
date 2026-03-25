package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunPrintsHelpWithoutError(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	if err := run(context.Background(), []string{"-h"}, &stdout); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Usage of whisper-cli:") {
		t.Fatalf("help output did not contain usage header: %q", output)
	}
	if !strings.Contains(output, "-input") {
		t.Fatalf("help output did not contain flag descriptions: %q", output)
	}
}
