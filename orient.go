package main

import "strings"

// Lipstick /lipstick/orientationLock. "landscape" stays on
// Qt.LandscapeOrientation and does not flip to the inverted one.
const orientLandscape = "landscape"

func parseOrient(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		s = s[1 : len(s)-1]
	}
	s = strings.Trim(s, `"`)
	switch s {
	case "dynamic", "portrait", "portrait-inverted", "landscape", "landscape-inverted":
		return s, true
	default:
		return "", false
	}
}
