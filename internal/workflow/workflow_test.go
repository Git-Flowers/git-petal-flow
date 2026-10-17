package workflow

import (
	"errors"
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"readline from user":  "readline-from-user",
		"Fix -- Login!!  bug": "fix-login-bug",
		"  --x--  ":           "x",
		"epic-001/some child": "epic-001/some-child",
		"/a//b/":              "a/b",
	}
	for in, want := range cases {
		got, err := Slugify(in)
		if err != nil || got != want {
			t.Errorf("Slugify(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "!!! ---", "日本語"} {
		if _, err := Slugify(in); !errors.Is(err, ErrEmptySlug) {
			t.Errorf("Slugify(%q) err = %v; want ErrEmptySlug", in, err)
		}
	}
	long, err := Slugify(strings.Repeat("abcdefghij ", 10))
	if err != nil || len(long) > maxSlug || strings.HasSuffix(long, "-") {
		t.Errorf("long slug %q (len %d) err %v; want <=40 chars, no trailing hyphen", long, len(long), err)
	}
}

func TestValidateParent(t *testing.T) {
	openEpic := func(b string) bool { return b == "epic/auth" }
	ok := []struct {
		t      Type
		parent string
	}{
		{Feature, "develop"},
		{Fix, "develop"},
		{Feature, "epic/auth"},
		{Epic, "develop"},
		{Release, "develop"},
	}
	for _, c := range ok {
		if err := ValidateParent(c.t, c.parent, openEpic); err != nil {
			t.Errorf("%s from %s: unexpected %v", c.t, c.parent, err)
		}
	}
	bad := []struct {
		t      Type
		parent string
	}{
		{Feature, "main"},
		{Feature, "epic/closed"},
		{Epic, "main"},
		{Release, "epic/auth"},
	}
	for _, c := range bad {
		if err := ValidateParent(c.t, c.parent, openEpic); !errors.Is(err, ErrParentNotAllowed) {
			t.Errorf("%s from %s: err = %v; want ErrParentNotAllowed", c.t, c.parent, err)
		}
	}
}

func TestParseType(t *testing.T) {
	if _, err := ParseType("hotfix"); !errors.Is(err, ErrUnknownType) {
		t.Errorf("hotfix err = %v; want ErrUnknownType", err)
	}
	if got, err := ParseType("fix"); err != nil || got != Fix {
		t.Errorf("fix = %v, %v", got, err)
	}
}

func TestValidateVersion(t *testing.T) {
	for _, v := range []string{"1.4.0", "0.0.1"} {
		if err := ValidateVersion(v); err != nil {
			t.Errorf("%s: %v", v, err)
		}
	}
	for _, v := range []string{"1.4", "v1.4.0", "1.4.0-rc1", ""} {
		if err := ValidateVersion(v); !errors.Is(err, ErrBadVersion) {
			t.Errorf("%q err = %v; want ErrBadVersion", v, err)
		}
	}
}

func TestConventionalType(t *testing.T) {
	cases := map[string]string{
		"feat(auth): add login": "feat",
		"fix!: drop legacy api": "fix",
		"chore: bump deps":      "chore",
	}
	for subject, want := range cases {
		got := ConventionalType(subject)
		if got == nil || *got != want {
			t.Errorf("ConventionalType(%q) = %v; want %q", subject, got, want)
		}
	}
	if got := ConventionalType("Update README"); got != nil {
		t.Errorf("non-conventional subject gave %q; want nil", *got)
	}
}
