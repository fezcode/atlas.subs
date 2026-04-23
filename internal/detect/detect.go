package detect

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	bracketed  = regexp.MustCompile(`[\[\{][^\[\]\{\}]*[\]\}]`)
	parens     = regexp.MustCompile(`\(([^()]*)\)`)
	year       = regexp.MustCompile(`\b(19|20)\d{2}\b`)
	whitespace = regexp.MustCompile(`\s+`)
	junkTokens = regexp.MustCompile(`(?i)\b(webrip|web-dl|webdl|bluray|blu-ray|bdrip|brrip|dvdrip|dvdscr|hdtv|hdrip|hdcam|cam|ts|x264|x265|h264|h265|hevc|aac|ac3|dts|1080p|720p|480p|2160p|4k|uhd|10bit|hdr|remux|repack|proper|extended|uncut|unrated|imax|yts|yify|rarbg|ettv|eztv|amzn|nf|hulu|dsnp|atvp)\b`)
	separators = regexp.MustCompile(`[._]+`)
)

// FromCWD returns a cleaned search query derived from the current working
// directory's base name, but only if the name looks like a media folder
// (contains a year, a release tag, or bracketed metadata). Returns "" for
// ordinary directories so we don't auto-search on e.g. Documents or tmp.
func FromCWD() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	base := filepath.Base(wd)
	if !looksLikeMedia(base) {
		return ""
	}
	return Clean(base)
}

func looksLikeMedia(name string) bool {
	if year.MatchString(name) {
		return true
	}
	if bracketed.MatchString(name) {
		return true
	}
	if junkTokens.MatchString(name) {
		return true
	}
	return false
}

// Clean parses a movie/show folder name into a search query. It strips
// bracketed release tags, converts dots/underscores to spaces, removes common
// scene tags, and preserves the title and year.
func Clean(name string) string {
	s := name

	s = bracketed.ReplaceAllString(s, " ")

	if m := parens.FindStringSubmatch(s); m != nil && year.MatchString(m[1]) {
		s = parens.ReplaceAllString(s, " "+m[1]+" ")
	} else {
		s = parens.ReplaceAllString(s, " ")
	}

	s = separators.ReplaceAllString(s, " ")
	s = junkTokens.ReplaceAllString(s, " ")
	s = whitespace.ReplaceAllString(s, " ")
	s = strings.Trim(s, " -_.")

	return s
}
