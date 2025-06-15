package gc

import (
	"cmd/compile/internal/base"
	"cmd/compile/internal/ssagen"
)

// Main - compile生成主方法
func Main(archInit func(*ssagen.ArchInfo)) {
	base.ParseFlags()

	dumpdata()

	dumpobj()
}
