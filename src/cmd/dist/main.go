package main

import (
	"os"
	"runtime"
)

func usage() {
}

// commands records the available commands. 工具链指令
var commands = map[string]func(){
	"bootstrap": cmdbootstrap, // 构建go命令
}

func main() {

	gohostos = runtime.GOOS

	// 初始化方法
	xinit()
	// 对应指令执行
	xmain()
}

// The OS-specific main calls into the portable code here.
func xmain() {
	cmd := os.Args[1]
	os.Args = os.Args[1:] // for flag parsing during cmd
	if f, ok := commands[cmd]; ok {
		f()
	}
}
