package main

import (
	"go/version"
	"os"
	"path/filepath"
	"regexp"
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
	"cmd/internal/bio",
	"cmd/internal/obj/...",
	"cmd/internal/objabi",
	"cmd/internal/quoted",
	"cmd/internal/sys",
	"cmd/link",
	"cmd/link/internal/...",
	"internal/buildcfg",
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
		dir = strings.TrimSuffix(dir, "/...")
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

	// Set up environment for invoking Go bootstrap toolchains go command.
	// GOROOT points at Go bootstrap GOROOT,
	// GOPATH points at our bootstrap workspace,
	// GOBIN is empty, so that binaries are installed to GOPATH/bin,
	// and GOOS, GOHOSTOS, GOARCH, and GOHOSTOS are empty,
	// so that Go bootstrap toolchain builds whatever kind of binary it knows how to build.
	// Restore GOROOT, GOPATH, and GOBIN when done.
	// Don't bother with GOOS, GOHOSTOS, GOARCH, and GOHOSTARCH,
	// because setup will take care of those when bootstrapBuildTools returns.
	/*
		注意：
		1.这里的作用首先是设置GOPATH创建独立的沙盒环境，构建隔离
		2.一个是编译安装的环境，构建完成后会恢复原始的GOROOT GOPATH GOBIN环境变量(如果不恢复，会影响到比如包的引用)
		3.如何很好的隔离环境编译工具类
	*/
	defer os.Setenv("GOROOT", os.Getenv("GOROOT"))
	os.Setenv("GOROOT", goroot_bootstrap)

	defer os.Setenv("GOPATH", os.Getenv("GOPATH"))
	os.Setenv("GOPATH", workspace)

	defer os.Setenv("GOBIN", os.Getenv("GOBIN"))
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

var (
	importRE      = regexp.MustCompile(`\Aimport\s+(\.|[A-Za-z0-9_]+)?\s*"([^"]+)"\s*(//.*)?\n\z`)
	importBlockRE = regexp.MustCompile(`\A\s*(?:(\.|[A-Za-z0-9_]+)?\s*"([^"]+)")?\s*(//.*)?\n\z`)
)

// importRE 用于解析独立的导入语句，如：import "fmt"
// importBlockRE 用于解析导入块，如：import (...)

func bootstrapFixImports(srcFile string) string {
	text := readfile(srcFile)
	lines := strings.SplitAfter(text, "\n")
	inBlock := false // 是否导入块
	for i, line := range lines {
		if strings.HasPrefix(line, "import (") {
			inBlock = true
			continue
		}
		if inBlock && strings.HasPrefix(line, ")") {
			inBlock = false
			continue
		}

		var m []string
		if !inBlock {
			if !strings.HasPrefix(line, "import ") {
				continue
			}
			m = importRE.FindStringSubmatch(line)
			if m == nil {
				fatalf("%s:%d: invalid import declaration: %q", srcFile, i+1, line)
			}
		} else {
			m = importBlockRE.FindStringSubmatch(line)
			if m == nil {
				fatalf("%s:%d: invalid import block line", srcFile, i+1)
			}
			if m[2] == "" {
				continue
			}
		}

		path := m[2]
		// 替换成bootstrap的工作路径
		if strings.HasPrefix(path, "cmd/") {
			path = "bootstrap/" + path
		} else {
			for _, dir := range bootstrapDirs {
				if path == dir {
					path = "bootstrap/" + dir
					break
				}
			}
		}

		// Otherwise, reject direct imports of internal packages,
		// since that implies knowledge of internal details that might
		// change from one bootstrap toolchain to the next.
		// There are many internal packages that are listed in
		// bootstrapDirs and made into bootstrap copies based on the
		// current repo's source code. Those are fine; this is catching
		// references to internal packages in the older bootstrap toolchain.
		if strings.HasPrefix(path, "internal/") {
			fatalf("%s:%d: bootstrap-copied source file cannot import %s", srcFile, i+1, path)
		}
		// 替换最终的构建路径
		if path != m[2] {
			lines[i] = strings.ReplaceAll(line, `"`+m[2]+`"`, `"`+path+`"`)
		}
	}

	lines[0] = generatedHeader + "// This is a bootstrap copy of " + srcFile + "\n\n//line " + srcFile + ":1\n" + lines[0]

	return strings.Join(lines, "")
}
