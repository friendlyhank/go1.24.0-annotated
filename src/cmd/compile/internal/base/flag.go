// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base

import (
	"cmd/internal/objabi"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"
)

/*
 * compile 所有指令参数在这
 */

func usage() {
	fmt.Fprintf(os.Stderr, "usage: compile [options] file.go...\n")
	Exit(2)
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
	LowerO string  "help:\"write output to `file`\""             // 输出的文件 LowerO会被处理成 -o
	LowerP *string "help:\"set expected package import `path`\"" // &Ctxt.Pkgpath, set below

	ImportCfg func(string) "help:\"read import configuration from `file`\"" // 读取导入的配置文件
	Pack      bool         "help:\"write to file.a instead of file.o\""     // 打包输出.a文件，而不是.o // todo hank .a和.o区别
	Std       bool         "help:\"compiling standard library\""            // 编译标准库
}

// ParseFlags parses the command-line flags into Flag. 解析命令参数
func ParseFlags() {

	Flag.LowerP = &Ctxt.Pkgpath

	Flag.ImportCfg = readImportCfg

	// 注册命令参数
	registerFlags()
	// 解析命令
	objabi.Flagparse(usage)

	Ctxt.Std = Flag.Std

	if flag.NArg() < 1 {
		usage()
	}
}

// registerFlags adds flag registrations for all the fields in Flag.
// See the comment on type CmdFlags for the rules.
// 通过反射解析设置参数
func registerFlags() {

	var (
		boolType      = reflect.TypeOf(bool(false))
		stringType    = reflect.TypeOf(string(""))
		ptrStringType = reflect.TypeOf(new(string))
		funcType      = reflect.TypeOf((func(string))(nil))
	)

	v := reflect.ValueOf(&Flag).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		var name string
		if len(f.Name) == 1 {
			name = f.Name
		} else if len(f.Name) == 6 && f.Name[:5] == "Lower" && 'A' <= f.Name[5] && f.Name[5] <= 'Z' {
			name = string(rune(f.Name[5] + 'a' - 'A'))
		} else {
			name = strings.ToLower(f.Name)
		}

		help := f.Tag.Get("help")
		if help == "" {
			panic(fmt.Sprintf("base.Flag.%s is missing help text", f.Name))
		}

		if k := f.Type.Kind(); (k == reflect.Ptr || k == reflect.Func) && v.Field(i).IsNil() {
			panic(fmt.Sprintf("base.Flag.%s is uninitialized %v", f.Name, f.Type))
		}

		switch f.Type {
		case boolType:
			p := v.Field(i).Addr().Interface().(*bool)
			flag.BoolVar(p, name, *p, help)
		case stringType:
			p := v.Field(i).Addr().Interface().(*string)
			flag.StringVar(p, name, *p, help)
		case ptrStringType:
			p := v.Field(i).Interface().(*string)
			flag.StringVar(p, name, *p, help)
		case funcType:
			f := v.Field(i).Interface().(func(string))
			objabi.Flagfn1(name, help, f)
		}
	}
}

// readImportCfg - 读取需要导入的文件配置 对应指令 -importcfg
func readImportCfg(file string) {

}
