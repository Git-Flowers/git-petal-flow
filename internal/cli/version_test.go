package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/josepalala-dev/gitgud/internal/semver"
)

func TestVersionIsSemver(t *testing.T) {
	v, err := semver.Parse(Version)
	if err != nil {
		t.Fatalf("Version %q is not semver: %v", Version, err)
	}
	if !v.IsRelease() {
		t.Errorf("Version %q is a pre-release; the MVP ships a release", Version)
	}
}

func TestVersionFlagPrintsVersion(t *testing.T) {
	var out, errb bytes.Buffer
	if code := Run(t.TempDir(), []string{"--version"}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("--version exit %d: %s", code, errb.String())
	}
	if got := strings.TrimSpace(out.String()); got != "gitgud "+Version {
		t.Errorf("--version printed %q", got)
	}
}
