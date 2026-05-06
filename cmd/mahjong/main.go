package main

import (
	"cmp"
	"fmt"
	"os"
	"regexp"
	"runtime/debug"
	"strings"

	"github.com/benny123tw/mahjong-cli/cmd"
)

var (
	goVersion = "unknown"

	// Populated by goreleaser during build via -ldflags.
	version = "unknown"
	commit  = "?"
	date    = ""
)

func main() {
	info := createBuildInfo()
	if err := cmd.Execute(info); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func createBuildInfo() cmd.BuildInfo {
	info := cmd.BuildInfo{
		Commit:    commit,
		Version:   version,
		GoVersion: goVersion,
		Date:      date,
	}

	buildInfo, available := debug.ReadBuildInfo()
	if !available {
		return info
	}

	info.GoVersion = buildInfo.GoVersion

	// goreleaser sets `date` non-empty — prefer those values.
	if date != "" {
		return info
	}

	// `go install` / `go build` path: derive version from build info.
	info.Version = buildInfo.Main.Version
	if matched, _ := regexp.MatchString(`v\d+\.\d+\.\d+`, buildInfo.Main.Version); matched {
		info.Version = strings.TrimPrefix(buildInfo.Main.Version, "v")
	}

	var revision, modified string
	for _, setting := range buildInfo.Settings {
		switch setting.Key {
		case "vcs.time":
			info.Date = setting.Value
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value
		}
	}

	info.Date = cmp.Or(info.Date, "(unknown)")
	info.Commit = fmt.Sprintf(
		"(%s, modified: %s, mod sum: %q)",
		cmp.Or(revision, "unknown"),
		cmp.Or(modified, "?"),
		buildInfo.Main.Sum,
	)
	return info
}
