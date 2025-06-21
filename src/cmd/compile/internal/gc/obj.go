package gc

import (
	"cmd/compile/internal/base"
	"cmd/internal/bio"
	"fmt"
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
	bout, err := bio.Create(outfile)
	if err != nil {
		fmt.Printf("can't create %s: %v\n", outfile, err)
		base.ErrorExit()
	}
	defer bout.Close()

	if mode&modeCompilerObj != 0 {
	}

	if mode&modeLinkerObj != 0 {
	}
}

// 生成数据
func dumpdata() {

}
