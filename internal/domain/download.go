package domain

import "path/filepath"

type Format string

const (
	Audio Format = "audio"
	Video Format = "video"
)

type DownloadRequest struct {
	URL      string
	SavePath string
	Format   Format
}

// Whisper language codes. Auto lets whisper guess from the first 30 seconds.
const (
	LangAuto    = "auto"
	LangPersian = "fa"
	LangEnglish = "en"
)

type TranscribeOptions struct {
	Language  string
	ModelPath string
	Translate bool // to English; whisper can only translate to English
}

// Model is a whisper.cpp model file.
type Model struct {
	Name string // e.g. "medium-q5_0"
	Path string
}

// Job is one item in the queue: a link to download, or a local file.
// Exactly one of URL and FilePath is set.
type Job struct {
	URL        string
	FilePath   string
	SavePath   string
	Format     Format
	Transcribe bool
	Options    TranscribeOptions
}

// Title is a short name for the UI.
func (j Job) Title() string {
	if j.FilePath != "" {
		return filepath.Base(j.FilePath)
	}
	return j.URL
}

type Result struct {
	MediaPath string
	TextPath  string // empty when the job did not transcribe
}
