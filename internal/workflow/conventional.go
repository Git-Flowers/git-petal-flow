package workflow

import "regexp"

var conventional = regexp.MustCompile(`^([a-z]+)(\([^)]*\))?!?: `)

// ConventionalType returns the Conventional Commit type of a subject line,
// such as "feat" for "feat(auth): add login". It returns nil when the subject
// does not follow the convention.
func ConventionalType(subject string) *string {
	m := conventional.FindStringSubmatch(subject)
	if m == nil {
		return nil
	}
	return &m[1]
}
