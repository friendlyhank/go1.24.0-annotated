package runtime

import (
	"internal/goarch"
	"internal/goos"
)

// GOOS is the running program's operating system target:
// one of darwin, freebsd, linux, and so on.
// To view possible combinations of GOOS and GOARCH, run "go tool dist list".
// 表示运行程序的目标操作系统，包含darwin、freebsd、linux 等等。可以执行go tool dist list 查看所有组合
const GOOS string = goos.GOOS

// GOARCH is the running program's architecture target:
// one of 386, amd64, arm, s390x, and so on.
const GOARCH string = goarch.GOARCH
