// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package flag implements command-line flag parsing.

# Usage

Define flags using [flag.String], [Bool], [Int], etc.

This declares an integer flag, -n, stored in the pointer nFlag, with type *int:

	import "flag"
	var nFlag = flag.Int("n", 1234, "help message for flag n")

If you like, you can bind the flag to a variable using the Var() functions.

	var flagvar int
	func init() {
		flag.IntVar(&flagvar, "flagname", 1234, "help message for flagname")
	}

Or you can create custom flags that satisfy the Value interface (with
pointer receivers) and couple them to flag parsing by

	flag.Var(&flagVal, "name", "help message for flagname")

For such flags, the default value is just the initial value of the variable.

After all flags are defined, call

	flag.Parse()

to parse the command line into the defined flags.

Flags may then be used directly. If you're using the flags themselves,
they are all pointers; if you bind to variables, they're values.

	fmt.Println("ip has value ", *ip)
	fmt.Println("flagvar has value ", flagvar)

After parsing, the arguments following the flags are available as the
slice [flag.Args] or individually as [flag.Arg](i).
The arguments are indexed from 0 through [flag.NArg]-1.

# Command line flag syntax

The following forms are permitted:

	-flag
	--flag   // double dashes are also permitted
	-flag=x
	-flag x  // non-boolean flags only

One or two dashes may be used; they are equivalent.
The last form is not permitted for boolean flags because the
meaning of the command

	cmd -x *

where * is a Unix shell wildcard, will change if there is a file
called 0, false, etc. You must use the -flag=false form to turn
off a boolean flag.

Flag parsing stops just before the first non-flag argument
("-" is a non-flag argument) or after the terminator "--".

Integer flags accept 1234, 0664, 0x1234 and may be negative.
Boolean flags may be:

	1, 0, t, f, T, F, true, false, TRUE, FALSE, True, False

Duration flags accept any input valid for time.ParseDuration.

The default set of command-line flags is controlled by
top-level functions.  The [FlagSet] type allows one to define
independent sets of flags, such as to implement subcommands
in a command-line interface. The methods of [FlagSet] are
analogous to the top-level functions for the command-line
flag set.
*/

/*
 * flag命令解析处理
 */

package flag

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
)

// ErrHelp is the error returned if the -help or -h flag is invoked
// but no such flag is defined. 帮助请求
var ErrHelp = errors.New("flag: help requested")

// errParse is returned by Set if a flag's value fails to parse, such as with an invalid integer for Int.
// It then gets wrapped through failf to provide more information.
// 解析错误
var errParse = errors.New("parse error")

type boolValue bool

func newBoolValue(val bool, p *bool) *boolValue {
	*p = val
	return (*boolValue)(p)
}

func (b *boolValue) Set(s string) error {
	v, err := strconv.ParseBool(s)
	if err != nil {
		err = errParse
	}
	*b = boolValue(v)
	return err
}

func (b *boolValue) Get() any { return bool(*b) }

func (b *boolValue) String() string { return strconv.FormatBool(bool(*b)) }

func (b *boolValue) IsBoolFlag() bool { return true }

// optional interface to indicate boolean flags that can be
// supplied without "=value" text
type boolFlag interface {
	Value
	IsBoolFlag() bool
}

// Value is the interface to the dynamic value stored in a flag.
// (The default value is represented as a string.)
//
// If a Value has an IsBoolFlag() bool method returning true,
// the command-line parser makes -name equivalent to -name=true
// rather than using the next command-line argument.
//
// Set is called once, in command line order, for each flag present.
// The flag package may call the [String] method with a zero-valued receiver,
// such as a nil pointer.
type Value interface {
	String() string
	Set(string) error
}

// ErrorHandling defines how [FlagSet.Parse] behaves if the parse fails.
type ErrorHandling int

// These constants cause [FlagSet.Parse] to behave as described if the parse fails.
const (
	ContinueOnError ErrorHandling = iota // 返回错误但继续执行 Return a descriptive error.
	ExitOnError                          // 错误然后终止程序 Call os.Exit(2) or for -h/-help Exit(0).
	PanicOnError                         // 错误会抛出异常 Call panic with a descriptive error.
)

// A FlagSet represents a set of defined flags. The zero value of a FlagSet
// has no name and has [ContinueOnError] error handling.
//
// [Flag] names must be unique within a FlagSet. An attempt to define a flag whose
// name is already in use will cause a panic.
type FlagSet struct {
	// Usage is the function called when an error occurs while parsing flags.
	// The field is a function (not a method) that may be changed to point to
	// a custom error handler. What happens after Usage is called depends
	// on the ErrorHandling setting; for the command line, this defaults
	// to ExitOnError, which exits the program after calling Usage.
	Usage func() // 用于当参数解析错误的自定义错误处理方式(标记使用方法)

	name          string           // 命令行第一个参数，即是可执行程序名字
	parsed        bool             // 是否已经解析
	actual        map[string]*Flag // 存储实际运行解析的参数
	formal        map[string]*Flag // 存储所有预定义的参数
	args          []string         // arguments after flags，所有的参数，从os.args获取
	errorHandling ErrorHandling    // 错误处理方式
	output        io.Writer        // nil means stderr; use Output() accessor
}

// A Flag represents the state of a flag.
type Flag struct {
	Name     string // 命令行参数名称 name as it appears on command line
	Usage    string // help message 帮助信息
	Value    Value  // 值设置 value as set
	DefValue string // 默认值 default value (as text); for usage message
}

// sortFlags returns the flags as a slice in lexicographical sorted order.
func sortFlags(flags map[string]*Flag) []*Flag {
	result := make([]*Flag, len(flags))
	i := 0
	for _, f := range flags {
		result[i] = f
		i++
	}
	slices.SortFunc(result, func(a, b *Flag) int {
		return strings.Compare(a.Name, b.Name)
	})
	return result
}

// Output returns the destination for usage and error messages. [os.Stderr] is returned if
// output was not set or was set to nil.
func (f *FlagSet) Output() io.Writer {
	if f.output == nil {
		return os.Stderr
	}
	return f.output
}

// Name returns the name of the flag set.
func (f *FlagSet) Name() string {
	return f.name
}

// ErrorHandling returns the error handling behavior of the flag set.
func (f *FlagSet) ErrorHandling() ErrorHandling {
	return f.errorHandling
}

// SetOutput sets the destination for usage and error messages.
// If output is nil, [os.Stderr] is used.
func (f *FlagSet) SetOutput(output io.Writer) {
	f.output = output
}

// VisitAll visits the flags in lexicographical order, calling fn for each.
// It visits all flags, even those not set.
// VisitAll - 遍历所有命令参数
func (f *FlagSet) VisitAll(fn func(*Flag)) {
	for _, flag := range sortFlags(f.formal) {
		fn(flag)
	}
}

// PrintDefaults prints, to standard error unless configured otherwise, the
// default values of all defined command-line flags in the set. See the
// documentation for the global function PrintDefaults for more information.
// 打印默认已定义的命令行参数
func (f *FlagSet) PrintDefaults() {

}

func (f *FlagSet) defaultUsage() {
	if f.name == "" {
		fmt.Fprintf(f.Output(), "Usage:\n")
	} else {
		fmt.Fprintf(f.Output(), "Usage of %s:\n", f.name)
	}
	f.PrintDefaults()
}

// todo hank 实现
func (f *FlagSet) Var(value Value, name string, usage string) {

}

// Var defines a flag with the specified name and usage string. The type and
// value of the flag are represented by the first argument, of type [Value], which
// typically holds a user-defined implementation of [Value]. For instance, the
// caller could create a flag that turns a comma-separated string into a slice
// of strings by giving the slice the methods of [Value]; in particular, [Set] would
// decompose the comma-separated string into the slice.
func Var(value Value, name string, usage string) {
	CommandLine.Var(value, name, usage)
}

// sprintf formats the message, prints it to output, and returns it.
func (f *FlagSet) sprintf(format string, a ...any) string {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintln(f.Output(), msg)
	return msg
}

// failf prints to standard error a formatted error and usage message and
// returns the error.
func (f *FlagSet) failf(format string, a ...any) error {
	msg := f.sprintf(format, a...)
	f.usage()
	return errors.New(msg)
}

// usage calls the Usage method for the flag set if one is specified,
// or the appropriate default usage function otherwise.
func (f *FlagSet) usage() {
	if f.Usage == nil {
		f.defaultUsage()
	} else {
		f.Usage()
	}
}

// parseOne parses one flag. It reports whether a flag was seen. 解析单个参数
func (f *FlagSet) parseOne() (bool, error) {
	// 检查是否还有参数需要解析
	if len(f.args) == 0 {
		return false, nil
	}
	s := f.args[0]
	// 如果不是-开头或参数长度小于2，则返回不再遍历
	if len(s) < 2 || s[0] != '-' {
		return false, nil
	}
	// 判断参数是几个-
	numMinuses := 1
	if s[1] == '-' {
		numMinuses++
		if len(s) == 2 { // "--" terminates the flags
			f.args = f.args[1:]
			return false, nil
		}
	}
	name := s[numMinuses:]
	if len(name) == 0 || name[0] == '-' || name[0] == '=' {
		return false, f.failf("bad flag syntax: %s", s)
	}

	// it's a flag. does it have an argument? 移除已解析的参数，设置为下一个
	f.args = f.args[1:]
	hasValue := false // 是否有值
	value := ""
	// 以=为标志，左边部分为参数名称，右边则为参数值
	for i := 1; i < len(name); i++ { // equals cannot be first
		if name[i] == '=' {
			value = name[i+1:]
			hasValue = true
			name = name[0:i]
			break
		}
	}

	// 查询在预定义中有没这个命令
	flag, ok := f.formal[name]
	if !ok {
		if name == "help" || name == "h" { // special case for nice help message.
			f.usage()
			return false, ErrHelp
		}
		return false, f.failf("flag provided but not defined: -%s", name)
	}

	if fv, ok := flag.Value.(boolFlag); ok && fv.IsBoolFlag() { // special case: doesn't need an arg
		// 显示带值方式 -flag=value
		// 隐示无值方式 -flag隐式设为 true
		if hasValue {
			if err := fv.Set(value); err != nil {
				return false, f.failf("invalid boolean value %q for -%s: %v", value, name, err)
			}
		} else {
			if err := fv.Set("true"); err != nil {
				return false, f.failf("invalid boolean flag %s: %v", name, err)
			}
		}
	} else {
		// It must have a value, which might be the next argument.
		// 布尔标志必须有值，优先从当前参数提取，其次从后续参数获取 （如 -input file.txt)
		if !hasValue && len(f.args) > 0 {
			// value is the next arg
			hasValue = true
			value, f.args = f.args[0], f.args[1:] // 从后面参数中获取值并将参数设置为下一个
		}
		if !hasValue {
			return false, f.failf("flag needs an argument: -%s", name)
		}
		if err := flag.Value.Set(value); err != nil {
			return false, f.failf("invalid value %q for flag -%s: %v", value, name, err)
		}
	}

	if f.actual == nil {
		f.actual = make(map[string]*Flag)
	}
	// 存储实际解析到的值
	f.actual[name] = flag
	return true, nil
}

// Parse parses flag definitions from the argument list, which should not
// include the command name. Must be called after all flags in the [FlagSet]
// are defined and before flags are accessed by the program.
// The return value will be [ErrHelp] if -help or -h were set but not defined.
func (f *FlagSet) Parse(arguments []string) error {
	f.parsed = true
	f.args = arguments
	for {
		// seen表示是否需要循环解析
		seen, err := f.parseOne()
		if seen {
			continue
		}
		// 到这里说明错误或没有参数要解析了
		if err == nil {
			break
		}
		switch f.errorHandling {
		case ContinueOnError:
			return err
		case ExitOnError:
			os.Exit(2)
		case PanicOnError:
		}
	}
	return nil
}

// Parse parses the command-line flags from [os.Args][1:]. Must be called
// after all flags are defined and before flags are accessed by the program.
// 解析命令参数
func Parse() {
	// Ignore errors; CommandLine is set for ExitOnError.
	CommandLine.Parse(os.Args[1:])
}

// CommandLine is the default set of command-line flags, parsed from [os.Args].
// The top-level functions such as [BoolVar], [Arg], and so on are wrappers for the
// methods of CommandLine.
var CommandLine *FlagSet

func init() {
	// It's possible for execl to hand us an empty os.Args.
	if len(os.Args) == 0 {
		CommandLine = NewFlagSet("", ExitOnError)
	} else {
		CommandLine = NewFlagSet(os.Args[0], ExitOnError)
	}
}

func NewFlagSet(name string, errorHandling ErrorHandling) *FlagSet {
	f := &FlagSet{
		name:          name,
		errorHandling: errorHandling,
	}
	f.Usage = f.defaultUsage
	return f
}
