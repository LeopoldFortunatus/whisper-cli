package audio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arykalin/whisper-cli/internal/platform/fsx"
)

func TestCollectAudioFilesFiltersAndSorts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	for _, name := range []string{"b.MP3", "a.m4a", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	service := Service{FS: fsx.OS{}}
	files, err := service.CollectAudioFiles(dir)
	if err != nil {
		t.Fatalf("CollectAudioFiles returned error: %v", err)
	}

	want := []string{
		filepath.Join(dir, "a.m4a"),
		filepath.Join(dir, "b.MP3"),
	}
	if len(files) != len(want) {
		t.Fatalf("files = %v, want %v", files, want)
	}
	for idx := range want {
		if files[idx] != want[idx] {
			t.Fatalf("files[%d] = %s, want %s", idx, files[idx], want[idx])
		}
	}
}
