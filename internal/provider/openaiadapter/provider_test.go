package openaiadapter

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arykalin/whisper-cli/internal/platform/fsx"
	"github.com/arykalin/whisper-cli/internal/provider"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
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
			{response: mustTranscription(`{"text":"hello","languages":[{"code":"ru"}]}`)},
		},
	}
	providerClient := newWithRequester("test-key", fsx.OS{}, zerolog.New(io.Discard), requester)

	response, err := providerClient.Transcribe(context.Background(), provider.Request{
		FilePath: audioPath,
		Model:    modelGPTTranscribe,
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
	if response.Transcript.Language != "ru" {
		t.Fatalf("language = %q, want ru", response.Transcript.Language)
	}
	if string(response.Raw) != `{"text":"hello","languages":[{"code":"ru"}]}` {
		t.Fatalf("raw = %q", response.Raw)
	}
}

func TestProviderUsesGPTTranscribeRequestShape(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	audioPath := filepath.Join(dir, "input.m4a")
	if err := os.WriteFile(audioPath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write audio: %v", err)
	}

	requester := &fakeRequester{
		steps: []fakeStep{
			{response: mustTranscription(`{"text":"hello"}`)},
		},
	}
	providerClient := newWithRequester("test-key", fsx.OS{}, zerolog.New(io.Discard), requester)

	_, err := providerClient.Transcribe(context.Background(), provider.Request{
		FilePath: audioPath,
		Model:    modelGPTTranscribe,
		Language: "ru",
		Prompt:   "technical vocabulary",
	})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}

	params := requester.lastParams[0]
	if params.Model != modelGPTTranscribe {
		t.Fatalf("model = %s", params.Model)
	}
	if params.ResponseFormat != openai.AudioResponseFormatJSON {
		t.Fatalf("responseFormat = %s", params.ResponseFormat)
	}
	if !param.IsOmitted(params.Language) {
		t.Fatalf("singular language must be omitted: %#v", params.Language)
	}
	if !params.Prompt.Valid() || params.Prompt.Value != "technical vocabulary" {
		t.Fatalf("prompt = %#v", params.Prompt)
	}
	if got := params.ExtraFields()["languages[]"]; got != "ru" {
		t.Fatalf("languages[] = %#v, want ru", got)
	}
}

func TestGPTTranscribeLanguageSerializesAsPluralMultipartField(t *testing.T) {
	t.Parallel()

	params := openai.AudioTranscriptionNewParams{
		File:           strings.NewReader("audio"),
		Model:          modelGPTTranscribe,
		ResponseFormat: openai.AudioResponseFormatJSON,
	}
	params.SetExtraFields(map[string]any{"languages[]": "ru"})

	data, contentType, err := params.MarshalMultipart()
	if err != nil {
		t.Fatalf("MarshalMultipart returned error: %v", err)
	}
	mediaType, values, err := readMultipartFields(data, contentType)
	if err != nil {
		t.Fatalf("read multipart fields: %v", err)
	}
	if mediaType != "multipart/form-data" {
		t.Fatalf("media type = %q", mediaType)
	}
	if values["languages[]"] != "ru" {
		t.Fatalf("languages[] = %q, want ru", values["languages[]"])
	}
	if _, ok := values["language"]; ok {
		t.Fatalf("singular language field must be absent: %#v", values)
	}
}

func readMultipartFields(data []byte, contentType string) (string, map[string]string, error) {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", nil, err
	}

	reader := multipart.NewReader(strings.NewReader(string(data)), params["boundary"])
	values := make(map[string]string)
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", nil, err
		}
		if part.FileName() != "" {
			continue
		}
		value, err := io.ReadAll(part)
		if err != nil {
			return "", nil, err
		}
		values[part.FormName()] = string(value)
	}
	return mediaType, values, nil
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
	if err := client.Preflight(); err == nil || !strings.Contains(err.Error(), "OpenAI API key is not configured") {
		t.Fatalf("expected preflight error for missing key")
	}
}

func TestProviderSupportsOnlyGPTTranscribe(t *testing.T) {
	t.Parallel()

	client := New("test-key", fsx.OS{}, zerolog.New(io.Discard))
	models := client.SupportedModels()
	want := []string{modelGPTTranscribe}
	if strings.Join(models, ",") != strings.Join(want, ",") {
		t.Fatalf("supported models = %v, want %v", models, want)
	}
}

func TestProviderRejectsRemovedOpenAIModelsBeforeRequest(t *testing.T) {
	t.Parallel()

	for _, model := range []string{
		"whisper-1",
		"gpt-4o-transcribe",
		"gpt-4o-mini-transcribe",
		"gpt-4o-transcribe-diarize",
	} {
		model := model
		t.Run(model, func(t *testing.T) {
			t.Parallel()

			requester := &fakeRequester{}
			client := newWithRequester("test-key", fsx.OS{}, zerolog.New(io.Discard), requester)
			_, err := client.Transcribe(context.Background(), provider.Request{Model: model})
			if err == nil || !strings.Contains(err.Error(), "is not supported by provider openai") {
				t.Fatalf("Transcribe error = %v", err)
			}
			if requester.callCount != 0 {
				t.Fatalf("request count = %d, want 0", requester.callCount)
			}
		})
	}
}
