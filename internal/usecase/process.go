package usecase

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"downloader/internal/domain"
)

var (
	ErrNoSource    = errors.New("add a link or a file")
	ErrTwoSources  = errors.New("a job takes a link or a file, not both")
	ErrInvalidURL  = errors.New("link must start with http:// or https://")
	ErrNoSavePath  = errors.New("choose a folder to save to")
	ErrNothingToDo = errors.New("a file job must transcribe")
	ErrNoModel     = errors.New("no whisper model selected")
)

type Stage int

const (
	StageDownload Stage = iota
	StageTranscribe
)

// Unknown is the percent value when progress is not known yet.
const Unknown = -1.0

type Downloader interface {
	// Download returns the path of the downloaded file. percent is 0-100.
	Download(ctx context.Context, req domain.DownloadRequest, onProgress func(percent float64)) (string, error)
}

type Transcriber interface {
	// Transcribe returns the path of the text file. percent is 0-100.
	Transcribe(ctx context.Context, mediaPath string, opts domain.TranscribeOptions, onProgress func(percent float64)) (string, error)
}

type ProcessUseCase struct {
	downloader  Downloader
	transcriber Transcriber
}

func NewProcessUseCase(d Downloader, t Transcriber) *ProcessUseCase {
	return &ProcessUseCase{downloader: d, transcriber: t}
}

// Run downloads the link (if any), then transcribes (if asked).
func (uc *ProcessUseCase) Run(ctx context.Context, job domain.Job, onProgress func(Stage, float64)) (domain.Result, error) {

	if err := Validate(job); err != nil {
		return domain.Result{}, err
	}
	if onProgress == nil {
		onProgress = func(Stage, float64) {}
	}

	res := domain.Result{MediaPath: job.FilePath}

	if job.URL != "" {
		onProgress(StageDownload, Unknown)

		req := domain.DownloadRequest{URL: job.URL, SavePath: job.SavePath, Format: job.Format}
		path, err := uc.downloader.Download(ctx, req, func(p float64) { onProgress(StageDownload, p) })
		if err != nil {
			return res, err
		}
		res.MediaPath = path
	}

	if !job.Transcribe {
		return res, nil
	}

	onProgress(StageTranscribe, Unknown)

	txt, err := uc.transcriber.Transcribe(ctx, res.MediaPath, job.Options, func(p float64) { onProgress(StageTranscribe, p) })
	if err != nil {
		return res, err
	}
	res.TextPath = txt

	return res, nil
}

func Validate(job domain.Job) error {
	switch {
	case job.URL == "" && job.FilePath == "":
		return ErrNoSource
	case job.URL != "" && job.FilePath != "":
		return ErrTwoSources
	case job.URL != "" && !validURL(job.URL):
		return ErrInvalidURL
	case job.URL != "" && job.SavePath == "":
		return ErrNoSavePath
	case job.FilePath != "" && !job.Transcribe:
		return ErrNothingToDo
	case job.Transcribe && job.Options.ModelPath == "":
		return ErrNoModel
	}
	return nil
}

// ParseLinks splits text into links, one per line.
// Empty lines and repeated links are skipped; lines that are not links go to invalid.
func ParseLinks(text string) (links, invalid []string) {

	seen := map[string]bool{}

	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || seen[line] {
			continue
		}
		seen[line] = true

		if validURL(line) {
			links = append(links, line)
		} else {
			invalid = append(invalid, line)
		}
	}
	return links, invalid
}

func validURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}
