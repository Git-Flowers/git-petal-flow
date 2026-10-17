package workflow

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	epicNumber   = regexp.MustCompile(`^epic/(\d+)-`)
	childEpicTag = regexp.MustCompile(`^epic-\d+[-/]`)
)

// EpicNumber returns the number that opens an epic's name, such as "001" for
// epic/001-some-epic. It returns "" when the epic name has no leading number.
func EpicNumber(epic string) string {
	m := epicNumber.FindStringSubmatch(epic)
	if m == nil {
		return ""
	}
	return m[1]
}

// NamingHint suggests a name for a branch that is about to be created. It
// returns "" when the name already follows the epic convention or when the
// branch has nothing to do with an epic. The hint is advice only: gitgud still
// creates the branch.
//
// A child of epic/001-some-epic should be named feature/epic-001/<slug>, so a
// person can see which epic it belongs to from the branch list alone.
func NamingHint(t Type, parent, branch string) string {
	if t == Epic && EpicNumber(branch) == "" {
		return fmt.Sprintf("hint: name epics with a number, for example epic/001-%s, so children can be matched to them",
			strings.TrimPrefix(branch, Epic.Prefix()))
	}
	if !strings.HasPrefix(parent, Epic.Prefix()) {
		return ""
	}
	num := EpicNumber(parent)
	if num == "" {
		return ""
	}
	slug := strings.TrimPrefix(branch, t.Prefix())
	want := "epic-" + num + "/"
	if strings.HasPrefix(slug, want) {
		return ""
	}
	slug = childEpicTag.ReplaceAllString(slug, "")
	return fmt.Sprintf("hint: %s is a child of %s; name it %s%s%s so people can see the epic from the branch name",
		branch, parent, t.Prefix(), want, slug)
}
