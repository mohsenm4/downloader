package usecase

import (
	"context"
	"errors"
	"slices"
	"testing"

	"downloader/internal/domain"
)

func TestValidate(t *testing.T) {

	tests := []struct {
		name string
		job  domain.Job
		want error
	}{
		{"no source", domain.Job{}, ErrNoSource},
		{"two sources", domain.Job{URL: "https://a.com", FilePath: "/a.mp3"}, ErrTwoSources},
		{"bad link", domain.Job{URL: "a.com/x", SavePath: "/tmp"}, ErrInvalidURL},
		{"no folder", domain.Job{URL: "https://a.com"}, ErrNoSavePath},
		{"file without transcribe", domain.Job{FilePath: "/a.mp3"}, ErrNothingToDo},
		{"no model", domain.Job{FilePath: "/a.mp3", Transcribe: true}, ErrNoModel},
		{"link ok", domain.Job{URL: "https://a.com", SavePath: "/tmp"}, nil},
		{"file ok", domain.Job{FilePath: "/a.mp3", Transcribe: true, Options: opts}, nil},
	}

	for _, tt := range tests {
		if got := Validate(tt.job); !errors.Is(got, tt.want) {
			t.Errorf("%s: Validate() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestRun_LinkThenTranscribe(t *testing.T) {

	d, tr := &FakeDownloader{}, &FakeTranscriber{}
	uc := NewProcessUseCase(d, tr)

	var stages []Stage
	res, err := uc.Run(context.Background(),
		domain.Job{URL: "https://a.com", SavePath: "/tmp", Format: domain.Audio, Transcribe: true, Options: opts},
		func(s Stage, p float64) { stages = append(stages, s) })

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.MediaPath != "/tmp/video.mp3" || res.TextPath != "/tmp/video.mp3.txt" {
		t.Errorf("result = %+v", res)
	}
	if !slices.Equal(tr.Paths, []string{"/tmp/video.mp3"}) {
		t.Errorf("transcriber got %v, want the downloaded file", tr.Paths)
	}
	want := []Stage{StageDownload, StageDownload, StageTranscribe, StageTranscribe}
	if !slices.Equal(stages, want) {
		t.Errorf("stages = %v, want %v", stages, want)
	}
}

func TestRun_FileSkipsDownload(t *testing.T) {

	d, tr := &FakeDownloader{}, &FakeTranscriber{}
	uc := NewProcessUseCase(d, tr)

	res, err := uc.Run(context.Background(), domain.Job{FilePath: "/a.mp3", Transcribe: true, Options: opts}, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(d.Calls) != 0 {
		t.Errorf("downloader should not be called")
	}
	if res.TextPath != "/a.mp3.txt" {
		t.Errorf("TextPath = %q", res.TextPath)
	}
}

func TestRun_DownloadErrorStops(t *testing.T) {

	d, tr := &FakeDownloader{Err: errors.New("boom")}, &FakeTranscriber{}
	uc := NewProcessUseCase(d, tr)

	_, err := uc.Run(context.Background(),
		domain.Job{URL: "https://a.com", SavePath: "/tmp", Transcribe: true, Options: opts}, nil)

	if err == nil || len(tr.Paths) != 0 {
		t.Errorf("expected error and no transcribe, got err=%v transcribe calls=%d", err, len(tr.Paths))
	}
}

func TestParseLinks(t *testing.T) {

	links, invalid := ParseLinks(" https://a.com/1 \n\nhttps://a.com/1\nnot a link\nhttp://b.com\n")

	if !slices.Equal(links, []string{"https://a.com/1", "http://b.com"}) {
		t.Errorf("links = %v", links)
	}
	if !slices.Equal(invalid, []string{"not a link"}) {
		t.Errorf("invalid = %v", invalid)
	}
}
