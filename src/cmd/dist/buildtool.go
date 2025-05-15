package main

import (
	"go/version"
	"os"
)

const minBootstrap = "go1.22.6" // 最低可构建编译的go版本

// 尝试查找已构建的go版本
var tryDirs = []string{
	"sdk/" + minBootstrap,
	minBootstrap,
}

// bootstrapBuildTools 用于构建go的工具链
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
	if version.Compare(ver, version.Lang(minBootstrap)) > 0 && version.Compare(ver, minBootstrap) < 0 {
		fatalf("%s does not meet the minimum bootstrap requirement of %s or later", ver, minBootstrap)
	}
	xprintf("Building Go toolchain1 using %s.\n", goroot_bootstrap)

	// Use $GOROOT/pkg/bootstrap as the bootstrap workspace root.
	// We use a subdirectory of $GOROOT/pkg because that's the
	// space within $GOROOT where we store all generated objects.
	// We could use a temporary directory outside $GOROOT instead,
	// but it is easier to debug on failure if the files are in a known location.
	workspace := pathf("%s/pkg/bootstrap", goroot)
	//pkg/bootstrap/src/bootstrap
	base := pathf("%s/src/bootstrap", workspace)

	// Run Go bootstrap to build binaries.
	// Use the math_big_pure_go build tag to disable the assembly in math/big
	// which may contain unsupported instructions.
	// Use the purego build tag to disable other assembly code.
	// math_big_pure_go 禁用 math/big 包中的汇编代码，避免使用可能不兼容的指令集。
	// compiler_bootstrap：启用编译器引导专用的代码路径
	// purego：全局禁用其他汇编代码，确保全部用 Go 实现
	cmd := []string{
		pathf("%s/bin/go", goroot_bootstrap),
		"install",
		"-tags=math_big_pure_go compiler_bootstrap purego",
	}
	if vflag > 0 {
		cmd = append(cmd, "-v")
	}
	cmd = append(cmd, "bootstrap/cmd/...")
	run(base, ShowOutput|CheckExit, cmd...)
}
