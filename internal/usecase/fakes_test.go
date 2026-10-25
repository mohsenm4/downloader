package usecase

import (
	"context"
	"sync"

	"downloader/internal/domain"
)

type FakeDownloader struct {
	mu    sync.Mutex
	Calls []domain.DownloadRequest
	Err   error
	Block bool // wait until ctx is canceled
}

func (f *FakeDownloader) Download(ctx context.Context, req domain.DownloadRequest, onProgress func(float64)) (string, error) {
	f.mu.Lock()
	f.Calls = append(f.Calls, req)
	f.mu.Unlock()

	onProgress(50)
	if f.Block {
		<-ctx.Done()
		return "", ctx.Err()
	}
	if f.Err != nil {
		return "", f.Err
	}
	return req.SavePath + "/video.mp3", nil
}

type FakeTranscriber struct {
	mu    sync.Mutex
	Paths []string
	Err   error
}

func (f *FakeTranscriber) Transcribe(ctx context.Context, mediaPath string, opts domain.TranscribeOptions, onProgress func(float64)) (string, error) {
	f.mu.Lock()
	f.Paths = append(f.Paths, mediaPath)
	f.mu.Unlock()

	onProgress(100)
	if f.Err != nil {
		return "", f.Err
	}
	return mediaPath + ".txt", nil
}

var opts = domain.TranscribeOptions{Language: domain.LangAuto, ModelPath: "/models/m.bin"}
