package ui

import (
	"context"
	"errors"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"downloader/internal/domain"
	"downloader/internal/usecase"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

// TestScreenshots draws the window to PNG files, to look at the design
// without opening the app:
//
//	UI_SHOTS=/tmp/shots go test ./internal/interface/ui -run Screenshots
func TestScreenshots(t *testing.T) {

	dir := os.Getenv("UI_SHOTS")
	if dir == "" {
		t.Skip("set UI_SHOTS=<dir> to render screenshots")
	}

	for _, variant := range []fyne.ThemeVariant{theme.VariantDark, theme.VariantLight} {

		a := test.NewTempApp(t)
		a.Settings().SetTheme(forceVariant{appTheme(), variant})

		w := a.NewWindow("Downloader")
		v := New(a, w, Deps{
			Process:    usecase.NewProcessUseCase(nopDownloader{}, nopTranscriber{}),
			ListModels: func() []domain.Model { return []domain.Model{{Name: "medium-q5_0", Path: "/m.bin"}} },
			ModelsDir:  t.TempDir(),
			Reveal:     func(string) error { return nil },
			Open:       func(string) error { return nil },
		})
		w.SetContent(v.Content())
		w.Resize(WindowSize)

		name := "dark"
		if variant == theme.VariantLight {
			name = "light"
		}
		save(t, w, filepath.Join(dir, name+"-empty.png"))

		v.links.SetText("https://www.youtube.com/watch?v=jNQXAC9IVRw\nhttps://soundcloud.com/a/talk")
		for _, st := range sampleJobs() {
			v.jobs.update(st)
		}
		save(t, w, filepath.Join(dir, name+"-jobs.png"))
	}
}

func sampleJobs() []usecase.JobStatus {
	link := domain.Job{URL: "https://www.youtube.com/watch?v=jNQXAC9IVRw", Transcribe: true}
	file := domain.Job{FilePath: "/Users/me/Music/podcast-episode-12.mp3", Transcribe: true}
	return []usecase.JobStatus{
		{ID: 1, Job: file, State: usecase.Done, Percent: 100,
			Result: domain.Result{MediaPath: "/Users/me/Music/podcast-episode-12.mp3", TextPath: "/Users/me/Music/podcast-episode-12.txt"}},
		{ID: 2, Job: link, State: usecase.Failed, Percent: -1,
			Err: errors.New("yt-dlp: exit status 1\nERROR: [youtube] jNQXAC9IVRw: Sign in to confirm you’re not a bot. Use --cookies-from-browser")},
		{ID: 3, Job: file, State: usecase.Transcribing, Percent: 45},
		{ID: 4, Job: domain.Job{URL: "https://soundcloud.com/a/talk"}, State: usecase.Downloading, Percent: 72},
		{ID: 5, Job: link, State: usecase.Waiting, Percent: -1},
		{ID: 6, Job: link, State: usecase.Canceled, Percent: -1},
	}
}

func save(t *testing.T, w fyne.Window, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, w.Canvas().Capture()); err != nil {
		t.Fatal(err)
	}
}

type nopDownloader struct{}

func (nopDownloader) Download(context.Context, domain.DownloadRequest, func(float64)) (string, error) {
	return "", nil
}

type nopTranscriber struct{}

func (nopTranscriber) Transcribe(context.Context, string, domain.TranscribeOptions, func(float64)) (string, error) {
	return "", nil
}
