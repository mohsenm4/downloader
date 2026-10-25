package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// Config holds what the app needs from the machine, in one place.
type Config struct {
	ModelsDir string   // whisper.cpp models (ggml-*.bin)
	Threads   int      // CPU threads for whisper
	Tools     []string // programs that must be in PATH
}

// Homebrew folders. Apps opened from the Dock only get /usr/bin:/bin:/usr/sbin:/sbin,
// so yt-dlp, ffmpeg and whisper-cli are not found without these.
var homebrewPaths = []string{"/opt/homebrew/bin", "/usr/local/bin"}

// Brew package for each tool, for the install hint.
var brewPackages = map[string]string{
	"yt-dlp":      "yt-dlp",
	"ffmpeg":      "ffmpeg",
	"whisper-cli": "whisper-cpp",
}

func Default() Config {
	dir, err := os.UserConfigDir() // ~/Library/Application Support
	if err != nil {
		dir = os.TempDir()
	}

	return Config{
		ModelsDir: filepath.Join(dir, "Downloader", "models"),
		Threads:   min(runtime.NumCPU(), 8),
		Tools:     []string{"yt-dlp", "ffmpeg", "whisper-cli"},
	}
}

// FixPath puts the Homebrew folders in PATH so tools can be run by name.
func FixPath() {
	parts := filepath.SplitList(os.Getenv("PATH"))
	for _, p := range slices.Backward(homebrewPaths) {
		if !slices.Contains(parts, p) {
			parts = append([]string{p}, parts...)
		}
	}
	os.Setenv("PATH", strings.Join(parts, string(os.PathListSeparator)))
}

// MissingTools returns the tools that are not in PATH.
func (c Config) MissingTools() []string {
	var missing []string
	for _, t := range c.Tools {
		if _, err := exec.LookPath(t); err != nil {
			missing = append(missing, t)
		}
	}
	return missing
}

// InstallCommand returns the brew command that installs the given tools.
func InstallCommand(tools []string) string {
	pkgs := make([]string, 0, len(tools))
	for _, t := range tools {
		if p, ok := brewPackages[t]; ok {
			pkgs = append(pkgs, p)
		} else {
			pkgs = append(pkgs, t)
		}
	}
	return "brew install " + strings.Join(pkgs, " ")
}
