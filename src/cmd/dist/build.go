package main

import (
	"fmt"
	"os"
)

var (
	gohostarch string // 主机架构如amd64、arm64
	gohostos   string // 操作系统 如linux、"darwin" (macOS), "windows"
	goroot     string // go路径 如local/usr/go

	rebuildall bool // 重新构建所有依赖
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
	fmt.Println(goosGoarch)
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
}
