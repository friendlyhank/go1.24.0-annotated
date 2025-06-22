package main

import (
	"cmd/internal/sys"
	"cmd/link/internal/arm64"
	"cmd/link/internal/ld"
	"fmt"
	"internal/buildcfg"
	"os"
)

func main() {
	var arch *sys.Arch
	var theArch ld.Arch

	buildcfg.Check()
	switch buildcfg.GOARCH {
	default:
		fmt.Fprintf(os.Stderr, "link: unknown architecture %q\n", buildcfg.GOARCH)
		os.Exit(2)
	case "arm64":
		arch, theArch = arm64.Init()
	}
	ld.Main(arch, theArch)
}
