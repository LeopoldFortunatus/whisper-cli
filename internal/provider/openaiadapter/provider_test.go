package openaiadapter

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arykalin/whisper-cli/internal/platform/fsx"
	"github.com/arykalin/whisper-cli/internal/provider"
	"github.com/openai/openai-go"
	"github.com/rs/zerolog"
)

type fakeRequester struct {
	steps      []fakeStep
	callCount  int
	lastParams []openai.AudioTranscriptionNewParams
}

type fakeStep struct {
	response *openai.Transcription
	err      error
}

func (f *fakeRequester) Transcribe(ctx context.Context, params openai.AudioTranscriptionNewParams) (*openai.Transcription, error) {
	f.callCount++
	f.lastParams = append(f.lastParams, params)
	step := f.steps[0]
	f.steps = f.steps[1:]
	return step.response, step.err
}

type statusError struct {
	code int
	msg  string
}

func (s statusError) Error() string {
	return s.msg
}

func (s statusError) StatusCode() int {
	return s.code
}

func TestProviderRetriesOnRetryableError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	audioPath := filepath.Join(dir, "input.m4a")
	if err := os.WriteFile(audioPath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write audio: %v", err)
	}

	requester := &fakeRequester{
		steps: []fakeStep{
			{err: statusError{code: 429, msg: "rate limited"}},
			{err: statusError{code: 503, msg: "service unavailable"}},
			{response: mustTranscription(`{"text":"hello","language":"ru","segments":[{"start":0,"end":1,"text":"hello"}]}`)},
		},
	}
	providerClient := newWithRequester("test-key", fsx.OS{}, zerolog.New(io.Discard), requester)

	response, err := providerClient.Transcribe(context.Background(), provider.Request{
		FilePath: audioPath,
		Model:    openai.AudioModelWhisper1,
		Language: "ru",
	})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}
	if requester.callCount != 3 {
		t.Fatalf("callCount = %d, want 3", requester.callCount)
	}
	if response.Transcript.Text != "hello" {
		t.Fatalf("text = %q", response.Transcript.Text)
	}
	if len(response.Transcript.Segments) != 1 {
		t.Fatalf("segments = %v", response.Transcript.Segments)
	}
}

func TestProviderUsesDiarizedRequestShape(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	audioPath := filepath.Join(dir, "input.m4a")
	if err := os.WriteFile(audioPath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write audio: %v", err)
	}

	requester := &fakeRequester{
		steps: []fakeStep{
			{response: mustTranscription(`{"text":"hello","language":"ru","segments":[{"start":0,"end":1,"speaker":"speaker_0","text":"hello"}]}`)},
		},
	}
	providerClient := newWithRequester("test-key", fsx.OS{}, zerolog.New(io.Discard), requester)

	response, err := providerClient.Transcribe(context.Background(), provider.Request{
		FilePath:        audioPath,
		Model:           diarizeModel,
		Language:        "ru",
		WantDiarization: true,
	})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}
	if len(response.Transcript.SpeakerSegments) != 1 {
		t.Fatalf("speakerSegments = %v", response.Transcript.SpeakerSegments)
	}

	params := requester.lastParams[0]
	if params.Model != diarizeModel {
		t.Fatalf("model = %s", params.Model)
	}
	if params.ResponseFormat != openai.AudioResponseFormat("diarized_json") {
		t.Fatalf("responseFormat = %s", params.ResponseFormat)
	}
	if string(params.ChunkingStrategy.OfAuto) != "auto" {
		t.Fatalf("chunking strategy was not auto")
	}
}

func mustTranscription(raw string) *openai.Transcription {
	var transcription openai.Transcription
	if err := json.Unmarshal([]byte(raw), &transcription); err != nil {
		panic(err)
	}
	return &transcription
}

func TestProviderPreflightRequiresKey(t *testing.T) {
	t.Parallel()

	client := newWithRequester("", fsx.OS{}, zerolog.New(io.Discard), &fakeRequester{})
	if err := client.Preflight(); err == nil || !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Fatalf("expected preflight error for missing key")
	}
}
