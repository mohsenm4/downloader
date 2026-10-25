package config

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestFixPath_AddsHomebrewOnce(t *testing.T) {

	t.Setenv("PATH", "/usr/bin:/bin:/usr/local/bin")

	FixPath()
	FixPath()

	parts := filepath.SplitList(os.Getenv("PATH"))
	want := []string{"/opt/homebrew/bin", "/usr/bin", "/bin", "/usr/local/bin"}
	if !slices.Equal(parts, want) {
		t.Errorf("PATH = %v, want %v", parts, want)
	}
}

func TestMissingTools(t *testing.T) {

	c := Config{Tools: []string{"sh", "no-such-tool-xyz"}}

	got := c.MissingTools()
	if !slices.Equal(got, []string{"no-such-tool-xyz"}) {
		t.Errorf("MissingTools() = %v", got)
	}
}

func TestInstallCommand(t *testing.T) {

	got := InstallCommand([]string{"yt-dlp", "whisper-cli"})
	if got != "brew install yt-dlp whisper-cpp" {
		t.Errorf("InstallCommand() = %q", got)
	}
}
