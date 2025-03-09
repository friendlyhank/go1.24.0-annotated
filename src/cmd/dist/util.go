package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// pathf is fmt.Sprintf for generating paths
// (on windows it turns / into \ after the printf).
func pathf(format string, args ...interface{}) string {
	return filepath.Clean(fmt.Sprintf(format, args...))
}

// filter returns a slice containing the elements x from list for which f(x) == true.
//
//	过滤切片中需要的元素
func filter(list []string, f func(string) bool) []string {
	var out []string
	for _, x := range list {
		if f(x) {
			out = append(out, x)
		}
	}
	return out
}

const (
	CheckExit  = 1 << iota // 异常停止并调用fatalf
	ShowOutput             // 显示命令的输出
	Background             // 运行在后台
)

var outputLock sync.Mutex

// run is like runEnv with no additional environment.执行命令不带指令
func run(dir string, mode int, cmd ...string) string {
	return runEnv(dir, mode, nil, cmd...)
}

// runEnv runs the command line cmd in dir with additional environment env. runEnv 在 dir 目录中使用额外的环境变量 env 执行命令行 cmd。
// If mode has ShowOutput set and Background unset, run passes cmd's output to 如果 mode 设置了 ShowOutput 且未设置 Background，则 run 将 cmd 的输出直接传递给 stdout/stderr。
// stdout/stderr directly. Otherwise, run returns cmd's output as a string. 否则，run 返回 cmd 的输出作为字符串。
// If mode has CheckExit set and the command fails, run calls fatalf. 如果 mode 设置了 CheckExit 且命令失败，则 run 调用 fatalf。
// If mode has Background set, this command is being run as a 如果 mode 设置了 Background，则该命令作为后台任务运行。
// Background job. Only bgrun should use the Background mode, 仅 bgrun 应该使用 Background 模式，其他调用者不应使用。
// not other callers.
func runEnv(dir string, mode int, env []string, cmd ...string) string {
	xcmd := exec.Command(cmd[0], cmd[1:]...)

	var data []byte
	var err error

	// If we want to show command output and this is not
	// a background command, assume it's the only thing
	// running, so we can just let it write directly stdout/stderr
	// as it runs without fear of mixing the output with some
	// other command's output. Not buffering lets the output
	// appear as it is printed instead of once the command exits.
	// This is most important for the invocation of 'go build -v bootstrap/...'.
	// 如果 mode 设置了 ShowOutput 且未设置 Background，则将命令的输出直接传递给 stdout/stderr
	if mode&(Background|ShowOutput) == ShowOutput {
		xcmd.Stdout = os.Stdout
		xcmd.Stderr = os.Stderr
		err = xcmd.Run()
	} else {
		// 否则，捕获命令的输出和错误信息
		data, err = xcmd.CombinedOutput()
	}
	if err != nil && mode&CheckExit != 0 {
		outputLock.Lock()
		if len(data) > 0 {
			xprintf("%s\n", data)
		}
		outputLock.Unlock()
		fatalf("FAILED: %v: %v", strings.Join(cmd, " "), err)
	}
	if mode&ShowOutput != 0 {
		outputLock.Lock()
		os.Stdout.Write(data)
		outputLock.Unlock()
	}
	return string(data)
}

var maxbg = 4 /* maximum number of jobs to run at once */

var (
	bgwork = make(chan func(), 1e5) // 后台工作异步进程

	bghelpers sync.WaitGroup

	dying = make(chan struct{}) // 正在进行状态
)

// bginit - 后台进程初始化
func bginit() {
	bghelpers.Add(maxbg)
	for i := 0; i < maxbg; i++ {
		go bghelper()
	}
}

func bghelper() {
	defer bghelpers.Done()
	for {
		select {
		case <-dying:
			return
		case w := <-bgwork:
			// Dying takes precedence over doing more work.
			select {
			case <-dying:
				return
			default:
				w()
			}
		}
	}
}

// bgrun is like run but runs the command in the background.
// CheckExit|ShowOutput mode is implied (since output cannot be returned).
// bgrun adds 1 to wg immediately, and calls Done when the work completes.
// 异步执行指令
func bgrun(wg *sync.WaitGroup, dir string, cmd ...string) {
	wg.Add(1)
	bgwork <- func() {
		defer wg.Done()
		run(dir, CheckExit|ShowOutput|Background, cmd...)
	}
}

// bgwait waits for pending bgruns to finish.
// bgwait must be called from only a single goroutine at a time.
// 等待后台程序执行完成
func bgwait(wg *sync.WaitGroup) {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-dying:
		// Don't return to the caller, to avoid reporting additional errors
		// to the user.
		select {}
	}
}

// isdir reports whether p names an existing directory. 文件路径是否存在
func isdir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

// isfile reports whether p names an existing file. 判断目录文件是否存在
func isfile(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}

const (
	writeExec     = 1 << iota // 文件可写和可执行
	writeSkipSame             // 写入文件跳过相同的内容
)

// writefile writes text to the named file, creating it if needed. 写入内容到文件
// if exec is non-zero, marks the file as executable. 如果是writeExec则文件可写和可执行
// If the file already exists and has the expected content,如果是writeSkipSame，则跳过文件相同内容
// it is not rewritten, to avoid changing the time stamp.
func writefile(text, file string, flag int) {
	new := []byte(text)
	if flag&writeSkipSame != 0 {
		old, err := os.ReadFile(file)
		if err == nil && bytes.Equal(old, new) {
			return
		}
	}
	mode := os.FileMode(0666)
	if flag&writeExec != 0 {
		mode = 0777
	}
	xremove(file) // in case of symlink tricks by misc/reboot test 先移除文件
	err := os.WriteFile(file, new, mode)
	if err != nil {
		fatalf("%v", err)
	}
}

// mtime returns the modification time of the file p. 获取文件下修改时间
func mtime(p string) time.Time {
	fi, err := os.Stat(p)
	if err != nil {
		return time.Time{}
	}
	return fi.ModTime()
}

// readfile returns the content of the named file.读取文件内容
func readfile(file string) string {
	data, err := os.ReadFile(file)
	if err != nil {
		fatalf("%v", err)
	}
	return string(data)
}

// xmkdir creates the directory p. 创建目录
func xmkdir(p string) {
	err := os.Mkdir(p, 0777)
	if err != nil {
		fatalf("%v", err)
	}
}

// xmkdirall creates the directory p and its parents, as needed. 递归创建目录
func xmkdirall(p string) {
	err := os.MkdirAll(p, 0777)
	if err != nil {
		fatalf("%v", err)
	}
}

// xremove removes the file p. 删除指定文件路径
func xremove(p string) {
	os.Remove(p)
}

// xremoveall removes the file or directory tree rooted at p.递归删除位于 p 的文件或目录树。
func xremoveall(p string) {
	os.RemoveAll(p)
}

// xreaddir replaces dst with a list of the names of the files and subdirectories in dir.
// The names are relative to dir; they are not full paths.
// xreaddir 读取指定目录中的文件和子目录的名称。
// 返回的名称是相对于给定目录的，不包含完整路径。
func xreaddir(dir string) []string {
	f, err := os.Open(dir)
	if err != nil {
		fatalf("%v", err)
	}
	defer f.Close()
	names, err := f.Readdirnames(-1)
	if err != nil {
		fatalf("reading %s: %v", dir, err)
	}
	return names
}

// xworkdir creates a new temporary directory to hold object files 创建临时的工作目录
// and returns the name of that directory.
func xworkdir() string {
	name, err := os.MkdirTemp(os.Getenv("GOTMPDIR"), "go-tool-dist-")
	if err != nil {
		fatalf("%v", err)
	}
	return name
}

// fatalf prints an error message to standard error and exits. 标准的输出错误并停止程序
func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "go tool dist: %s\n", fmt.Sprintf(format, args...))

	xexit(2)
}

var atexits []func() // 记录程序停止需要执行的方法

// xexit exits the process with return code n. 停止程序，返回错误码
func xexit(n int) {
	// 程序停止，执行对应要执行的方法
	for i := len(atexits) - 1; i >= 0; i-- {
		atexits[i]()
	}
	os.Exit(n)
}

// xatexit schedules the exit-handler f to be run when the program exits.
// xatexit 调度退出处理程序 f 以在程序退出时运行。
func xatexit(f func()) {
	atexits = append(atexits, f)
}

// xprintf prints a message to standard output. 标准打印输出
func xprintf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// errprintf prints a message to standard output. 标准打印输出
func errprintf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}

// count is a flag.Value that is like a flag.Bool and a flag.Int.
// If used as -name, it increments the count, but -name=x sets the count.
// Used for verbose flag -v.
type count int

func (c *count) String() string {
	return fmt.Sprint(int(*c))
}

func (c *count) Set(s string) error {
	switch s {
	case "true":
		*c++
	case "false":
		*c = 0
	default:
		n, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("invalid count %q", s)
		}
		*c = count(n)
	}
	return nil
}

func (c *count) IsBoolFlag() bool {
	return true
}

func xflagparse(maxargs int) {
	flag.Var((*count)(&vflag), "v", "verbosity")
	flag.Parse()
	if maxargs >= 0 && flag.NArg() > maxargs {
		flag.Usage()
	}
}
