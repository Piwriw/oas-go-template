// Package version holds build-time metadata injected via -ldflags "-X ...".
// Default values are used when running `go run` or `go build` without ldflags.
package version

// These vars are overwritten at build time via:
//
//	go build -ldflags "-X github.com/piwriw/oas-go-template/internal/version.Version=x.y.z ..."
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

// Info returns the structured build metadata.
func Info() InfoT {
	return InfoT{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
	}
}

// InfoT is the structured form of the build metadata.
type InfoT struct {
	Version   string
	GitCommit string
	BuildTime string
}
