// Package buildinfo holds build-time metadata injected via -ldflags.
package buildinfo

// These values are overridden at build time, e.g.
//
//	go build -ldflags "-X github.com/aiShuiJiaoDeXioShou/lhcli/internal/buildinfo.Version=v1.0.0"
var (
	// Version is the semantic version of the build.
	Version = "dev"
	// Commit is the git commit the binary was built from.
	Commit = "none"
	// Date is the build timestamp.
	Date = "unknown"
)
