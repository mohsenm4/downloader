package ui

import (
	"strings"
)

// errorHints turn common yt-dlp / ffmpeg / whisper errors into plain words.
// The full error is still shown by the "Why?" button.
var errorHints = []struct{ match, text string }{
	{"not a bot", "YouTube wants a sign-in. Try another VPN server."},
	{"Redirection detected", "The site redirected. The video may be removed, or blocked where your VPN is."},
	{"Unsupported URL", "This site is not supported."},
	{"HTTP Error 403", "The site refused the download (403). Try another VPN server."},
	{"HTTP Error 404", "Not found (404). The link may be wrong or removed."},
	{"Video unavailable", "This video is not available."},
	{"Private video", "This video is private."},
	{"Unable to download webpage", "Could not reach the site. Check your internet or VPN."},
	{"timed out", "The connection timed out. Check your internet or VPN."},
	{"whisper model not found", "The text model file is missing. Check the Model setting."},
	{"Invalid data found when processing input", "This file has no audio that can be read."},
}

// friendlyError returns a short, plain message for the queue row.
func friendlyError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	for _, h := range errorHints {
		if strings.Contains(msg, h.match) {
			return h.text
		}
	}
	return lastLine(err)
}

// lastLine is the most useful part of a program error, like "ERROR: Unsupported URL".
func lastLine(err error) string {
	lines := strings.Split(strings.TrimSpace(err.Error()), "\n")
	return lines[len(lines)-1]
}
