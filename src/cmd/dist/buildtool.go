package main

import (
	"os"
)

const minBootstrap = "go1.22.6" // 最低可构建编译的go版本

// 尝试查找已构建的go版本
var tryDirs = []string{
	"sdk/" + minBootstrap,
	minBootstrap,
}

// bootstrapBuildTools 用于构建 Go 工具链，使用一个引导 Go 安装环境。
func bootstrapBuildTools() {
	goroot_bootstrap := os.Getenv("GOROOT_BOOTSTRAP")
	if goroot_bootstrap == "" {
		home := os.Getenv("HOME")
		goroot_bootstrap = pathf("%s/go1.4", home)
		for _, d := range tryDirs {
			if p := pathf("%s/%s", home, d); isdir(p) {
				goroot_bootstrap = p
			}
		}
	}

	ver := run(pathf("%s/bin", goroot_bootstrap), CheckExit, pathf("%s/bin/go", goroot_bootstrap), "env", "GOVERSION")
	// go env GOVERSION output like "go1.22.6\n" or "devel go1.24-ffb3e574 Thu Aug 29 20:16:26 2024 +0000\n".
	ver = ver[:len(ver)-1]
	xprintf("Building Go toolchain1 using %s.\n", goroot_bootstrap)

}
