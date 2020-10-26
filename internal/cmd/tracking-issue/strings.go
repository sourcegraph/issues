package main

import "strings"

func partition(s, beginMarker, endMarker string) (string, string, string, bool) {
	start, ok := indexOf(s, beginMarker)
	if !ok {
		return "", "", "", false
	}
	end, ok := indexOf(s[start:], endMarker)
	if !ok {
		return "", "", "", false
	}
	end += start              // adjust slice bounds
	start += len(beginMarker) // keep begin marker in suffix

	return s[:start], s[start:end], s[end:], true
}

func indexOf(s, marker string) (int, bool) {
	if location := strings.Index(s, marker); location != -1 {
		return location, true
	}

	return -1, false
}
