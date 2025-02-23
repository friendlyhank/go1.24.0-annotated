package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// pathf is fmt.Sprintf for generating paths
// (on windows it turns / into \ after the printf).
func pathf(format string, args ...interface{}) string {
	return filepath.Clean(fmt.Sprintf(format, args...))
}

// isdir reports whether p names an existing directory. 文件路径是否存在
func isdir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

// xmkdir creates the directory p. 创建目录
func xmkdir(p string) {
	err := os.Mkdir(p, 0777)
	if err != nil {
		fatalf("%v", err)
	}
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "go tool dist: %s\n", fmt.Sprintf(format, args...))

	xexit(2)
}

func xexit(n int) {
	os.Exit(n)
}
