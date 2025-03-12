package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// todo hank 这里可能需要总结
var (
	goarch      string // 表示目标架构 如amd64、arm64
	gorootBin   string // 需要目标安装的bin路径
	gorootBinGo string // 需要目标的go包文件路径
	gohostarch  string // 表示当前编译主机架构如amd64、arm64
	gohostos    string // 表示当前编译运行的操作系统 如linux、"darwin" (macOS), "windows"
	goos        string // 表示目标操作系统的类型，即你希望编译生成的程序运行的操作系统(允许交叉编译)
	goroot      string // 需要目标安装的go路径 如local/usr/go
	workdir     string // 这个不确定具体做啥
	tooldir     string // 工具库路径(todo hank 这个还不知道具体做啥)
	exe         string // window的程序后缀

	rebuildall bool // 重新构建所有依赖

	vflag int // verbosity // 版本参数标记
)

// xinit handles initialization of the various global state, like goroot and goarch. 初始化全局信息例如goroot
func xinit() {
	// todo hank 这里要重新调整
	goroot = "/Users/hank/go/src/github.com/friendlyhank/go1.24.0-annotated"

	gorootBin = pathf("%s/bin", goroot)

	// Don't run just 'go' because the build infrastructure
	// runs cmd/dist inside go/bin often, and on Windows
	// it will be found in the current directory and refuse to exec.
	// All exec calls rewrite "go" into gorootBinGo.
	gorootBinGo = pathf("%s/bin/go", goroot)

	b := os.Getenv("GOHOSTARCH")
	if b != "" {
		gohostarch = b
	}

	b = os.Getenv("GOARCH")
	if b == "" {
		b = gohostarch
	}
	goarch = b

	// 编译go文件生成临时的工作目录
	workdir = xworkdir()
	// 程序停止时销毁临时的工作目录
	xatexit(rmworkdir) // todo hank 临时关闭

	// 工具地址
	tooldir = pathf("%s/pkg/tool/%s_%s", goroot, gohostos, gohostarch)
}

// rmworkdir deletes the work directory.删除临时的工作目录
func rmworkdir() {
	if vflag > 1 {
		errprintf("rm -rf %s\n", workdir)
	}
	xremoveall(workdir)
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

	// Create tool directory.
	// We keep it in pkg/, just like the object directory above.
	// todo hank 暂时先关闭
	//if rebuildall {
	//	xremoveall(tooldir)
	//}
	//xmkdirall(tooldir)
}

// depsuffix records the allowed suffixes for source files. 查找允许安装的前缀文件
var depsuffix = []string{
	".s",
	".go",
}

// installed maps from a dir name (as given to install) to a chan
// closed when the dir's package is installed.
var installed = make(map[string]chan struct{})
var installedMu sync.Mutex

// 同目录不能并发安装 todo hank 这种写法可以参考学习
func install(dir string) {
	<-startInstall(dir)
}

// startInstall - 用chan方式开始安装程序
func startInstall(dir string) chan struct{} {
	installedMu.Lock()
	ch := installed[dir]
	if ch == nil {
		ch = make(chan struct{})
		installed[dir] = ch
		go runInstall(dir, ch)
	}
	installedMu.Unlock()
	return ch
}

// runInstall installs the library, package, or binary associated with pkg,
// runInstall 安装与 pkg 关联的库、包或二进制文件，
// which is relative to $GOROOT/src.
// pkg 是相对于 $GOROOT/src 的路径。
func runInstall(pkg string, ch chan struct{}) {

	defer close(ch)

	// unsafe 包不需要安装
	if pkg == "unsafe" {
		return
	}

	if vflag > 0 {
		errprintf("%s\n", pkg)
	}

	// 创建安装的工作目录
	workdir := pathf("%s/%s", workdir, pkg)
	xmkdirall(workdir)

	// 清理列表，用于存储需要清理的文件
	var clean []string
	defer func() {
		for _, name := range clean {
			xremove(name)
		}
	}()

	// dir = full path to pkg. 获取完整的包路径
	dir := pathf("%s/src/%s", goroot, pkg)
	name := filepath.Base(dir)

	// ispkg predicts whether the package should be linked as a binary, based
	// on the name. There should be no "main" packages in vendor, since
	// 'go mod vendor' will only copy imported packages there.
	// 判断包是否应该被链接为二进制文件
	// 如果包路径不以 cmd/ 开头，或者包含 /internal/ 或 /vendor/，则视为库包
	ispkg := !strings.HasPrefix(pkg, "cmd/") || strings.Contains(pkg, "/internal/") || strings.Contains(pkg, "/vendor/")

	var (
		link      []string
		targ      int
		ispackcmd bool
	)

	if ispkg {
		// 如果是库包，使用 pack 命令打包
	} else {
		// Go command.
		elem := name
		// 如果安装的是cmd/go
		if elem == "go" {
			elem = "go_bootstrap"
		}
		link = []string{pathf("%s/link", tooldir)}
		link = append(link, "-extld=")                                                          // 指定不使用外部链接器
		link = append(link, "-L="+pathf("%s/pkg/obj/go-bootstrap/%s_%s", goroot, goos, goarch)) // 指定链接器链接对象文件路径
		link = append(link, "-o", pathf("%s/%s%s", tooldir, elem, exe))                         // 二进制文件输出路径
		targ = len(link) - 1
	}

	// 读取要安装目录下的文件信息
	files := xreaddir(dir)

	// Convert to absolute paths.转换为绝对路径
	for i, p := range files {
		if !filepath.IsAbs(p) {
			files[i] = pathf("%s/%s", dir, p)
		}
	}

	var gofiles []string
	files = filter(files, func(p string) bool {
		for _, suf := range depsuffix {
			if strings.HasSuffix(p, suf) {
				goto ok
			}
		}
		return false
	ok:
		if strings.HasSuffix(p, ".go") {
			gofiles = append(gofiles, p)
		}
		return true
	})

	// If there are no files to compile, we're done.
	if len(files) == 0 {
		return
	}

	// Resolve imported packages to actual package paths. 将导入的包解析为实际的包路径
	//  确保他们已安装
	// Make sure they're installed.
	importMap := make(map[string]string)
	for _, p := range gofiles {
		// 读取要编译安装的go文件导入包的信息
		for _, imp := range readimports(p) {
			importMap[imp] = resolveVendor(imp, dir)
		}
	}
	sortedImports := make([]string, 0, len(importMap))
	for imp := range importMap {
		sortedImports = append(sortedImports, imp)
	}
	sort.Strings(sortedImports)

	// Build an importcfg file for the compiler. 构造用于编译的importcfg文件
	buf := &bytes.Buffer{}
	for _, imp := range sortedImports {
		if imp == "unsafe" {
			continue
		}
		dep := importMap[imp]
		fmt.Fprintf(buf, "packagefile %s=%s\n", dep, packagefile(dep))
	}
	importcfg := pathf("%s/importcfg", workdir)
	if err := os.WriteFile(importcfg, buf.Bytes(), 0666); err != nil {
		fatalf("cannot write importcfg file: %v", err)
	}

	// The next loop will compile individual non-Go files.
	// Hand the Go files to the compiler en masse.
	// For packages containing assembly, this writes go_asm.h, which
	// the assembly files will need.
	pkgName := pkg
	if strings.HasPrefix(pkg, "cmd/") && strings.Count(pkg, "/") == 1 {
		pkgName = "main"
	}
	b := pathf("%s/_go_.a", workdir)
	clean = append(clean, b)
	if !ispackcmd {
		link = append(link, b)
	}

	// Compile Go code. 编译go代码
	compile := []string{pathf("%s/compile", tooldir), "-std", "-pack", "-o", b, "-p", pkgName, "-importcfg", importcfg}
	compile = append(compile, gofiles...)
	var wg sync.WaitGroup
	// We use bgrun and immediately wait for it instead of calling run() synchronously.
	// This executes all jobs through the bgwork channel and allows the process
	// to exit cleanly in case an error occurs.
	bgrun(&wg, dir, compile...)
	bgwait(&wg)

	// Remove target before writing it. 删除旧的目标文件
	// 执行link相关指令
	xremove(link[targ])
	bgrun(&wg, "", link...)
	bgwait(&wg)
}

// packagefile returns the path to a compiled .a file for the given package
// path. Paths may need to be resolved with resolveVendor first.
// 对应静态库包(其实就是go导入的包)
func packagefile(pkg string) string {
	return pathf("%s/pkg/obj/go-bootstrap/%s_%s/%s.a", goroot, goos, goarch, pkg)
}

// clean - 构建go包先进行清理
func clean() {

}

var (
	timeLogEnabled = os.Getenv("GOBUILDTIMELOGFILE") != "" //  判断环境变量是否设置耗时日志路径
	timeLogMu      sync.Mutex
	timeLogFile    *os.File  // 耗时日志
	timeLogStart   time.Time // 耗时日志开始时间
)

// timelog 耗时日志
func timelog(op, name string) {
	// 是否开启耗时日志
	if !timeLogEnabled {
		return
	}
	timeLogMu.Lock()
	defer timeLogMu.Unlock()
	if timeLogFile == nil {
		f, err := os.OpenFile(os.Getenv("GOBUILDTIMELOGFILE"), os.O_RDWR|os.O_APPEND, 0666)
		if err != nil {
			log.Fatal(err)
		}
		// 从日志找到开始时间日志
		buf := make([]byte, 100)
		n, _ := f.Read(buf)
		s := string(buf[:n])
		if i := strings.Index(s, "\n"); i >= 0 {
			s = s[:i]
		}
		i := strings.Index(s, " start")
		if i < 0 {
			log.Fatalf("time log %s does not begin with start line", os.Getenv("GOBUILDTIMELOGFILE"))
		}
		t, err := time.Parse(time.UnixDate, s[:i])
		if err != nil {
			log.Fatalf("cannot parse time log line %q: %v", s, err)
		}
		timeLogStart = t
		timeLogFile = f
	}
	t := time.Now()
	fmt.Fprintf(timeLogFile, "%s %+.1fs %s %s\n", t.Format(time.UnixDate), t.Sub(timeLogStart).Seconds(), op, name)
}

// toolenv - 工具构建的环境
func toolenv() []string {
	var env []string
	return env
}

// 工具链包
var toolchain = []string{"cmd/asm", "cmd/cgo", "cmd/compile", "cmd/link", "cmd/preprofile"}

// cmdbootstrap - 构建go工具
func cmdbootstrap() {
	timelog("start", "dist bootstrap")
	defer timelog("end", "dist bootstrap")

	// 重新构建所有
	var debug, distpack, force, noBanner, noClean bool
	// rebuildall 标志用于控制是否重建所有内容
	flag.BoolVar(&rebuildall, "a", rebuildall, "rebuild all")
	// debug 标志用于控制是否启用引导过程的调试
	flag.BoolVar(&debug, "d", debug, "enable debugging of bootstrap process")
	// distpack 标志用于控制是否将分发文件写入 pkg/distpack
	flag.BoolVar(&distpack, "distpack", distpack, "write distribution files to pkg/distpack")
	// force 标志用于控制是否强制构建，即使端口被标记为损坏
	flag.BoolVar(&force, "force", force, "build even if the port is marked as broken")
	// noBanner 标志用于控制是否不打印横幅
	flag.BoolVar(&noBanner, "no-banner", noBanner, "do not print banner")
	// noClean 标志用于控制是否打印过时警告
	flag.BoolVar(&noClean, "no-clean", noClean, "print deprecation warning")

	xflagparse(0)

	// 重新构建所有
	if rebuildall {
		clean()
	}

	setup()

	// 构建步骤1
	timelog("build", "toolchain1")

	bootstrapBuildTools()

	goos = gohostos

	// For the main bootstrap, building for host os/arch.
	timelog("build", "go_bootstrap")
	xprintf("Building Go bootstrap cmd/go (go_bootstrap) using Go toolchain1.\n")
	// 安装src/cmd/go 这里会生成go_bootstrap二进制文件，go_bootstrap最终生成bin/go
	install("cmd/go")
	if vflag > 0 {
		xprintf("\n")
	}

	goBootstrap := pathf("%s/go_bootstrap", tooldir)

	timelog("build", "toolchain2")
	if vflag > 0 {
		xprintf("\n")
	}
	xprintf("Building Go toolchain2 using go_bootstrap and Go toolchain1.\n")

	timelog("build", "toolchain3")
	if vflag > 0 {
		xprintf("\n")
	}
	xprintf("Building Go toolchain3 using go_bootstrap and Go toolchain2.\n")

	// 首先会生成goBootstrap二进制文件,然后执行这个二进制文件生成goBinary
	goInstall(toolenv(), goBootstrap, "cmd")

	// Check that there are no new files in $GOROOT/bin other than
	// go and gofmt and $GOOS_$GOARCH (target bin when cross-compiling).
	// 检查新go文件是否生成成功
	binFiles, err := filepath.Glob(pathf("%s/bin/*", goroot))
	if err != nil {
		fatalf("glob: %v", err)
	}
	for _, f := range binFiles {
		if gohostos == "darwin" && filepath.Base(f) == ".DS_Store" {
			continue // unfortunate but not unexpected
		}
	}

	// 需要横幅
	if !noBanner {
		banner()
	}
}

func goInstall(env []string, goBinary string, args ...string) {
	goCmd(env, goBinary, "install", args...)
}

func goCmd(env []string, goBinary string, cmd string, args ...string) {
	goCmd := []string{goBinary, cmd}
	runEnv(workdir, ShowOutput|CheckExit, env, append(goCmd, args...)...)
}

// banner - 构建包成功，打印横幅
func banner() {
	if vflag > 0 {
		xprintf("\n")
	}
	xprintf("---\n")
	xprintf("Installed Go for %s/%s in %s\n", goos, goarch, goroot)
	xprintf("Installed commands in %s\n", gorootBin)
}

// Version prints the Go version. 打印go版本信息
func cmdversion() {
	// 解析参数
	xflagparse(0)
	// 打印输出go版本信息
	xprintf("%s\n", findgoversion())
}
