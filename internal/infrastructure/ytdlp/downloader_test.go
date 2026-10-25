package ytdlp

import (
	"slices"
	"testing"

	"downloader/internal/domain"
)

func TestBuildArgs_Audio(t *testing.T) {

	args := buildArgs(domain.DownloadRequest{URL: "https://a.com/v", SavePath: "/tmp", Format: domain.Audio})

	for _, want := range []string{"--no-playlist", "wa/w", "-x", "mp3"} {
		if !slices.Contains(args, want) {
			t.Errorf("args missing %q: %v", want, args)
		}
	}
	if n := len(args); args[n-2] != "--" || args[n-1] != "https://a.com/v" {
		t.Errorf("link must come last after --: %v", args)
	}
}

func TestBuildArgs_Video(t *testing.T) {

	args := buildArgs(domain.DownloadRequest{URL: "https://a.com/v", SavePath: "/tmp", Format: domain.Video})

	if slices.Contains(args, "-x") || !slices.Contains(args, "worst") {
		t.Errorf("video args = %v", args)
	}
}

func TestParseProgress(t *testing.T) {

	tests := []struct {
		line string
		want float64
		ok   bool
	}{
		{"dl-progress 45.5", 45.5, true},
		{"dl-progress 100.0", 100, true},
		{"dl-progress  7.1%", 7.1, true},
		{"dl-progress NA", 0, false},
		{"/tmp/a.mp3", 0, false},
	}

	for _, tt := range tests {
		got, ok := parseProgress(tt.line)
		if got != tt.want || ok != tt.ok {
			t.Errorf("parseProgress(%q) = %v, %v; want %v, %v", tt.line, got, ok, tt.want, tt.ok)
		}
	}
}
