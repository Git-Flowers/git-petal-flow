package semver

import (
	"errors"
	"testing"
)

func TestParseValid(t *testing.T) {
	cases := map[string]string{
		"1.0.0":                      "1.0.0",
		"0.0.1":                      "0.0.1",
		"10.20.30":                   "10.20.30",
		"1.0.0-alpha.1":              "1.0.0-alpha.1",
		"1.0.0-0.3.7":                "1.0.0-0.3.7",
		"1.0.0-x.7.z.92":             "1.0.0-x.7.z.92",
		"1.0.0+20130313144700":       "1.0.0+20130313144700",
		"1.0.0-beta+exp.sha.5114f85": "1.0.0-beta+exp.sha.5114f85",
	}
	for in, want := range cases {
		v, err := Parse(in)
		if err != nil {
			t.Errorf("Parse(%q): %v", in, err)
			continue
		}
		if v.String() != want {
			t.Errorf("Parse(%q).String() = %q; want %q", in, v.String(), want)
		}
	}
}

func TestParseRejectsInvalid(t *testing.T) {
	// Each of these is forbidden by the semver 2.0.0 grammar.
	for _, in := range []string{
		"", "1", "1.2", "1.2.3.4", "01.2.3", "1.02.3", "1.2.03",
		"v1.2.3", "1.2.3-", "1.2.3-01", "1.2.3+", "1.2.3-alpha..1",
		"1.2.3 ", "a.b.c", "99999999999999999999.0.0",
	} {
		if _, err := Parse(in); !errors.Is(err, ErrInvalid) {
			t.Errorf("Parse(%q) err = %v; want ErrInvalid", in, err)
		}
	}
}

// The precedence example from semver.org section 11.
func TestCompareSpecOrder(t *testing.T) {
	ordered := []string{
		"1.0.0-alpha",
		"1.0.0-alpha.1",
		"1.0.0-alpha.beta",
		"1.0.0-beta",
		"1.0.0-beta.2",
		"1.0.0-beta.11",
		"1.0.0-rc.1",
		"1.0.0",
		"2.0.0",
		"2.1.0",
		"2.1.1",
	}
	for i := 0; i+1 < len(ordered); i++ {
		a, _ := Parse(ordered[i])
		b, _ := Parse(ordered[i+1])
		if c := Compare(a, b); c != -1 {
			t.Errorf("Compare(%s, %s) = %d; want -1", a, b, c)
		}
		if c := Compare(b, a); c != 1 {
			t.Errorf("Compare(%s, %s) = %d; want 1", b, a, c)
		}
	}
}

func TestCompareIgnoresBuild(t *testing.T) {
	a, _ := Parse("1.0.0+one")
	b, _ := Parse("1.0.0+two")
	if Compare(a, b) != 0 {
		t.Error("build metadata changed precedence")
	}
}

func TestIsRelease(t *testing.T) {
	r, _ := Parse("1.0.0")
	p, _ := Parse("1.0.0-rc.1")
	if !r.IsRelease() || p.IsRelease() {
		t.Error("IsRelease wrong for release or pre-release")
	}
}
