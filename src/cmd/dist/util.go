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
)

// pathf is fmt.Sprintf for generating paths
// (on windows it turns / into \ after the printf).
func pathf(format string, args ...interface{}) string {
	return filepath.Clean(fmt.Sprintf(format, args...))
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

// fatalf prints an error message to standard error and exits. 标准的输出错误并停止程序
func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "go tool dist: %s\n", fmt.Sprintf(format, args...))

	xexit(2)
}

// xexit exits the process with return code n. 停止程序，返回错误码
func xexit(n int) {
	os.Exit(n)
}

// xprintf prints a message to standard output. 标准打印输出
func xprintf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
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
