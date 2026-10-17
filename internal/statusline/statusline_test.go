package statusline

import "testing"

func TestDirFromInput(t *testing.T) {
	cases := map[string]string{
		`{"workspace":{"current_dir":"/a"},"cwd":"/b"}`: "/a",
		`{"cwd":"/b"}`:                       "/b",
		`{"workspace":{"project_dir":"/c"}}`: "",
		`not json`:                           "",
		``:                                   "",
	}
	for in, want := range cases {
		if got := DirFromInput([]byte(in)); got != want {
			t.Errorf("DirFromInput(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestColourRespectsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if got := green("x"); got != "x" {
		t.Errorf("NO_COLOR still coloured: %q", got)
	}
	t.Setenv("NO_COLOR", "")
	if got := green("x"); got == "x" {
		t.Errorf("colour missing without NO_COLOR")
	}
}
