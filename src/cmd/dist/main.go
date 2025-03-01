package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
)

func usage() {
	xprintf(`usage: go tool dist [command]
Commands are:

banner                  print installation banner
bootstrap               rebuild everything
clean                   deletes all built files
env [-p]                print environment (-p: include $PATH)
install [dir]           install individual directory
list [-json] [-broken]  list all supported platforms
test [-h]               run Go test(s)
version                 print Go version

All commands take -v flags to emit extra information.
`)
	xexit(2)
}

// commands records the available commands. 工具链指令
var commands = map[string]func(){
	"bootstrap": cmdbootstrap, // 构建go命令
	"version":   cmdversion,   // 打印go版本信息
}

func main() {

	gohostos = runtime.GOOS

	// 获取主机架构
	if gohostarch == "" {
		// 执行uname -m 执行架构
		out := run("", CheckExit, "uname", "-m")
		outAll := run("", CheckExit, "uname", "-a")
		switch {
		case strings.Contains(outAll, "RELEASE_ARM64"):
			// MacOS prints
			// Darwin p1.local 21.1.0 Darwin Kernel Version 21.1.0: Wed Oct 13 17:33:01 PDT 2021; root:xnu-8019.41.5~1/RELEASE_ARM64_T6000 x86_64
			// on ARM64 laptops when there is an x86 parent in the
			// process tree. Look for the RELEASE_ARM64 to avoid being
			// confused into building an x86 toolchain.
			gohostarch = "arm64"
		case strings.Contains(out, "x86_64"), strings.Contains(out, "amd64"):
			gohostarch = "amd64"
		case strings.Contains(out, "86"):
			gohostarch = "386"
			if gohostos == "darwin" {
				// Even on 64-bit platform, some versions of macOS uname -m prints i386.
				// We don't support any of the OS X versions that run on 32-bit-only hardware anymore.
				gohostarch = "amd64"
			}
		default:
			fatalf("unknown architecture: %s", out)
		}
	}

	// 初始化环境变量信息
	xinit()
	// 对应指令执行
	xmain()
	xexit(0)
}

// The OS-specific main calls into the portable code here.
func xmain() {
	if len(os.Args) < 2 {
		usage()
	}
	cmd := os.Args[1]
	os.Args = os.Args[1:] // for flag parsing during cmd
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: go tool dist %s [options]\n", cmd)
		flag.PrintDefaults()
		os.Exit(2)
	}
	if f, ok := commands[cmd]; ok {
		f()
	} else {
		xprintf("unknown command %s\n", cmd)
		usage()
	}
}
