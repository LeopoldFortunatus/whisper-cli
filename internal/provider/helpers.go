package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/arykalin/whisper-cli/internal/domain"
	"github.com/rs/zerolog"
)

type responsePayload struct {
	Text     string           `json:"text"`
	Language string           `json:"language"`
	Segments []segmentPayload `json:"segments"`
}

type segmentPayload struct {
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Text    string  `json:"text"`
	Speaker string  `json:"speaker"`
}

type statusCoder interface {
	StatusCode() int
}

func ParseOpenAICompatibleTranscript(providerName domain.Provider, model string, raw []byte, fallbackText string) domain.Transcript {
	transcript := domain.Transcript{
		Provider: providerName,
		Model:    model,
		Text:     strings.TrimSpace(fallbackText),
	}

	if len(raw) == 0 {
		return transcript
	}

	var payload responsePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return transcript
	}

	if strings.TrimSpace(payload.Text) != "" {
		transcript.Text = strings.TrimSpace(payload.Text)
	}
	transcript.Language = strings.TrimSpace(payload.Language)

	for _, segment := range payload.Segments {
		if strings.TrimSpace(segment.Text) == "" {
			continue
		}
		transcript.Segments = append(transcript.Segments, domain.Segment{
			Start: segment.Start,
			End:   segment.End,
			Text:  strings.TrimSpace(segment.Text),
		})
		if strings.TrimSpace(segment.Speaker) != "" {
			transcript.SpeakerSegments = append(transcript.SpeakerSegments, domain.SpeakerSegment{
				Start:   segment.Start,
				End:     segment.End,
				Speaker: strings.TrimSpace(segment.Speaker),
				Text:    strings.TrimSpace(segment.Text),
			})
		}
	}

	if transcript.Text == "" {
		transcript.Text = transcript.PlainText()
	}

	return transcript
}

func Retry(ctx context.Context, logger zerolog.Logger, label string, fn func() error) error {
	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}
		if !shouldRetry(err) || attempt == 3 {
			return err
		}

		logger.Warn().Err(err).Int("attempt", attempt).Str("provider", label).Msg("transcription request failed, retrying")

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt) * 100 * time.Millisecond):
		}
	}
	return err
}

func shouldRetry(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var withStatus statusCoder
	if errors.As(err, &withStatus) {
		code := withStatus.StatusCode()
		return code == 429 || code >= 500
	}

	msg := strings.ToLower(err.Error())
	for _, marker := range []string{
		"429",
		"500",
		"502",
		"503",
		"504",
		"timeout",
		"temporarily unavailable",
		"connection reset",
	} {
		if strings.Contains(msg, marker) {
			return true
		}
	}

	return false
}

func MarshalRawArray(items [][]byte) ([]byte, error) {
	if len(items) == 0 {
		return nil, nil
	}

	rawItems := make([]json.RawMessage, 0, len(items))
	for _, item := range items {
		if len(item) == 0 {
			continue
		}
		if json.Valid(item) {
			rawItems = append(rawItems, json.RawMessage(item))
			continue
		}

		encoded, err := json.Marshal(map[string]string{"raw": string(item)})
		if err != nil {
			return nil, fmt.Errorf("marshal raw artifact: %w", err)
		}
		rawItems = append(rawItems, encoded)
	}

	return json.MarshalIndent(rawItems, "", "  ")
}
