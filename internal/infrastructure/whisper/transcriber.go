package whisper

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"downloader/internal/domain"
	"downloader/internal/infrastructure/runner"
)

type WhisperTranscriber struct {
	threads int
}

func NewWhisperTranscriber(threads int) *WhisperTranscriber {
	return &WhisperTranscriber{threads: threads}
}

// Transcribe writes <name>.txt and <name>.srt next to the media file
// (<name>.en.txt when translating) and returns the .txt path.
func (t *WhisperTranscriber) Transcribe(ctx context.Context, mediaPath string, opts domain.TranscribeOptions, onProgress func(percent float64)) (string, error) {

	if _, err := os.Stat(opts.ModelPath); err != nil {
		return "", fmt.Errorf("whisper model not found: %s", opts.ModelPath)
	}

	// whisper.cpp needs 16kHz mono PCM WAV
	wav, err := os.CreateTemp("", "downloader-*.wav")
	if err != nil {
		return "", err
	}
	wav.Close()
	defer os.Remove(wav.Name())

	err = runner.Run(ctx, nil, "ffmpeg",
		"-nostdin", "-y",
		"-i", mediaPath,
		"-vn",
		"-ar", "16000",
		"-ac", "1",
		"-c:a", "pcm_s16le",
		"-loglevel", "error",
		wav.Name(),
	)
	if err != nil {
		return "", err
	}

	base := outputBase(mediaPath, opts)

	err = runner.Run(ctx, func(line string) {
		if p, ok := parseProgress(line); ok && onProgress != nil {
			onProgress(float64(p))
		}
	}, "whisper-cli", buildArgs(opts, wav.Name(), base, t.threads)...)
	if err != nil {
		return "", err
	}

	return base + ".txt", nil
}

// outputBase keeps the original-language text when translating:
// "talk.mp3" -> "talk", or "talk.en" with Translate.
func outputBase(mediaPath string, opts domain.TranscribeOptions) string {
	base := strings.TrimSuffix(mediaPath, filepath.Ext(mediaPath))
	if opts.Translate {
		base += ".en"
	}
	return base
}

func buildArgs(opts domain.TranscribeOptions, wav, base string, threads int) []string {

	lang := opts.Language
	if lang == "" {
		lang = domain.LangAuto
	}

	args := []string{
		"-m", opts.ModelPath,
		"-f", wav,
		"-l", lang,
		"-of", base,
		"-otxt", "-osrt",
		"-pp",
		"-t", strconv.Itoa(threads),
		"-mc", "0", // no previous text as context: stops repeat loops on silence
		"-et", "2.8",
		"-tpi", "0.2",
	}
	if opts.Translate {
		args = append(args, "-tr")
	}
	return args
}

// parseProgress reads lines like "whisper_print_progress_callback: progress =  45%".
func parseProgress(line string) (int, bool) {
	_, after, found := strings.Cut(line, "progress =")
	if !found {
		return 0, false
	}
	p, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(after), "%"))
	if err != nil {
		return 0, false
	}
	return p, true
}

// ListModels finds the models in dir: ggml-medium-q5_0.bin -> "medium-q5_0".
// Voice-detection (silero) models are skipped.
func ListModels(dir string) []domain.Model {

	paths, _ := filepath.Glob(filepath.Join(dir, "ggml-*.bin"))
	slices.Sort(paths)

	var models []domain.Model
	for _, p := range paths {
		name := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(p), "ggml-"), ".bin")
		if strings.Contains(name, "silero") {
			continue
		}
		models = append(models, domain.Model{Name: name, Path: p})
	}
	return models
}
