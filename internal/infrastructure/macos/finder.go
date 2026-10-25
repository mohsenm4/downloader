package macos

import "os/exec"

// Reveal shows the file selected in Finder.
func Reveal(path string) error {
	return exec.Command("/usr/bin/open", "-R", path).Run()
}

// Open opens a file or folder with its default app.
func Open(path string) error {
	return exec.Command("/usr/bin/open", path).Run()
}
