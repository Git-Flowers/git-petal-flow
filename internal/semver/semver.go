// Package semver parses and compares versions as defined by semver.org 2.0.0.
package semver

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ErrInvalid is returned for any string that is not a valid semantic version.
var ErrInvalid = errors.New("invalid semantic version")

// Version is a parsed semantic version.
type Version struct {
	Major, Minor, Patch uint64
	// Pre holds pre-release identifiers, empty for a release version.
	Pre []string
	// Build holds build metadata. It is ignored when comparing versions.
	Build []string
}

// The official grammar from semver.org, with a leading "v" rejected.
var pattern = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)` +
	`(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?` +
	`(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

// Parse reads a version string such as "1.4.0", "2.0.0-rc.1" or "1.0.0+build.5".
func Parse(s string) (Version, error) {
	m := pattern.FindStringSubmatch(s)
	if m == nil {
		return Version{}, fmt.Errorf("%w: %q", ErrInvalid, s)
	}
	var v Version
	var err error
	if v.Major, err = strconv.ParseUint(m[1], 10, 64); err != nil {
		return Version{}, fmt.Errorf("%w: major: %v", ErrInvalid, err)
	}
	if v.Minor, err = strconv.ParseUint(m[2], 10, 64); err != nil {
		return Version{}, fmt.Errorf("%w: minor: %v", ErrInvalid, err)
	}
	if v.Patch, err = strconv.ParseUint(m[3], 10, 64); err != nil {
		return Version{}, fmt.Errorf("%w: patch: %v", ErrInvalid, err)
	}
	if m[4] != "" {
		v.Pre = strings.Split(m[4], ".")
	}
	if m[5] != "" {
		v.Build = strings.Split(m[5], ".")
	}
	return v, nil
}

// IsRelease reports whether v has no pre-release part.
func (v Version) IsRelease() bool {
	return len(v.Pre) == 0
}

// String renders v in canonical form.
func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if len(v.Pre) > 0 {
		s += "-" + strings.Join(v.Pre, ".")
	}
	if len(v.Build) > 0 {
		s += "+" + strings.Join(v.Build, ".")
	}
	return s
}

// Compare returns -1, 0 or 1 as a is lower than, equal to, or higher than b,
// using semver precedence. Build metadata does not affect precedence.
func Compare(a, b Version) int {
	for _, pair := range [][2]uint64{{a.Major, b.Major}, {a.Minor, b.Minor}, {a.Patch, b.Patch}} {
		if c := cmpUint(pair[0], pair[1]); c != 0 {
			return c
		}
	}
	switch {
	case a.IsRelease() && b.IsRelease():
		return 0
	case a.IsRelease():
		return 1
	case b.IsRelease():
		return -1
	}
	return comparePre(a.Pre, b.Pre)
}

func comparePre(a, b []string) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if c := compareIdent(a[i], b[i]); c != 0 {
			return c
		}
	}
	return cmpUint(uint64(len(a)), uint64(len(b)))
}

// compareIdent orders numeric identifiers numerically, compares alphanumeric
// ones lexically, and ranks numeric below alphanumeric.
func compareIdent(a, b string) int {
	an, aNum := numeric(a)
	bn, bNum := numeric(b)
	switch {
	case aNum && bNum:
		return cmpUint(an, bn)
	case aNum:
		return -1
	case bNum:
		return 1
	}
	return strings.Compare(a, b)
}

func numeric(s string) (uint64, bool) {
	n, err := strconv.ParseUint(s, 10, 64)
	return n, err == nil
}

func cmpUint(a, b uint64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}
