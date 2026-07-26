package version

import "runtime/debug"

// Version is the current build version. It is overridable via
// -ldflags "-X github.com/taigrr/teaqlite/internal/version.Version=v1.2.3"
// and otherwise falls back to the module version embedded by the toolchain.
var Version = "devel"

func init() {
	if Version != "devel" {
		return
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			Version = v
		}
	}
}
