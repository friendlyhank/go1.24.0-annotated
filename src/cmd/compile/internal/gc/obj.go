package gc

import (
	"cmd/compile/internal/base"
)

const (
	modeCompilerObj = 1 << iota // 编译器
	modeLinkerObj               // 链接器
)

func dumpobj() {
	dumpobj1(base.Flag.LowerO, modeCompilerObj|modeLinkerObj)
	return
}

func dumpobj1(outfile string, mode int) {
	if mode&modeCompilerObj != 0 {
	}

	if mode&modeLinkerObj != 0 {
	}
}

// 生成数据
func dumpdata() {

}
