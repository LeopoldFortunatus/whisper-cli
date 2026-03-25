package domain

import (
	"fmt"
	"slices"
	"strings"
)

type Provider string

const (
	ProviderOpenAI     Provider = "openai"
	ProviderGroq       Provider = "groq"
	ProviderOpenRouter Provider = "openrouter"
)

type ArtifactKind string

const (
	ArtifactTimestamps ArtifactKind = "timestamps"
	ArtifactSRT        ArtifactKind = "srt"
	ArtifactVTT        ArtifactKind = "vtt"
	ArtifactDiarized   ArtifactKind = "diarized"
	ArtifactRaw        ArtifactKind = "raw"
)

var knownArtifacts = map[ArtifactKind]struct{}{
	ArtifactTimestamps: {},
	ArtifactSRT:        {},
	ArtifactVTT:        {},
	ArtifactDiarized:   {},
	ArtifactRaw:        {},
}

type ArtifactSet map[ArtifactKind]bool

func DefaultArtifacts() ArtifactSet {
	return ArtifactSet{
		ArtifactTimestamps: true,
	}
}

func ParseArtifactSet(value string) (ArtifactSet, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.EqualFold(trimmed, "none") {
		return ArtifactSet{}, nil
	}

	result := ArtifactSet{}
	for _, raw := range strings.Split(trimmed, ",") {
		name := ArtifactKind(strings.ToLower(strings.TrimSpace(raw)))
		if name == "" {
			continue
		}
		if _, ok := knownArtifacts[name]; !ok {
			return nil, fmt.Errorf("unsupported output artifact %q", raw)
		}
		result[name] = true
	}
	return result, nil
}

func (a ArtifactSet) Enabled(kind ArtifactKind) bool {
	return a[kind]
}

func (a ArtifactSet) Sorted() []ArtifactKind {
	items := make([]ArtifactKind, 0, len(a))
	for kind, enabled := range a {
		if enabled {
			items = append(items, kind)
		}
	}
	slices.Sort(items)
	return items
}

type Capabilities struct {
	SupportsPrompt            bool
	SupportsSegmentTimestamps bool
	SupportsWordTimestamps    bool
	SupportsSRT               bool
	SupportsVTT               bool
	SupportsDiarization       bool
}

type Transcript struct {
	Provider        Provider         `json:"provider"`
	Model           string           `json:"model"`
	Source          string           `json:"source,omitempty"`
	Language        string           `json:"language,omitempty"`
	Text            string           `json:"text"`
	Segments        []Segment        `json:"segments,omitempty"`
	SpeakerSegments []SpeakerSegment `json:"speaker_segments,omitempty"`
}

type Segment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

type SpeakerSegment struct {
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Speaker string  `json:"speaker"`
	Text    string  `json:"text"`
}

func ShiftSegments(src []Segment, offset float64) []Segment {
	if len(src) == 0 {
		return nil
	}

	shifted := make([]Segment, 0, len(src))
	for _, segment := range src {
		shifted = append(shifted, Segment{
			Start: segment.Start + offset,
			End:   segment.End + offset,
			Text:  segment.Text,
		})
	}
	return shifted
}

func ShiftSpeakerSegments(src []SpeakerSegment, offset float64) []SpeakerSegment {
	if len(src) == 0 {
		return nil
	}

	shifted := make([]SpeakerSegment, 0, len(src))
	for _, segment := range src {
		shifted = append(shifted, SpeakerSegment{
			Start:   segment.Start + offset,
			End:     segment.End + offset,
			Speaker: segment.Speaker,
			Text:    segment.Text,
		})
	}
	return shifted
}

func (t Transcript) PlainText() string {
	if strings.TrimSpace(t.Text) != "" {
		return strings.TrimSpace(t.Text)
	}

	if len(t.SpeakerSegments) > 0 {
		var parts []string
		for _, segment := range t.SpeakerSegments {
			if text := strings.TrimSpace(segment.Text); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	}

	if len(t.Segments) > 0 {
		var parts []string
		for _, segment := range t.Segments {
			if text := strings.TrimSpace(segment.Text); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	}

	return ""
}
