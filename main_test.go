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
	if !strings.Contains(output, "completion bash") {
		t.Fatalf("help output did not mention completion command: %q", output)
	}
}

func TestRunPrintsBashCompletionWithoutError(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	if err := run(context.Background(), []string{"completion", "bash"}, &stdout); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "_whisper_cli()") {
		t.Fatalf("completion output did not contain completion function: %q", output)
	}
	if !strings.Contains(output, "complete -F _whisper_cli whisper-cli") {
		t.Fatalf("completion output did not contain registration line: %q", output)
	}
}

func TestRunRejectsInvalidCompletionArgs(t *testing.T) {
	t.Parallel()

	tests := [][]string{
		{"completion"},
		{"completion", "zsh"},
		{"completion", "bash", "extra"},
	}

	for _, args := range tests {
		var stdout bytes.Buffer
		err := run(context.Background(), args, &stdout)
		if err == nil {
			t.Fatalf("run(%v) returned nil error", args)
		}
		if !strings.Contains(err.Error(), "usage: whisper-cli completion bash") {
			t.Fatalf("run(%v) error = %q", args, err)
		}
		if stdout.Len() != 0 {
			t.Fatalf("run(%v) unexpectedly wrote to stdout: %q", args, stdout.String())
		}
	}
}
