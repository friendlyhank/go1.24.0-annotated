package main

import (
	"os"
)

// bootstrapBuildTools 用于构建 Go 工具链，使用一个引导 Go 安装环境。
func bootstrapBuildTools() {
	goroot_bootstrap := os.Getenv("GOROOT_BOOTSTRAP")

	ver := run(pathf("%s/bin", goroot_bootstrap), CheckExit, pathf("%s/bin/go", goroot_bootstrap), "env", "GOVERSION")

}
