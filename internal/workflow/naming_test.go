package workflow

import (
	"strings"
	"testing"
)

func TestNamingHint(t *testing.T) {
	cases := []struct {
		name   string
		t      Type
		parent string
		branch string
		want   string // substring the hint must contain; "" means no hint
	}{
		{"correct child", Feature, "epic/001-some-epic", "feature/epic-001/some-child", ""},
		{"hyphen child", Feature, "epic/001-some-epic", "feature/epic-001-some-child", "feature/epic-001/some-child"},
		{"no epic parent", Feature, "develop", "feature/login", ""},
		{"wrong number", Feature, "epic/001-some-epic", "feature/epic-002/child", "feature/epic-001/child"},
		{"unnumbered child", Fix, "epic/001-some-epic", "fix/session", "fix/epic-001/session"},
		{"unnumbered epic", Epic, "develop", "epic/auth", "epic/001-auth"},
		{"numbered epic", Epic, "develop", "epic/001-auth", ""},
	}
	for _, c := range cases {
		got := NamingHint(c.t, c.parent, c.branch)
		if c.want == "" {
			if got != "" {
				t.Errorf("%s: hint %q; want none", c.name, got)
			}
			continue
		}
		if !strings.Contains(got, c.want) {
			t.Errorf("%s: hint %q; want it to contain %q", c.name, got, c.want)
		}
	}
}

func TestEpicNumber(t *testing.T) {
	if got := EpicNumber("epic/001-some-epic"); got != "001" {
		t.Errorf("EpicNumber = %q; want 001", got)
	}
	if got := EpicNumber("epic/auth"); got != "" {
		t.Errorf("EpicNumber(epic/auth) = %q; want empty", got)
	}
}
