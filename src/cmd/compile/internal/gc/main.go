package gc

import (
	"cmd/compile/internal/base"
	"cmd/compile/internal/ssagen"
	"cmd/internal/obj"
)

// Main - compile生成主方法
func Main(archInit func(*ssagen.ArchInfo)) {
	base.Ctxt = obj.Linknew(ssagen.Arch.LinkArch)
	// 解析参数
	base.ParseFlags()

	dumpdata()

	// 生成文件
	dumpobj()
}
