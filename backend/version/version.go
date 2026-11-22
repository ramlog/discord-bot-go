// Package version holds build information that is injected at compile time:
//
//	go build -ldflags "-X github.com/cyb3rplis/discord-bot-go/version.Version=v2.0.0 ..."
//
// Without -ldflags (e.g. `go run` locally) the defaults below are used.
package version

import (
	"fmt"
	"runtime"
	"time"
)

var (
	Version   = "dev"     // git describe output, e.g. v2.0.0 or v2.0.0-3-gabc1234
	Commit    = "unknown" // full git commit SHA
	BuildDate = "unknown" // RFC3339 UTC timestamp of the build
	RepoURL   = ""        // e.g. https://github.com/ramlog/discord-bot-go
)

// StartedAt is when the bot process started, used for the uptime.
var StartedAt = time.Now()

// ShortCommit returns the first 7 characters of the commit SHA.
func ShortCommit() string {
	if len(Commit) > 7 {
		return Commit[:7]
	}
	return Commit
}

// String returns a one-line summary for logs.
func String() string {
	return fmt.Sprintf("%s (commit %s, built %s, %s)", Version, ShortCommit(), BuildDate, runtime.Version())
}
