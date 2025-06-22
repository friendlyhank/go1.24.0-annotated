package buildcfg

import (
	"fmt"
	"os"
	"path/filepath"
)

var (
	GOARCH = envOr("GOARCH", defaultGOARCH)
)

// Error is one of the errors found (if any) in the build configuration.
var Error error

// Check exits the program with a fatal error if Error is non-nil.
func Check() {
	if Error != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", filepath.Base(os.Args[0]), Error)
		os.Exit(2)
	}
}

func envOr(key, value string) string {
	if x := os.Getenv(key); x != "" {
		return x
	}
	return value
}
