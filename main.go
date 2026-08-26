package main

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/app"
)

var Version = "dev" // This will be set by the build systems to the release version

var semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+`)

func normalizedVersion(version string) string {
	if semverRe.MatchString(version) && !strings.HasPrefix(version, "v") {
		return "v" + version
	}
	return version
}

func versionString(version string) string {
	return fmt.Sprintf("ghrepocfg version %s (%s, %s/%s)", normalizedVersion(version), runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

// main is the entry point for the ghrepocfg command-line tool.
func main() {
	// Set the build version from the build info if not set by the build system
	if Version == "dev" || Version == "" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
				Version = bi.Main.Version
			}
		}
	}

	os.Exit(app.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, versionString(Version)))
}
