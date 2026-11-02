package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func String() string {
	manta := "unknown"
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, d := range bi.Deps {
			if d.Path == "github.com/dotabuff/manta" {
				manta = d.Version
			}
		}
	}
	return fmt.Sprintf("manta-cli %s (commit %s, built %s, %s, manta %s)",
		Version, Commit, Date, runtime.Version(), manta)
}
