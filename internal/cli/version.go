package cli

import "runtime/debug"

// Version is set at build time with -ldflags "-X .../internal/cli.Version=...".
// Release builds set it; otherwise it's filled in from the Go build info.
var Version = "dev"

// How this binary was built. It decides what `chekin update` may do.
const (
	buildRelease   = "release"    // from a GitHub release (Version set with -ldflags)
	buildGoInstall = "go install" // go install github.com/...@version
	buildDev       = "dev"        // built from a checkout
)

var buildKind string

func init() {
	bi, _ := debug.ReadBuildInfo()
	Version, buildKind = detectBuild(Version, bi)
}

// detectBuild works out the version and build kind. `go install module@v1.2.3`
// records v1.2.3 as the main module version and has no VCS information, while a
// `go build` in a checkout records the commit.
func detectBuild(ldVersion string, bi *debug.BuildInfo) (version, kind string) {
	if ldVersion != "dev" && ldVersion != "" {
		return ldVersion, buildRelease
	}
	if bi == nil || bi.Main.Version == "" || bi.Main.Version == "(devel)" {
		return "dev", buildDev
	}
	for _, s := range bi.Settings {
		if s.Key == "vcs.revision" {
			return bi.Main.Version, buildDev
		}
	}
	return bi.Main.Version, buildGoInstall
}
