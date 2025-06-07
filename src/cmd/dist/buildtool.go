package main

import (
	"go/version"
	"os"
	"path/filepath"
	"strings"
)

/*
 * 工具链构建文件
 */

const minBootstrap = "go1.22.6" // 最低可构建编译的go版本

// bootstrapDirs is a list of directories holding code that must be
// compiled with the Go bootstrap toolchain to produce the bootstrapTargets.
// All directories in this list are relative to and must be below $GOROOT/src.
//
// The list has two kinds of entries: names beginning with cmd/ with
// no other slashes, which are commands, and other paths, which are packages
// supporting the commands. Packages in the standard library can be listed
// if a newer copy needs to be substituted for the Go bootstrap copy when used
// by the command packages. Paths ending with /... automatically
// include all packages within subdirectories as well.
// These will be imported during bootstrap as bootstrap/name, like bootstrap/math/big.
var bootstrapDirs = []string{
	"cmd/compile",
}

// 尝试查找已构建的go版本
var tryDirs = []string{
	"sdk/" + minBootstrap,
	minBootstrap,
}

/*
 *bootstrapBuildTools 用于构建go的工具链
 *为了模块的隔离，工具链的编译会在pkg/bootstrap/src/bootstrap目录下完成
 */
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
	// pkg/bootstrap/src/bootstrap
	base := pathf("%s/src/bootstrap", workspace)
	xmkdirall(base)

	minBootstrapVers := requiredBootstrapVersion(goModVersion()) // require the minimum required go version to build this go version in the go.mod file
	// 生成对应/pkg/bootstrap/src/bootstrap/的mod文件
	writefile("module bootstrap\ngo "+minBootstrapVers+"\n", pathf("%s/%s", base, "go.mod"), 0)
	// 将src/cmd文件拷贝到/pkg/bootstrap/src/bootstrap/目录下,主要为了模块的隔离
	for _, dir := range bootstrapDirs {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fatalf("walking bootstrap dirs failed: %v: %v", path, err)
			}

			//name := filepath.Base(path)
			src := pathf("%s/src/%s", goroot, path)
			dst := pathf("%s/%s", base, path)

			// 如果是目录，则创建目标目录
			if info.IsDir() {
				// 创建目标目录
				xmkdirall(dst)
				return nil
			}

			// 重写文件到bootstrap路径
			text := bootstrapRewriteFile(src)
			writefile(text, dst, 0)
			return nil
		})
	}

	// 设置生成工具类的环境
	os.Setenv("GOROOT", goroot_bootstrap)
	os.Setenv("GOPATH", workspace)
	os.Setenv("GOBIN", "")

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
	}
	if vflag > 0 {
		cmd = append(cmd, "-v")
	}
	cmd = append(cmd, "bootstrap/cmd/...")
	run(base, ShowOutput|CheckExit, cmd...)

	// Copy binaries into tool binary directory.
	// 将二进制文件拷贝到工具类目录下
	for _, name := range bootstrapDirs {
		if !strings.HasPrefix(name, "cmd/") {
			continue
		}
		name = name[len("cmd/"):]
		if !strings.Contains(name, "/") {
			copyfile(pathf("%s/%s%s", tooldir, name, exe), pathf("%s/bin/%s%s", workspace, name, exe), writeExec)
		}
	}

	if vflag > 0 {
		xprintf("\n")
	}
}

// bootstrapRewriteFile - 重写工具链文件到/pkg/bootstrap/src/bootstrap/cmd目录下
func bootstrapRewriteFile(srcFile string) string {
	return bootstrapFixImports(srcFile)
}

func bootstrapFixImports(srcFile string) string {
	text := readfile(srcFile)
	lines := strings.SplitAfter(text, "\n")
	return strings.Join(lines, "")
}
