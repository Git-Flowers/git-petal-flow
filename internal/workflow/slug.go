package workflow

import (
	"errors"
	"regexp"
	"strings"
)

// ErrEmptySlug means a description had no ASCII letters or digits.
var ErrEmptySlug = errors.New("description has no letters or digits to build a branch name from")

const maxSlug = 40

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify turns free text into a branch-safe slug: lowercase ASCII, runs of
// other characters collapsed to one hyphen, at most 40 characters. A "/" in the
// input is kept as a separator between segments, so "epic-001/some child"
// becomes "epic-001/some-child". That is how a child of an epic gets the
// feature/epic-001/<name> form.
func Slugify(s string) (string, error) {
	var segs []string
	for _, part := range strings.Split(s, "/") {
		seg := nonSlug.ReplaceAllString(strings.ToLower(part), "-")
		seg = strings.Trim(seg, "-")
		if seg != "" {
			segs = append(segs, seg)
		}
	}
	slug := strings.Join(segs, "/")
	if len(slug) > maxSlug {
		slug = strings.TrimRight(slug[:maxSlug], "-/")
	}
	if slug == "" {
		return "", ErrEmptySlug
	}
	return slug, nil
}
