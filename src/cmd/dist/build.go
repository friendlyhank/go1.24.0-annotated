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
	"strconv"
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
	tooldir     string // 工具库路径(编译好的各种工具地址，这个路径要关注，构建go包不可缺少的)
	exe         string // window的程序后缀

	rebuildall bool // 重新构建所有依赖

	vflag int // verbosity // 版本参数标记(用于标记日志打印等级)
)

// xinit handles initialization of the various global state, like goroot and goarch. 初始化全局信息例如goroot
func xinit() {
	b := os.Getenv("GOROOT")
	if b == "" {
		fatalf("$GOROOT must be set")
	}
	goroot = filepath.Clean(b)
	gorootBin = pathf("%s/bin", goroot)

	// Don't run just 'go' because the build infrastructure
	// runs cmd/dist inside go/bin often, and on Windows
	// it will be found in the current directory and refuse to exec.
	// All exec calls rewrite "go" into gorootBinGo.
	gorootBinGo = pathf("%s/bin/go", goroot)

	b = os.Getenv("GOHOSTARCH")
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
	xatexit(rmworkdir)

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

// goModVersion returns the go version declared in src/go.mod. This is the
// go version to use in the go.mod building go_bootstrap, toolchain2, and toolchain3.
// (toolchain1 must be built with requiredBootstrapVersion(goModVersion))
// 根据src/go.mod的文件将会生成新的go.mod文件被用于go_bootstrap, toolchain2, and toolchain3工具链的生成
func goModVersion() string {
	goMod := readfile(pathf("%s/src/go.mod", goroot))
	m := regexp.MustCompile(`(?m)^go (1.\d+)$`).FindStringSubmatch(goMod)
	if m == nil {
		fatalf("std go.mod does not contain go 1.X")
	}
	return m[1]
}

// requiredBootstrapVersion 用于编译工具链的版本
func requiredBootstrapVersion(v string) string {
	minorstr, ok := strings.CutPrefix(v, "1.")
	if !ok {
		fatalf("go version %q in go.mod does not start with %q", v, "1.")
	}
	minor, err := strconv.Atoi(minorstr)
	if err != nil {
		fatalf("invalid go version minor component %q: %v", minorstr, err)
	}
	// Per go.dev/doc/install/source, for N >= 22, Go version 1.N will require a Go 1.M compiler,
	// where M is N-2 rounded down to an even number. Example: Go 1.24 and 1.25 require Go 1.22.
	requiredMinor := minor - 2 - minor%2
	return "1." + strconv.Itoa(requiredMinor)
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

// setup sets up the tree for the initial build
// 初始化构建，主要为目录创建、移除
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
	xatexit(func() {
		if files := xreaddir(goosGoarch); len(files) == 0 {
			xremove(goosGoarch)
		}
	})

	// Create object directory.
	// We used to use it for C objects.
	// Now we use it for the build cache, to separate dist's cache
	// from any other cache the user might have, and for the location
	// to build the bootstrap versions of the standard library.
	obj := pathf("%s/pkg/obj", goroot)
	if !isdir(obj) {
		xmkdir(obj)
	}
	//xatexit(func() { xremove(obj) })

	// Create directory for bootstrap versions of standard library .a files.
	// 标准库.a文件的路径(重要)
	objGoBootstrap := pathf("%s/pkg/obj/go-bootstrap", goroot)
	if rebuildall {
		xremoveall(objGoBootstrap)
	}
	xmkdirall(objGoBootstrap)

	// Create tool directory.
	// We keep it in pkg/, just like the object directory above.
	// todo hank 临时先关闭 工具类路径删除
	if rebuildall {
		xremoveall(tooldir)
	}
	xmkdirall(tooldir)
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
	ch := installed[dir] // 这里写的挺妙的，每个安装导入的包可能重复，这里直接去重了
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
// 这里安装的包是重点(这个要完整理解)
func runInstall(pkg string, ch chan struct{}) {
	defer close(ch)

	// unsafe 包不需要安装
	if pkg == "unsafe" {
		return
	}

	if vflag > 0 {
		errprintf("%s\n", pkg)
	}

	// 创建安装的工作目录(使用临时目录)
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
		// 如果是库包，使用 pack 命令打包(主要为非cmd目录下的文件)
		ispackcmd = true
		link = []string{"pack", packagefile(pkg)}
		targ = len(link) - 1
		xmkdirall(filepath.Dir(link[targ]))
	} else {
		// 这里主要是编译cmd目录下的文件
		//Go command.
		elem := name
		// 如果安装的是cmd/go
		if elem == "go" {
			elem = "go_bootstrap"
		}
		link = []string{pathf("%s/link", tooldir)}
		link = append(link, "-extld=")
		link = append(link, "-L="+pathf("%s/pkg/obj/go-bootstrap/%s_%s", goroot, goos, goarch)) // 指定链接器链接对象文件路径
		link = append(link, "-o", pathf("%s/%s%s", tooldir, elem, exe))                         // 二进制文件输出路径
		targ = len(link) - 1
	}

	// 读取要安装目录下的文件信息
	files := xreaddir(dir)

	// Remove files beginning with . or _,
	// which are likely to be editor temporary files.
	// This is the same heuristic build.ScanDir uses.
	// There do exist real C files beginning with _,
	// so limit that check to just Go files.
	// 过滤源文件列表，移除编辑器临时文件
	// 保留规则：
	// 1. 排除以 '.' 开头的隐藏文件（如 .git/.DS_Store）
	// 2. 排除以 '_' 开头的 Go 源文件（如 _test.go），但保留非 Go 文件
	files = filter(files, func(p string) bool {
		return !strings.HasPrefix(p, ".") && (!strings.HasPrefix(p, "_") || !strings.HasSuffix(p, ".go"))
	})
	files = uniq(files)

	// Convert to absolute paths.转换为绝对路径(make.bash编译的，所以只要转成绝对路径就可以了	)
	for i, p := range files {
		if !filepath.IsAbs(p) {
			files[i] = pathf("%s/%s", dir, p)
		}
	}

	var gofiles, sfiles []string
	files = filter(files, func(p string) bool {
		for _, suf := range depsuffix {
			if strings.HasSuffix(p, suf) {
				goto ok
			}
		}
		return false
	ok:
		if !strings.HasSuffix(p, ".a") && !shouldbuild(p, pkg) {
			return false
		}
		if strings.HasSuffix(p, ".go") {
			gofiles = append(gofiles, p)
		} else if strings.HasSuffix(p, ".s") {
			sfiles = append(sfiles, p)
		}
		return true
	})

	// todo hank 需要调试，暂时先不删除
	for _, p := range gofiles {
		println("g文件", p)
	}
	for _, s := range sfiles {
		println("s文件", s)
	}

	// If there are no files to compile, we're done.
	if len(files) == 0 {
		return
	}

	// For package runtime, copy some files into the work space.
	//构建runtime需要用到汇编，需要拷贝对应的汇编文件
	if pkg == "runtime" {
		xmkdirall(pathf("%s/pkg/include", goroot))
		copyfile(pathf("%s/pkg/include/textflag.h", goroot),
			pathf("%s/src/runtime/textflag.h", goroot), 0)
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

	// 这两步有其精妙之处，先异步安装，然后同步获得结果
	// 先构建需要导入的依赖包，安装编译成.a文件
	for _, dep := range importMap {
		startInstall(dep)
	}
	for _, dep := range importMap {
		install(dep)
	}

	// ba
	asmArgs := []string{
		pathf("%s/asm", tooldir),
		"-I", workdir,
		"-I", pathf("%s/pkg/include", goroot),
		"-D", "GOOS_" + goos,
		"-D", "GOARCH_" + goarch,
		"-D", "GOOS_GOARCH_" + goos + "_" + goarch,
		"-p", pkg,
	}
	goasmh := pathf("%s/go_asm.h", workdir)

	// Collect symabis from assembly code.处理汇编代码处理(这里主要用于编译库包的时候汇编代码能正常被go代码使用)
	var symabis string
	if len(sfiles) > 0 {
		symabis = pathf("%s/symabis", workdir)
		var wg sync.WaitGroup
		asmabis := append(asmArgs[:len(asmArgs):len(asmArgs)], "-gensymabis", "-o", symabis)
		asmabis = append(asmabis, sfiles...)
		if err := os.WriteFile(goasmh, nil, 0666); err != nil {
			fatalf("cannot write empty go_asm.h: %s", err)
		}
		bgrun(&wg, dir, asmabis...)
		bgwait(&wg)
	}

	//Build an importcfg file for the compiler. 构造用于编译的importcfg文件
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

	// todo hank 临时增加
	println("========打印包引用 start========")
	println("构建包地址", pkg)
	println("包引用", string(buf.Bytes()))
	println("========打印包引用 end========")

	var archive string
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
	} else {
		archive = b
	}

	// Compile Go code. 编译go代码
	compile := []string{pathf("%s/compile", tooldir), "-std", "-pack", "-o", b, "-p", pkgName, "-importcfg", importcfg}
	if len(sfiles) > 0 {
		compile = append(compile, "-asmhdr", goasmh)
	}
	if symabis != "" {
		compile = append(compile, "-symabis", symabis)
	}
	compile = append(compile, gofiles...)
	var wg sync.WaitGroup
	// We use bgrun and immediately wait for it instead of calling run() synchronously.
	// This executes all jobs through the bgwork channel and allows the process
	// to exit cleanly in case an error occurs.
	bgrun(&wg, dir, compile...)
	bgwait(&wg)

	// Compile the files. 将汇编代码处理成可执行的.o文件，可以被链接器使用
	for _, p := range sfiles {
		// Assembly file for a Go package.
		compile := asmArgs[:len(asmArgs):len(asmArgs)]

		doclean := true
		b := pathf("%s/%s", workdir, filepath.Base(p))

		// Change the last character of the output file (which was c or s).
		b = b[:len(b)-1] + "o"
		compile = append(compile, "-o", b, p)
		bgrun(&wg, dir, compile...)

		link = append(link, b)
		if doclean {
			clean = append(clean, b)
		}
	}
	bgwait(&wg)

	// 如果是库包，将文件从临时目录拷贝到pkg/obj/go-bootstrap,将.a和.o汇编文件以指定格式打包成.a,主要生成.a文件
	if ispackcmd {
		xremove(link[targ])
		dopack(link[targ], archive, link[targ+1:])
		return
	}

	//
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

// shouldbuild reports whether we should build this file.
// It applies the same rules that are used with context tags
// in package go/build, except it's less picky about the order
// of GOOS and GOARCH.
// We also allow the special tag cmd_go_bootstrap.
// See ../go/bootstrap.go and package go/build.
// 判断文件是否可以构建
func shouldbuild(file, pkg string) bool {
	name := filepath.Base(file)

	// Omit test files.
	// 测试文件不参与
	if strings.Contains(name, "_test") {
		return false
	}

	// Check file contents for //go:build lines.
	for _, p := range strings.Split(readfile(file), "\n") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		code := p
		i := strings.Index(code, "//")
		if i > 0 {
			code = strings.TrimSpace(code[:i])
		}
		if code == "package documentation" {
			return false
		}
	}

	return true
}

// copyfile copies the file src to dst, via memory (so only good for small files).拷贝文件
func copyfile(dst, src string, flag int) {
	if vflag > 1 {
		errprintf("cp %s %s\n", src, dst)
	}
	writefile(readfile(src), dst, flag)
}

// dopack copies the package src to dst,
// appending the files listed in extra.
// The archive format is the traditional Unix ar format.
// 将库包从临时目录拷贝到pkg/obj/go-bootstrap目录下,将.a和.o汇编文件以指定格式打包成.a
func dopack(dst, src string, extra []string) {
	bdst := bytes.NewBufferString(readfile(src))
	for _, file := range extra {
		b := readfile(file)
		// find last path element for archive member name
		i := strings.LastIndex(file, "/") + 1
		j := strings.LastIndex(file, `\`) + 1
		if i < j {
			i = j
		}

		fmt.Fprintf(bdst, "%-16.16s%-12d%-6d%-6d%-8o%-10d`\n", file[i:], 0, 0, 0, 0644, len(b))
		bdst.WriteString(b)
		if len(b)&1 != 0 {
			bdst.WriteByte(0)
		}
	}
	writefile(bdst.String(), dst, 0)
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

// cmdbootstrap - 构建go工具(关键go工具生成代码)
/*
 1.初始化目录相关信息
 2.构建工具类(复制文件内容到新模块，环境隔离)
 3.用工具类构建库包，主要是compile
 4.用工具类将link链接在一起
*/
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

	// 用于构建 Go tool工具链
	bootstrapBuildTools()

	goos = gohostos

	// For the main bootstrap, building for host os/arch.
	timelog("build", "go_bootstrap")
	xprintf("Building Go bootstrap cmd/go (go_bootstrap) using Go toolchain1.\n")
	// 安装src/cmd/go 这里会生成go_bootstrap二进制文件，go_bootstrap最终生成bin/go
	install("runtime") // link需要依赖于runtime
	install("cmd/go")  // go包编译

	//install("cmd/go")
	if vflag > 0 {
		xprintf("\n")
	}

	//goBootstrap := pathf("%s/go_bootstrap", tooldir)

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
	//goInstall(toolenv(), goBootstrap, "cmd")

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
