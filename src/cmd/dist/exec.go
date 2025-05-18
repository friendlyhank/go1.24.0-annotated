package main

import (
	"os/exec"
)

/*
 * 命令执行相关设置
 */

// setDir sets cmd.Dir to dir, and also adds PWD=dir to cmd's environment. 设置指令的路径
func setDir(cmd *exec.Cmd, dir string) {
	cmd.Dir = dir
}
