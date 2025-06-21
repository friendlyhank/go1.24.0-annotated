package base

import "os"

// ErrorExit - 错误停止处理
func ErrorExit() {
	if Flag.LowerO != "" {
		os.Remove(Flag.LowerO)
	}
	os.Exit(2)
}
