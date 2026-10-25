package ytdlp

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"strings"

	"downloader/internal/domain"
	"downloader/internal/infrastructure/runner"
)

const progressPrefix = "dl-progress "

type YtDlpDownloader struct{}

func NewYtDlpDownloader() *YtDlpDownloader {
	return &YtDlpDownloader{}
}

func (d *YtDlpDownloader) Download(ctx context.Context, req domain.DownloadRequest, onProgress func(percent float64)) (string, error) {

	var path string

	err := runner.Run(ctx, func(line string) {
		if p, ok := parseProgress(line); ok {
			if onProgress != nil {
				onProgress(p)
			}
			return
		}
		// --print after_move:filepath prints the final path (after mp3 extraction)
		if filepath.IsAbs(line) {
			path = line
		}
	}, "yt-dlp", buildArgs(req)...)

	if err != nil {
		return "", err
	}
	if path == "" {
		return "", errors.New("yt-dlp did not report the downloaded file")
	}
	return path, nil
}

func buildArgs(req domain.DownloadRequest) []string {

	args := []string{
		"--no-playlist", // a video link from a playlist downloads only that video
		"--newline",
		"--progress", // --print turns on quiet mode; keep the progress lines
		"--progress-template", "download:" + progressPrefix + "%(progress._percent)s",
		"--print", "after_move:filepath",
		"-P", req.SavePath,
	}

	if req.Format == domain.Audio {
		// audio only, smallest size. "w" is the fallback for sites
		// that have no separate audio stream.
		args = append(args, "-f", "wa/w", "-x", "--audio-format", "mp3")
	} else {
		args = append(args, "-f", "worst")
	}

	// "--" so a link can never be read as an option
	return append(args, "--", req.URL)
}

// parseProgress reads lines like "dl-progress 45.3".
func parseProgress(line string) (float64, bool) {
	rest, found := strings.CutPrefix(line, progressPrefix)
	if !found {
		return 0, false
	}
	p, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(rest), "%"), 64)
	if err != nil {
		return 0, false // "NA" when the size is not known
	}
	return min(max(p, 0), 100), true
}
