package audio

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arykalin/whisper-cli/internal/platform/fsx"
)

type runCall struct {
	name string
	args []string
}

type fakeRunner struct {
	calls       []runCall
	lookPathErr map[string]error
	runFunc     func(ctx context.Context, name string, args ...string) ([]byte, []byte, error)
}

func (f *fakeRunner) LookPath(name string) (string, error) {
	if err, ok := f.lookPathErr[name]; ok {
		return "", err
	}
	return "/usr/bin/" + name, nil
}

func (f *fakeRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	f.calls = append(f.calls, runCall{
		name: name,
		args: append([]string(nil), args...),
	})
	if f.runFunc != nil {
		return f.runFunc(ctx, name, args...)
	}
	return nil, nil, nil
}

func TestCollectMediaFilesFiltersAndSorts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	for _, name := range []string{"b.MP3", "a.m4a", "c.WEBM", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	service := Service{FS: fsx.OS{}}
	files, err := service.CollectMediaFiles(dir)
	if err != nil {
		t.Fatalf("CollectMediaFiles returned error: %v", err)
	}

	want := []string{
		filepath.Join(dir, "a.m4a"),
		filepath.Join(dir, "b.MP3"),
		filepath.Join(dir, "c.WEBM"),
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

func TestPrepareInputSkipsConversionForM4A(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	input := filepath.Join(dir, "input.m4a")
	if err := os.WriteFile(input, []byte("x"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	workDir := filepath.Join(dir, "work")
	runner := &fakeRunner{}
	service := Service{FS: fsx.OS{}, Runner: runner}

	prepared, err := service.PrepareInput(context.Background(), input, workDir)
	if err != nil {
		t.Fatalf("PrepareInput returned error: %v", err)
	}

	if prepared.OriginalPath != input {
		t.Fatalf("OriginalPath = %s, want %s", prepared.OriginalPath, input)
	}
	if prepared.ChunkSourcePath != input {
		t.Fatalf("ChunkSourcePath = %s, want %s", prepared.ChunkSourcePath, input)
	}
	if prepared.Converted {
		t.Fatalf("Converted = true, want false")
	}
	if len(runner.calls) != 0 {
		t.Fatalf("unexpected runner calls: %#v", runner.calls)
	}
	if _, err := os.Stat(workDir); err != nil {
		t.Fatalf("expected workDir to exist: %v", err)
	}
}

func TestPrepareInputConvertsNonM4AInput(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp3")
	if err := os.WriteFile(input, []byte("x"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	workDir := filepath.Join(dir, "work")
	runner := &fakeRunner{}
	service := Service{FS: fsx.OS{}, Runner: runner}

	prepared, err := service.PrepareInput(context.Background(), input, workDir)
	if err != nil {
		t.Fatalf("PrepareInput returned error: %v", err)
	}

	wantOutput := filepath.Join(workDir, "source.m4a")
	if prepared.OriginalPath != input {
		t.Fatalf("OriginalPath = %s, want %s", prepared.OriginalPath, input)
	}
	if prepared.ChunkSourcePath != wantOutput {
		t.Fatalf("ChunkSourcePath = %s, want %s", prepared.ChunkSourcePath, wantOutput)
	}
	if !prepared.Converted {
		t.Fatalf("Converted = false, want true")
	}
	if len(runner.calls) != 1 {
		t.Fatalf("runner call count = %d, want 1", len(runner.calls))
	}
	if runner.calls[0].name != "ffmpeg" {
		t.Fatalf("command = %s, want ffmpeg", runner.calls[0].name)
	}
	wantArgs := []string{"-y", "-i", input, "-vn", "-c:a", "aac", wantOutput}
	if strings.Join(runner.calls[0].args, "\n") != strings.Join(wantArgs, "\n") {
		t.Fatalf("args = %v, want %v", runner.calls[0].args, wantArgs)
	}
}

func TestPrepareInputPreservesFFmpegStderr(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")
	if err := os.WriteFile(input, []byte("x"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	runner := &fakeRunner{
		runFunc: func(context.Context, string, ...string) ([]byte, []byte, error) {
			return nil, []byte("bad codec"), errors.New("exit status 1")
		},
	}
	service := Service{FS: fsx.OS{}, Runner: runner}

	_, err := service.PrepareInput(context.Background(), input, filepath.Join(dir, "work"))
	if err == nil {
		t.Fatalf("expected PrepareInput error")
	}
	if !strings.Contains(err.Error(), "convert input to m4a") || !strings.Contains(err.Error(), "bad codec") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareChunksWritesChunksInWorkDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	workDir := filepath.Join(dir, "work")
	runner := &fakeRunner{
		runFunc: func(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
			switch name {
			case "ffmpeg":
				for _, file := range []string{
					filepath.Join(workDir, "chunk_000.m4a"),
					filepath.Join(workDir, "chunk_001.m4a"),
				} {
					if err := os.WriteFile(file, []byte("chunk"), 0o644); err != nil {
						t.Fatalf("write chunk %s: %v", file, err)
					}
				}
				return nil, nil, nil
			case "ffprobe":
				switch filepath.Base(args[len(args)-1]) {
				case "chunk_000.m4a":
					return []byte("1.5\n"), nil, nil
				case "chunk_001.m4a":
					return []byte("2.25\n"), nil, nil
				default:
					t.Fatalf("unexpected ffprobe target: %v", args)
				}
			default:
				t.Fatalf("unexpected command: %s", name)
			}
			return nil, nil, nil
		},
	}
	service := Service{FS: fsx.OS{}, Runner: runner}

	chunks, err := service.PrepareChunks(context.Background(), filepath.Join(workDir, "source.m4a"), workDir, 600)
	if err != nil {
		t.Fatalf("PrepareChunks returned error: %v", err)
	}

	if len(chunks) != 2 {
		t.Fatalf("chunk count = %d, want 2", len(chunks))
	}
	if chunks[0].Path != filepath.Join(workDir, "chunk_000.m4a") {
		t.Fatalf("chunk[0].Path = %s", chunks[0].Path)
	}
	if chunks[0].Offset != 0 || chunks[0].Duration != 1.5 {
		t.Fatalf("chunk[0] = %#v", chunks[0])
	}
	if chunks[1].Path != filepath.Join(workDir, "chunk_001.m4a") {
		t.Fatalf("chunk[1].Path = %s", chunks[1].Path)
	}
	if chunks[1].Offset != 1.5 || chunks[1].Duration != 2.25 {
		t.Fatalf("chunk[1] = %#v", chunks[1])
	}

	if len(runner.calls) != 3 {
		t.Fatalf("runner call count = %d, want 3", len(runner.calls))
	}
	wantSplitArgs := []string{
		"-y",
		"-i", filepath.Join(workDir, "source.m4a"),
		"-f", "segment",
		"-segment_time", "600",
		"-c", "copy",
		filepath.Join(workDir, "chunk_%03d.m4a"),
	}
	if strings.Join(runner.calls[0].args, "\n") != strings.Join(wantSplitArgs, "\n") {
		t.Fatalf("split args = %v, want %v", runner.calls[0].args, wantSplitArgs)
	}
}

func TestPrepareChunksPreservesFFmpegStderr(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	runner := &fakeRunner{
		runFunc: func(context.Context, string, ...string) ([]byte, []byte, error) {
			return nil, []byte("split failed"), errors.New("exit status 1")
		},
	}
	service := Service{FS: fsx.OS{}, Runner: runner}

	_, err := service.PrepareChunks(context.Background(), filepath.Join(dir, "source.m4a"), filepath.Join(dir, "work"), 600)
	if err == nil {
		t.Fatalf("expected PrepareChunks error")
	}
	if !strings.Contains(err.Error(), "split audio") || !strings.Contains(err.Error(), "split failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareChunksPreservesFFprobeStderr(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	workDir := filepath.Join(dir, "work")
	runner := &fakeRunner{
		runFunc: func(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
			switch name {
			case "ffmpeg":
				chunkFile := filepath.Join(workDir, "chunk_000.m4a")
				if err := os.WriteFile(chunkFile, []byte("chunk"), 0o644); err != nil {
					t.Fatalf("write chunk: %v", err)
				}
				return nil, nil, nil
			case "ffprobe":
				return nil, []byte("probe failed"), errors.New("exit status 1")
			default:
				t.Fatalf("unexpected command: %s", name)
			}
			return nil, nil, nil
		},
	}
	service := Service{FS: fsx.OS{}, Runner: runner}

	_, err := service.PrepareChunks(context.Background(), filepath.Join(workDir, "source.m4a"), workDir, 600)
	if err == nil {
		t.Fatalf("expected PrepareChunks error")
	}
	if !strings.Contains(err.Error(), "duration for") || !strings.Contains(err.Error(), "probe failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}
