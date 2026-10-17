package workflow

import (
	"errors"
	"strings"

	"github.com/josepalala-dev/gitgud/internal/semver"
)

// Long-lived branches.
const (
	Develop = "develop"
	Main    = "main"
)

// Type is the kind of branch gitgud creates.
type Type string

const (
	Feature Type = "feature"
	Fix     Type = "fix"
	Epic    Type = "epic"
	Release Type = "release"
)

var (
	ErrUnknownType      = errors.New("unknown branch type (want feature, fix, epic or release)")
	ErrParentNotAllowed = errors.New("parent branch is not allowed for this type")
	ErrBadVersion       = errors.New("version must be MAJOR.MINOR.PATCH, for example 1.4.0")
)

// ParseType accepts one of the four supported branch types.
func ParseType(s string) (Type, error) {
	switch t := Type(s); t {
	case Feature, Fix, Epic, Release:
		return t, nil
	}
	return "", ErrUnknownType
}

// Prefix returns the branch prefix for a type, such as "feature/".
func (t Type) Prefix() string {
	return string(t) + "/"
}

// ValidateParent checks that parent may be the starting point for a branch of
// type t. Features and fixes may start from develop or an open epic. Epics and
// releases may only start from develop. isOpenEpic reports whether an epic/*
// branch is recorded and still open.
func ValidateParent(t Type, parent string, isOpenEpic func(string) bool) error {
	switch t {
	case Feature, Fix:
		if parent == Develop {
			return nil
		}
		if strings.HasPrefix(parent, Epic.Prefix()) && isOpenEpic(parent) {
			return nil
		}
		return ErrParentNotAllowed
	case Epic, Release:
		if parent == Develop {
			return nil
		}
		return ErrParentNotAllowed
	}
	return ErrUnknownType
}

// ValidateVersion accepts a release version: semver core only, with no
// pre-release or build suffix, because release branches are tagged vX.Y.Z.
func ValidateVersion(v string) error {
	parsed, err := semver.Parse(v)
	if err != nil || !parsed.IsRelease() || len(parsed.Build) > 0 {
		return ErrBadVersion
	}
	return nil
}
