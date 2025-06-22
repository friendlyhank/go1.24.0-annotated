package arm64

import (
	"cmd/internal/sys"
	"cmd/link/internal/ld"
)

func Init() (*sys.Arch, ld.Arch) {
	arch := sys.ArchARM64
	theArch := ld.Arch{}

	return arch, theArch
}
