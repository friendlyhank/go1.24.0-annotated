package main

import "os"

func usage() {
	xprintf(`usage: go tool dist [command]
Commands are:

banner                  print installation banner
bootstrap               rebuild everything
clean                   deletes all built files
env [-p]                print environment (-p: include $PATH)
install [dir]           install individual directory
list [-json] [-broken]  list all supported platforms
test [-h]               run Go test(s)
version                 print Go version

All commands take -v flags to emit extra information.
`)
	xexit(2)
}

// commands records the available commands. 工具链指令
var commands = map[string]func(){
	"version": cmdversion,
}

func main() {
	// 初始化方法
	xinit()
	// 对应指令执行
	xmain()
}

// The OS-specific main calls into the portable code here.
func xmain() {
	if len(os.Args) < 2 {
		usage()
	}

	cmd := os.Args[1]

	if f, ok := commands[cmd]; ok {
		f()
	} else {
		xprintf("unknown command %s\n", cmd)
		usage()
	}
}
