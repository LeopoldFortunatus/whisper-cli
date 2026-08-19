package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arykalin/whisper-cli/internal/domain"
	"github.com/arykalin/whisper-cli/internal/platform/fsx"
)

func TestWriteArtifactsWritesExpectedFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	publicDir := filepath.Join(dir, "public")
	transcript := domain.Transcript{
		Provider: domain.ProviderOpenAI,
		Model:    "whisper-1",
		Language: "ru",
		Text:     "first\nsecond",
		Segments: []domain.Segment{
			{Start: 0, End: 1.2, Text: "first"},
			{Start: 2, End: 3.5, Text: "second"},
		},
		SpeakerSegments: []domain.SpeakerSegment{
			{Start: 0, End: 1.2, Speaker: "speaker_0", Text: "first"},
		},
	}
	outputs := domain.ArtifactSet{
		domain.ArtifactTimestamps: true,
		domain.ArtifactSRT:        true,
		domain.ArtifactVTT:        true,
		domain.ArtifactDiarized:   true,
		domain.ArtifactRaw:        true,
	}

	err := WriteArtifacts(fsx.OS{}, WriteRequest{
		Layout: Layout{
			CacheDir:  cacheDir,
			PublicDir: publicDir,
			Stem:      "lecture",
		},
		Transcript: transcript,
		Outputs:    outputs,
		RawArtifacts: [][]byte{
			[]byte(`{"chunk":0}`),
			[]byte(`{"chunk":1}`),
		},
	})
	if err != nil {
		t.Fatalf("WriteArtifacts returned error: %v", err)
	}

	for _, name := range []string{
		"lecture.txt",
		"lecture.timestamps.txt",
		"lecture.srt",
		"lecture.vtt",
		"lecture.diarized.json",
		"lecture.raw.json",
	} {
		if _, err := os.Stat(filepath.Join(publicDir, name)); err != nil {
			t.Fatalf("expected %s to exist: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "transcript.json")); err != nil {
		t.Fatalf("expected cached transcript.json to exist: %v", err)
	}

	timestamps, err := os.ReadFile(filepath.Join(publicDir, "lecture.timestamps.txt"))
	if err != nil {
		t.Fatalf("read timestamps: %v", err)
	}
	if !strings.Contains(string(timestamps), "[00:00:00 - 00:00:01] first") {
		t.Fatalf("unexpected timestamps content: %s", string(timestamps))
	}

	vtt, err := os.ReadFile(filepath.Join(publicDir, "lecture.vtt"))
	if err != nil {
		t.Fatalf("read vtt: %v", err)
	}
	if !strings.HasPrefix(string(vtt), "WEBVTT") {
		t.Fatalf("unexpected vtt content: %s", string(vtt))
	}
}

func TestWriteArtifactsDefaultWritesOnlyPublicTextAndCachedJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	publicDir := filepath.Join(dir, "public")
	if err := os.MkdirAll(publicDir, 0o755); err != nil {
		t.Fatalf("mkdir public dir: %v", err)
	}
	publicTranscript := filepath.Join(publicDir, "meeting.notes.txt")
	if err := os.WriteFile(publicTranscript, []byte("old\n"), 0o644); err != nil {
		t.Fatalf("write old transcript: %v", err)
	}

	err := WriteArtifacts(fsx.OS{}, WriteRequest{
		Layout: Layout{
			CacheDir:  cacheDir,
			PublicDir: publicDir,
			Stem:      "meeting.notes",
		},
		Transcript: domain.Transcript{
			Provider: domain.ProviderOpenAI,
			Model:    "gpt-4o-transcribe",
			Text:     "new transcript",
		},
		Outputs: domain.DefaultArtifacts(),
	})
	if err != nil {
		t.Fatalf("WriteArtifacts returned error: %v", err)
	}

	entries, err := os.ReadDir(publicDir)
	if err != nil {
		t.Fatalf("read public dir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "meeting.notes.txt" {
		t.Fatalf("public entries = %v, want only meeting.notes.txt", entries)
	}
	text, err := os.ReadFile(publicTranscript)
	if err != nil {
		t.Fatalf("read public transcript: %v", err)
	}
	if string(text) != "new transcript\n" {
		t.Fatalf("public transcript = %q", string(text))
	}
	info, err := os.Stat(filepath.Join(cacheDir, "transcript.json"))
	if err != nil {
		t.Fatalf("stat cached transcript.json: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("cached transcript mode = %o, want 600", info.Mode().Perm())
	}
}

func TestPublicArtifactPathsUsesStemPrefixedNames(t *testing.T) {
	t.Parallel()

	layout := Layout{
		CacheDir:  "/cache/job",
		PublicDir: "/output",
		Stem:      "lecture",
	}
	outputs := domain.ArtifactSet{
		domain.ArtifactTimestamps: true,
		domain.ArtifactSRT:        true,
		domain.ArtifactVTT:        true,
		domain.ArtifactDiarized:   true,
		domain.ArtifactRaw:        true,
	}

	paths := PublicArtifactPaths(layout, outputs)
	want := []string{
		"/output/lecture.txt",
		"/output/lecture.timestamps.txt",
		"/output/lecture.srt",
		"/output/lecture.vtt",
		"/output/lecture.diarized.json",
		"/output/lecture.raw.json",
	}
	if strings.Join(paths, "\n") != strings.Join(want, "\n") {
		t.Fatalf("paths = %v, want %v", paths, want)
	}
}
