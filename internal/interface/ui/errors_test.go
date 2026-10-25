package ui

import (
	"errors"
	"testing"
)

func TestFriendlyError(t *testing.T) {

	tests := []struct {
		err  error
		want string
	}{
		{errors.New("yt-dlp: exit status 1\nERROR: [youtube] x: Sign in to confirm you’re not a bot."),
			"YouTube wants a sign-in. Try another VPN server."},
		{errors.New("yt-dlp: exit status 1\nERROR: [generic] x: Redirection detected; the video may be deleted"),
			"The site redirected. The video may be removed, or blocked where your VPN is."},
		{errors.New("yt-dlp: exit status 1\nWARNING: a\nERROR: something new"), "ERROR: something new"},
		{nil, ""},
	}

	for _, tt := range tests {
		if got := friendlyError(tt.err); got != tt.want {
			t.Errorf("friendlyError(%v) = %q, want %q", tt.err, got, tt.want)
		}
	}
}

func TestPrettyLink(t *testing.T) {

	tests := map[string]string{
		"https://www.youtube.com/watch?v=jNQXAC9IVRw": "youtube.com · jNQXAC9IVRw",
		"https://soundcloud.com/artist/talk":          "soundcloud.com · /artist/talk",
		"not a url":                                   "not a url",
	}

	for in, want := range tests {
		if got := prettyLink(in); got != want {
			t.Errorf("prettyLink(%q) = %q, want %q", in, got, want)
		}
	}
}
