// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base

import (
	"flag"
	"fmt"
	"os"
)

/*
 * compile 所有指令参数在这
 */

func usage() {
	fmt.Fprintf(os.Stderr, "usage: compile [options] file.go...\n")
}

// Flag holds the parsed command-line flags.
// See ParseFlag for non-zero defaults.
var Flag CmdFlags

// CmdFlags defines the command-line flags (see var Flag).
// Each struct field is a different flag, by default named for the lower-case of the field name.
// If the flag name is a single letter, the default flag name is left upper-case.
// If the flag name is "Lower" followed by a single letter, the default flag name is the lower-case of the last letter.
//
// If this default flag name can't be made right, the `flag` struct tag can be used to replace it,
// but this should be done only in exceptional circumstances: it helps everyone if the flag name
// is obvious from the field name when the flag is used elsewhere in the compiler sources.
// The `flag:"-"` struct tag makes a field invisible to the flag logic and should also be used sparingly.
//
// Each field must have a `help` struct tag giving the flag help message.
//
// The allowed field types are bool, int, string, pointers to those (for values stored elsewhere),
// CountFlag (for a counting flag), and func(string) (for a flag that uses special code for parsing).
// compile指令参数
type CmdFlags struct {
	LowerO string "help:\"write output to `file`\"" // 输出的文件

	Pack bool "help:\"write to file.a instead of file.o\""
	Std  bool "help:\"compiling standard library\"" // 编译标准库
}

// ParseFlags parses the command-line flags into Flag. 解析命令参数
func ParseFlags() {
	if flag.NArg() < 1 {
		usage()
	}

	if Flag.LowerO == "" {
		p := flag.Arg(0)
		fmt.Println("=======", p)
	}
}
