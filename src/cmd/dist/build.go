package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// todo hank 这里可能需要总结
var (
	gohostarch string // 主机架构如amd64、arm64
	gohostos   string // 操作系统 如linux、"darwin" (macOS), "windows"
	goroot     string // go路径 如local/usr/go

	rebuildall bool // 重新构建所有依赖

	vflag int // verbosity // 版本参数标记
)

// xinit handles initialization of the various global state, like goroot and goarch. 初始化全局信息例如goroot
func xinit() {
	// todo hank 这里要重新调整
	goroot = "/Users/hank/go/src/github.com/friendlyhank/go1.24.0-annotated"

	b := os.Getenv("GOHOSTARCH")
	if b != "" {
		gohostarch = b
	}
}

// Remove trailing spaces. 删除尾部空格信息
func chomp(s string) string {
	return strings.TrimRight(s, " \t\r\n")
}

// findgoversion - 找到go版本信息
func findgoversion() string {
	// The $GOROOT/VERSION file takes priority, for distributions
	// without the source repo. 从go安装目录找到version文件做解析
	path := pathf("%s/VERSION", goroot)
	if isfile(path) {
		b := chomp(readfile(path))

		// Starting in Go 1.21 the VERSION file starts with the
		// version on a line by itself but then can contain other
		// metadata about the release, one item per line.
		if i := strings.Index(b, "\n"); i >= 0 {
			rest := b[i+1:]
			b = chomp(b[:i])
			for _, line := range strings.Split(rest, "\n") {
				f := strings.Fields(line)
				if len(f) == 0 {
					continue
				}
				switch f[0] {
				default:
					fatalf("VERSION: unexpected line: %s", line)
				case "time":
					if len(f) != 2 {
						fatalf("VERSION: unexpected time line: %s", line)
					}
					_, err := time.Parse(time.RFC3339, f[1])
					if err != nil {
						fatalf("VERSION: bad time: %s", err)
					}
				}
			}
		}

		// Commands such as "dist version > VERSION" will cause
		// the shell to create an empty VERSION file and set dist's
		// stdout to its fd. dist in turn looks at VERSION and uses
		// its content if available, which is empty at this point.
		// Only use the VERSION file if it is non-empty.
		if b != "" {
			return b
		}
	}

	// The $GOROOT/VERSION.cache file is a cache to avoid invoking
	// git every time we run this command. Unlike VERSION, it gets
	// deleted by the clean command. 从version缓存中读取
	path = pathf("%s/VERSION.cache", goroot)
	if isfile(path) {
		return chomp(readfile(path))
	}

	// 判断goroot是否是git仓库代码
	if !isGitRepo() {
		fatalf("FAILED: not a Git repo; must put a VERSION file in $GOROOT")
	}

	// Otherwise, use Git.
	//
	// Include 1.x base version, hash, and date in the version.
	//
	// Note that we lightly parse internal/goversion/goversion.go to
	// obtain the base version. We can't just import the package,
	// because cmd/dist is built with a bootstrap GOROOT which could
	// be an entirely different version of Go. We assume
	// that the file contains "const Version = <Integer>".
	goversionSource := readfile(pathf("%s/src/internal/goversion/goversion.go", goroot))
	// 通过正则获取版本信息
	m := regexp.MustCompile(`(?m)^const Version = (\d+)`).FindStringSubmatch(goversionSource)
	if m == nil {
		fatalf("internal/goversion/goversion.go does not contain 'const Version = ...'")
	}

	// 从git log中获取版本信息
	version := fmt.Sprintf("devel go1.%s-", m[1])
	version += chomp(run(goroot, CheckExit, "git", "log", "-n", "1", "--format=format:%h %cd", "HEAD"))

	// Cache version.  生成缓存版本信息
	writefile(version, path, 0)

	return version
}

// isGitRepo reports whether the working directory is inside a Git repository. 判断是否git仓库代码
func isGitRepo() bool {
	// NB: simply checking the exit code of `git rev-parse --git-dir` would
	// suffice here, but that requires deviating from the infrastructure
	// provided by `run`.
	gitDir := chomp(run(goroot, 0, "git", "rev-parse", "--git-dir"))
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(goroot, gitDir)
	}
	return isdir(gitDir)
}

// setup sets up the tree for the initial build. go 项目构建
func setup() {
	// Create bin directory. 创建bin目录
	if p := pathf("%s/bin", goroot); !isdir(p) {
		xmkdir(p)
	}

	// Create package directory. 创建pkg目录
	if p := pathf("%s/pkg", goroot); !isdir(p) {
		xmkdir(p)
	}

	goosGoarch := pathf("%s/pkg/%s_%s", goroot, gohostos, gohostarch)
	if rebuildall {
		xremoveall(goosGoarch)
	}
	xmkdirall(goosGoarch)
}

// clean - 构建go包先进行清理
func clean() {

}

// cmdbootstrap - 构建go工具
func cmdbootstrap() {

	// 重新构建所有
	if rebuildall {
		clean()
	}

	setup()

	bootstrapBuildTools()
}

// Version prints the Go version. 打印go版本信息
func cmdversion() {
	// 解析参数
	xflagparse(0)
	// 打印输出go版本信息
	xprintf("%s\n", findgoversion())
}
