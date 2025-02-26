package main

import (
	"os"
	"runtime"
	"strings"
)

func usage() {
}

// commands records the available commands. 工具链指令
var commands = map[string]func(){
	"bootstrap": cmdbootstrap, // 构建go命令
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

	// 初始化方法
	xinit()
	// 对应指令执行
	xmain()
}

// The OS-specific main calls into the portable code here.
func xmain() {
	cmd := os.Args[1]
	os.Args = os.Args[1:] // for flag parsing during cmd
	if f, ok := commands[cmd]; ok {
		f()
	}
}
