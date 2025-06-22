package ld

import (
	"cmd/internal/objabi"
	"cmd/internal/quoted"
	"cmd/internal/sys"
	"flag"
	"fmt"
	"log"
	"os"
)

func init() {
	flag.Var(&flagExtld, "extld", "use `linker` when linking in external mode")
}

// Flags used by the linker. The exported flags are used by the architecture-specific packages.对应所有指令
var (
	flagOutfile = flag.String("o", "", "write output to `file`") // 输出文件

	flagExtld quoted.Flag
)

func Main(arch *sys.Arch, theArch Arch) {
	log.SetPrefix("link: ")
	log.SetFlags(0)

	ctxt := linknew(arch)

	objabi.Flagfn1("L", "add specified `directory` to library path", func(a string) {})

	objabi.Flagparse(usage)

	ctxt.hostlink()
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: link [options] main.o\n")
	objabi.Flagprint(os.Stderr)
	Exit(2)
}
