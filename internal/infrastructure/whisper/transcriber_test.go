package whisper

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"downloader/internal/domain"
)

func TestParseProgress(t *testing.T) {

	tests := []struct {
		line string
		want int
		ok   bool
	}{
		{"whisper_print_progress_callback: progress =   5%", 5, true},
		{"whisper_print_progress_callback: progress = 100%", 100, true},
		{"whisper_init_from_file_with_params_no_state: loading model", 0, false},
		{"progress = abc%", 0, false},
	}

	for _, tt := range tests {
		got, ok := parseProgress(tt.line)
		if got != tt.want || ok != tt.ok {
			t.Errorf("parseProgress(%q) = %d, %v; want %d, %v", tt.line, got, ok, tt.want, tt.ok)
		}
	}
}

func TestBuildArgs(t *testing.T) {

	args := buildArgs(domain.TranscribeOptions{ModelPath: "/m.bin", Language: domain.LangPersian}, "/a.wav", "/a", 4)
	if i := slices.Index(args, "-l"); i < 0 || args[i+1] != "fa" {
		t.Errorf("language not passed: %v", args)
	}
	if slices.Contains(args, "-tr") {
		t.Errorf("-tr without Translate: %v", args)
	}

	args = buildArgs(domain.TranscribeOptions{ModelPath: "/m.bin", Translate: true}, "/a.wav", "/a.en", 4)
	if !slices.Contains(args, "-tr") {
		t.Errorf("Translate should add -tr: %v", args)
	}
	if i := slices.Index(args, "-l"); args[i+1] != "auto" {
		t.Errorf("empty language should be auto: %v", args)
	}
}

func TestOutputBase(t *testing.T) {

	if got := outputBase("/x/talk.mp3", domain.TranscribeOptions{}); got != "/x/talk" {
		t.Errorf("outputBase = %q", got)
	}
	if got := outputBase("/x/talk.mp3", domain.TranscribeOptions{Translate: true}); got != "/x/talk.en" {
		t.Errorf("outputBase with Translate = %q", got)
	}
}

func TestListModels(t *testing.T) {

	dir := t.TempDir()
	for _, f := range []string{"ggml-small.bin", "ggml-medium-q5_0.bin", "ggml-silero-v5.bin", "notes.txt"} {
		os.WriteFile(filepath.Join(dir, f), nil, 0o644)
	}

	var names []string
	for _, m := range ListModels(dir) {
		names = append(names, m.Name)
	}

	if want := []string{"medium-q5_0", "small"}; !slices.Equal(names, want) {
		t.Errorf("models = %v, want %v", names, want)
	}
}
