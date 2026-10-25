package main

import (
	"os"

	"downloader/internal/config"
	"downloader/internal/domain"
	"downloader/internal/infrastructure/macos"
	"downloader/internal/infrastructure/whisper"
	"downloader/internal/infrastructure/ytdlp"
	"downloader/internal/interface/ui"
	"downloader/internal/usecase"

	"fyne.io/fyne/v2/app"
)

func main() {
	config.FixPath()
	cfg := config.Default()
	os.MkdirAll(cfg.ModelsDir, 0o755)

	a := app.NewWithID("io.mohsen.downloader")
	a.Settings().SetTheme(ui.NewTheme())
	w := a.NewWindow("Downloader")

	// Dependency Injection
	process := usecase.NewProcessUseCase(
		ytdlp.NewYtDlpDownloader(),
		whisper.NewWhisperTranscriber(cfg.Threads),
	)

	missing := cfg.MissingTools()
	view := ui.New(a, w, ui.Deps{
		Process:      process,
		ListModels:   func() []domain.Model { return whisper.ListModels(cfg.ModelsDir) },
		ModelsDir:    cfg.ModelsDir,
		MissingTools: missing,
		InstallHint:  config.InstallCommand(missing),
		Reveal:       macos.Reveal,
		Open:         macos.Open,
	})

	w.SetContent(view.Content())
	w.Resize(ui.WindowSize)
	w.ShowAndRun()

	// the window is closed: stop any running yt-dlp / whisper before exit
	view.Shutdown()
}
