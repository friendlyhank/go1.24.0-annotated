package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// pathf is fmt.Sprintf for generating paths
// (on windows it turns / into \ after the printf). 路径信息
func pathf(format string, args ...interface{}) string {
	return filepath.Clean(fmt.Sprintf(format, args...))
}

// readfile returns the content of the named file. 读取文件
func readfile(file string) string {
	data, err := os.ReadFile(file)
	if err != nil {
		fatalf("%v", err)
	}
	return string(data)
}

// isfile reports whether p names an existing file.判断是否文件
func isfile(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "go tool dist: %s\n", fmt.Sprintf(format, args...))

	xexit(2)
}

// xexit - 停止程序
func xexit(n int) {
	os.Exit(n)
}

// xprintf prints a message to standard output.
func xprintf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}
